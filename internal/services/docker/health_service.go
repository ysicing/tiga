package docker

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

const (
	// healthCheckConcurrency is the number of concurrent health checks
	healthCheckConcurrency = 10
)

// DockerHealthService handles Docker instance health checking
type DockerHealthService struct {
	repo           repository.DockerInstanceRepositoryInterface
	agentForwarder *AgentForwarderV2
	db             *gorm.DB // For checking agent connection status
}

// NewDockerHealthService creates a new DockerHealthService
func NewDockerHealthService(
	repo repository.DockerInstanceRepositoryInterface,
	agentForwarder *AgentForwarderV2,
	db *gorm.DB,
) *DockerHealthService {
	return &DockerHealthService{
		repo:           repo,
		agentForwarder: agentForwarder,
		db:             db,
	}
}

// CheckInstance checks the health of a single Docker instance
func (s *DockerHealthService) CheckInstance(ctx context.Context, instanceID uuid.UUID) error {
	// Get instance
	instance, err := s.repo.GetByID(ctx, instanceID)
	if err != nil {
		return err
	}

	// Skip archived instances
	if instance.IsArchived() {
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"instance_id":   instanceID,
		"instance_name": instance.Name,
		"agent_id":      instance.AgentID,
	}).Debug("Checking Docker instance health")

	// T032: For integrated architecture (Docker instances from agent auto-discovery),
	// check agent connection status instead of calling AgentForwarder
	var agentConn models.AgentConnection
	err = s.db.Where("id = ?", instance.AgentID).First(&agentConn).Error
	if err == nil {
		// Agent connection record found, check if agent is online
		if agentConn.Status == models.AgentStatusOnline {
			// Agent is online, mark Docker instance as online
			if err := s.repo.MarkOnline(ctx, instanceID); err != nil {
				logrus.WithError(err).Error("Failed to mark instance as online")
			}
			logrus.WithFields(logrus.Fields{
				"instance_id":   instanceID,
				"instance_name": instance.Name,
			}).Debug("Docker instance health check succeeded (agent online)")
			return nil
		} else {
			// Agent is offline, mark Docker instance as offline
			if err := s.repo.MarkOffline(ctx, instanceID); err != nil {
				logrus.WithError(err).Error("Failed to mark instance as offline")
			}
			logrus.WithFields(logrus.Fields{
				"instance_id":   instanceID,
				"instance_name": instance.Name,
			}).Debug("Docker instance marked offline (agent offline)")
			return nil
		}
	}

	// Fall back to old logic for standalone Docker Agent architecture
	// Call GetDockerInfo via agent forwarder
	info, err := s.agentForwarder.GetDockerInfo(instance.AgentID)
	if err != nil {
		// Health check failed, mark as offline
		logrus.WithFields(logrus.Fields{
			"instance_id":   instanceID,
			"instance_name": instance.Name,
			"error":         err.Error(),
		}).Warn("Docker instance health check failed")

		if markErr := s.repo.MarkOffline(ctx, instanceID); markErr != nil {
			logrus.WithError(markErr).Error("Failed to mark instance as offline")
		}
		return err
	}

	// Health check succeeded, update status and statistics
	containerCount := int(info.Info.Containers)
	imageCount := int(info.Info.Images)
	volumeCount := 0  // TODO: Add volume count when available
	networkCount := 0 // TODO: Add network count when available

	if err := s.repo.UpdateHealthStatus(
		ctx,
		instanceID,
		"online",
		containerCount,
		imageCount,
		volumeCount,
		networkCount,
	); err != nil {
		logrus.WithError(err).Error("Failed to update instance health status")
		return err
	}

	// Also update Docker daemon information
	dockerInfo := map[string]interface{}{
		"docker_version":   info.Info.Driver,           // Using driver as version field
		"api_version":      "",                         // Not available in SystemInfo
		"min_api_version":  "",                         // Not available in SystemInfo
		"storage_driver":   info.Info.Driver,
		"operating_system": info.Info.OperatingSystem,
		"architecture":     info.Info.Architecture,
		"kernel_version":   info.Info.KernelVersion,
		"mem_total":        info.Info.MemTotal,
		"n_cpu":            int(info.Info.Ncpu),
	}

	if err := s.repo.UpdateDockerInfo(ctx, instanceID, dockerInfo); err != nil {
		logrus.WithError(err).Error("Failed to update Docker info")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"instance_id":    instanceID,
		"instance_name":  instance.Name,
		"containers":     containerCount,
		"images":         imageCount,
		"docker_version": info.Info.Driver,  // Using driver as version
	}).Debug("Docker instance health check succeeded")

	return nil
}

