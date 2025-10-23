package docker

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// DockerInstanceService handles business logic for Docker instances
type DockerInstanceService struct {
	repo           repository.DockerInstanceRepositoryInterface
	agentForwarder *AgentForwarder
	db             *gorm.DB
}

// NewDockerInstanceService creates a new DockerInstanceService
func NewDockerInstanceService(db *gorm.DB, agentForwarder *AgentForwarder) *DockerInstanceService {
	return &DockerInstanceService{
		repo:           repository.NewDockerInstanceRepository(db),
		agentForwarder: agentForwarder,
		db:             db,
	}
}

// Create creates a new Docker instance (manual registration)
func (s *DockerInstanceService) Create(ctx context.Context, instance *models.DockerInstance) error {
	// Validate required fields
	if instance.Name == "" {
		return fmt.Errorf("instance name is required")
	}
	if instance.AgentID == uuid.Nil {
		return fmt.Errorf("agent ID is required")
	}

	// Check if instance with same name already exists
	existing, err := s.repo.GetByName(ctx, instance.Name)
	if err == nil && existing != nil {
		return fmt.Errorf("instance with name '%s' already exists", instance.Name)
	}

	// Check if instance with same agent ID already exists
	existing, err = s.repo.GetByAgentID(ctx, instance.AgentID)
	if err == nil && existing != nil {
		return fmt.Errorf("instance with agent ID '%s' already exists", instance.AgentID)
	}

	// Set default values
	if instance.HealthStatus == "" {
		instance.HealthStatus = "unknown"
	}

	// Create instance
	if err := s.repo.Create(ctx, instance); err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"instance_id":   instance.ID,
		"instance_name": instance.Name,
		"agent_id":      instance.AgentID,
	}).Info("Docker instance created")

	return nil
}

// GetByID retrieves a Docker instance by ID
func (s *DockerInstanceService) GetByID(ctx context.Context, id uuid.UUID) (*models.DockerInstance, error) {
	instance, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("instance not found")
		}
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}
	return instance, nil
}

// GetByName retrieves a Docker instance by name
func (s *DockerInstanceService) GetByName(ctx context.Context, name string) (*models.DockerInstance, error) {
	instance, err := s.repo.GetByName(ctx, name)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("instance not found")
		}
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}
	return instance, nil
}

// GetByAgentID retrieves a Docker instance by agent ID
func (s *DockerInstanceService) GetByAgentID(ctx context.Context, agentID uuid.UUID) (*models.DockerInstance, error) {
	instance, err := s.repo.GetByAgentID(ctx, agentID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("instance not found")
		}
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}
	return instance, nil
}

// Update updates a Docker instance
func (s *DockerInstanceService) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	// Check if instance exists
	instance, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("instance not found")
		}
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Check if updating name and new name already exists
	if newName, ok := updates["name"].(string); ok && newName != instance.Name {
		existing, err := s.repo.GetByName(ctx, newName)
		if err == nil && existing != nil && existing.ID != id {
			return fmt.Errorf("instance with name '%s' already exists", newName)
		}
	}

	// Update fields
	if err := s.repo.UpdateFields(ctx, id, updates); err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"instance_id":   id,
		"instance_name": instance.Name,
		"updates":       updates,
	}).Info("Docker instance updated")

	return nil
}

// Delete deletes a Docker instance
func (s *DockerInstanceService) Delete(ctx context.Context, id uuid.UUID) error {
	// Check if instance exists
	instance, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("instance not found")
		}
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Delete instance
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"instance_id":   id,
		"instance_name": instance.Name,
	}).Info("Docker instance deleted")

	return nil
}

// ListInstances lists Docker instances with filtering and pagination
func (s *DockerInstanceService) ListInstances(ctx context.Context, filter *repository.DockerInstanceFilter) ([]*models.DockerInstance, int64, error) {
	instances, total, err := s.repo.ListInstances(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list instances: %w", err)
	}
	return instances, total, nil
}