// CheckAllInstances checks health of all non-archived instances with controlled concurrency
func (s *DockerHealthService) CheckAllInstances(ctx context.Context) error {
	start := time.Now()

	// Get all non-archived instances
	instances, _, err := s.repo.ListInstances(ctx, &repository.DockerInstanceFilter{
		PageSize: 1000, // Large page size to get all instances
		Page:     1,
	})
	if err != nil {
		return err
	}

	// Filter out archived instances
	var activeInstances []uuid.UUID
	for _, instance := range instances {
		if !instance.IsArchived() {
			activeInstances = append(activeInstances, instance.ID)
		}
	}

	if len(activeInstances) == 0 {
		logrus.Info("No active Docker instances to check")
		return nil
	}

	logrus.WithField("count", len(activeInstances)).Info("Starting health check for all Docker instances")

	// Use worker pool pattern for controlled concurrency
	var wg sync.WaitGroup
	instanceChan := make(chan uuid.UUID, len(activeInstances))
	resultChan := make(chan error, len(activeInstances))

	// Start workers
	for i := 0; i < healthCheckConcurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for instanceID := range instanceChan {
				if err := s.CheckInstance(ctx, instanceID); err != nil {
					logrus.WithFields(logrus.Fields{
						"worker_id":   workerID,
						"instance_id": instanceID,
						"error":       err.Error(),
					}).Debug("Health check failed for instance")
					resultChan <- err
				} else {
					resultChan <- nil
				}
			}
		}(i)
	}

	// Send instances to workers
	for _, instanceID := range activeInstances {
		instanceChan <- instanceID
	}
	close(instanceChan)

	// Wait for all workers to finish
	wg.Wait()
	close(resultChan)

	// Count successes and failures
	successCount := 0
	failureCount := 0
	for range activeInstances {
		err := <-resultChan
		if err == nil {
			successCount++
		} else {
			failureCount++
		}
	}

	duration := time.Since(start)

	logrus.WithFields(logrus.Fields{
		"total":    len(activeInstances),
		"success":  successCount,
		"failure":  failureCount,
		"duration": duration.String(),
		"workers":  healthCheckConcurrency,
	}).Info("Completed health check for all Docker instances")

	return nil
}

// CheckInstancesByAgentID checks health of all instances associated with an agent
func (s *DockerHealthService) CheckInstancesByAgentID(ctx context.Context, agentID uuid.UUID) error {
	// Get instance by agent ID
	instance, err := s.repo.GetByAgentID(ctx, agentID)
	if err != nil {
		return err
	}

	return s.CheckInstance(ctx, instance.ID)
}

// GetHealthSummary returns a summary of Docker instance health
func (s *DockerHealthService) GetHealthSummary(ctx context.Context) (*HealthSummary, error) {
	stats, err := s.repo.GetStatistics(ctx)
	if err != nil {
		return nil, err
	}

	// Calculate percentages
	total := float64(stats.Total)
	onlinePercent := 0.0
	offlinePercent := 0.0
	if total > 0 {
		onlinePercent = float64(stats.Online) / total * 100
		offlinePercent = float64(stats.Offline) / total * 100
	}

	return &HealthSummary{
		Total:          stats.Total,
		Online:         stats.Online,
		Offline:        stats.Offline,
		Archived:       stats.Archived,
		Unknown:        stats.Unknown,
		OnlinePercent:  onlinePercent,
		OfflinePercent: offlinePercent,
	}, nil
}

// HealthSummary contains health summary statistics
type HealthSummary struct {
	Total          int64   `json:"total"`
	Online         int64   `json:"online"`
	Offline        int64   `json:"offline"`
	Archived       int64   `json:"archived"`
	Unknown        int64   `json:"unknown"`
	OnlinePercent  float64 `json:"online_percent"`
	OfflinePercent float64 `json:"offline_percent"`
}