// ListOnlineInstances lists all online Docker instances
func (s *DockerInstanceService) ListOnlineInstances(ctx context.Context) ([]*models.DockerInstance, error) {
	instances, err := s.repo.ListOnlineInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list online instances: %w", err)
	}
	return instances, nil
}

// SearchByName searches Docker instances by name
func (s *DockerInstanceService) SearchByName(ctx context.Context, name string) ([]*models.DockerInstance, error) {
	instances, err := s.repo.SearchByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to search instances: %w", err)
	}
	return instances, nil
}

// GetStatistics returns statistics about Docker instances
func (s *DockerInstanceService) GetStatistics(ctx context.Context) (*repository.DockerInstanceStatistics, error) {
	stats, err := s.repo.GetStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}
	return stats, nil
}

// TestConnection tests the connection to a Docker instance by calling GetDockerInfo
func (s *DockerInstanceService) TestConnection(ctx context.Context, id uuid.UUID) (bool, error) {
	// Get instance
	instance, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return false, fmt.Errorf("instance not found: %w", err)
	}

	// Try to call GetDockerInfo via agent forwarder
	_, err = s.agentForwarder.GetDockerInfo(instance.AgentID)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"instance_id": id,
			"agent_id":    instance.AgentID,
			"error":       err.Error(),
		}).Warn("Failed to connect to Docker instance")
		return false, nil
	}

	logrus.WithFields(logrus.Fields{
		"instance_id": id,
		"agent_id":    instance.AgentID,
	}).Info("Successfully connected to Docker instance")

	return true, nil
}

// AutoDiscoverOrUpdate automatically discovers and creates/updates Docker instance when Agent reports
// This is called when Agent connects and sends Docker info
func (s *DockerInstanceService) AutoDiscoverOrUpdate(ctx context.Context, agentID uuid.UUID, dockerInfo map[string]interface{}) (*models.DockerInstance, error) {
	// Try to find existing instance by agent ID
	instance, err := s.repo.GetByAgentID(ctx, agentID)

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to query instance: %w", err)
	}

	if err == gorm.ErrRecordNotFound {
		// Instance doesn't exist, create new one
		instance = &models.DockerInstance{
			Name:         fmt.Sprintf("docker-%s", agentID.String()[:8]), // Default name
			AgentID:      agentID,
			HealthStatus: "online",
		}

		// Populate Docker info
		s.populateDockerInfo(instance, dockerInfo)

		if err := s.repo.Create(ctx, instance); err != nil {
			return nil, fmt.Errorf("failed to create instance: %w", err)
		}

		logrus.WithFields(logrus.Fields{
			"instance_id":   instance.ID,
			"instance_name": instance.Name,
			"agent_id":      agentID,
		}).Info("Auto-discovered new Docker instance")

		return instance, nil
	}

	// Instance exists, update Docker info and mark online
	s.populateDockerInfo(instance, dockerInfo)
	instance.HealthStatus = "online"

	if err := s.repo.Update(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to update instance: %w", err)
	}

	// Mark online
	if err := s.repo.MarkOnline(ctx, instance.ID); err != nil {
		return nil, fmt.Errorf("failed to mark instance online: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"instance_id":   instance.ID,
		"instance_name": instance.Name,
		"agent_id":      agentID,
	}).Info("Updated Docker instance from auto-discovery")

	return instance, nil
}

// MarkOffline marks a Docker instance as offline
func (s *DockerInstanceService) MarkOffline(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.MarkOffline(ctx, id); err != nil {
		return fmt.Errorf("failed to mark instance offline: %w", err)
	}

	logrus.WithField("instance_id", id).Info("Marked Docker instance as offline")
	return nil
}

// MarkArchived marks a Docker instance as archived (when Agent is deleted)
func (s *DockerInstanceService) MarkArchived(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.MarkArchived(ctx, id); err != nil {
		return fmt.Errorf("failed to mark instance archived: %w", err)
	}

	logrus.WithField("instance_id", id).Info("Marked Docker instance as archived")
	return nil
}

// MarkArchivedByAgentID marks all Docker instances associated with an agent as archived
func (s *DockerInstanceService) MarkArchivedByAgentID(ctx context.Context, agentID uuid.UUID) error {
	// Get instance by agent ID
	instance, err := s.repo.GetByAgentID(ctx, agentID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// No instance found, nothing to archive
			return nil
		}
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Mark as archived
	if err := s.repo.MarkArchived(ctx, instance.ID); err != nil {
		return fmt.Errorf("failed to mark instance archived: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"instance_id": instance.ID,
		"agent_id":    agentID,
	}).Info("Marked Docker instance as archived (agent deleted)")

	return nil
}

// MarkOfflineByAgentID marks all Docker instances associated with an agent as offline
func (s *DockerInstanceService) MarkOfflineByAgentID(ctx context.Context, agentID uuid.UUID) error {
	if err := s.repo.MarkAllInstancesOfflineByAgentID(ctx, agentID); err != nil {
		return fmt.Errorf("failed to mark instances offline: %w", err)
	}

	logrus.WithField("agent_id", agentID).Info("Marked all Docker instances for agent as offline")
	return nil
}

// populateDockerInfo populates Docker instance fields from Docker info map
func (s *DockerInstanceService) populateDockerInfo(instance *models.DockerInstance, info map[string]interface{}) {
	if version, ok := info["version"].(string); ok {
		instance.DockerVersion = version
	}
	if apiVersion, ok := info["api_version"].(string); ok {
		instance.APIVersion = apiVersion
	}
	if minAPIVersion, ok := info["min_api_version"].(string); ok {
		instance.MinAPIVersion = minAPIVersion
	}
	if storageDriver, ok := info["storage_driver"].(string); ok {
		instance.StorageDriver = storageDriver
	}
	if os, ok := info["operating_system"].(string); ok {
		instance.OperatingSystem = os
	}
	if arch, ok := info["architecture"].(string); ok {
		instance.Architecture = arch
	}
	if kernelVersion, ok := info["kernel_version"].(string); ok {
		instance.KernelVersion = kernelVersion
	}
	if memTotal, ok := info["mem_total"].(int64); ok {
		instance.MemTotal = memTotal
	}
	if ncpu, ok := info["n_cpu"].(int32); ok {
		instance.NCPU = int(ncpu)
	}
	if containers, ok := info["containers"].(int32); ok {
		instance.ContainerCount = int(containers)
	}
	if images, ok := info["images"].(int32); ok {
		instance.ImageCount = int(images)
	}
}

// CreateManualInstance creates a Docker instance manually (not via auto-discovery)
func (s *DockerInstanceService) CreateManualInstance(
	ctx context.Context,
	name string,
	description string,
	agentID uuid.UUID,
	tags []string,
) (*models.DockerInstance, error) {
	instance := &models.DockerInstance{
		Name:         name,
		Description:  description,
		AgentID:      agentID,
		HealthStatus: "unknown",
		Tags:         tags,
	}

	if err := s.Create(ctx, instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// Archive archives a Docker instance (soft delete)
func (s *DockerInstanceService) Archive(ctx context.Context, id uuid.UUID) error {
	return s.MarkArchived(ctx, id)
}

// UpdateFromModel updates a Docker instance using the instance model
func (s *DockerInstanceService) UpdateFromModel(ctx context.Context, instance *models.DockerInstance) error {
	updates := map[string]interface{}{
		"name":        instance.Name,
		"description": instance.Description,
		"tags":        instance.Tags,
	}
	// Call the existing Update(ctx, id, updates) method
	if err := s.repo.UpdateFields(ctx, instance.ID, updates); err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"instance_id":   instance.ID,
		"instance_name": instance.Name,
	}).Info("Docker instance updated from model")

	return nil
}
