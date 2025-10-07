package managers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// ManagerCoordinator coordinates service managers
type ManagerCoordinator struct {
	factory           *ManagerFactory
	instanceRepo      *repository.InstanceRepository
	metricsRepo       *repository.MetricsRepository
	auditRepo         *repository.AuditLogRepository
	managers          map[uuid.UUID]ServiceManager
	managersMu        sync.RWMutex
	monitoringEnabled bool
	monitorInterval   time.Duration
	stopCh            chan struct{}
	wg                sync.WaitGroup
}

// NewManagerCoordinator creates a new manager coordinator
func NewManagerCoordinator(
	factory *ManagerFactory,
	instanceRepo *repository.InstanceRepository,
	metricsRepo *repository.MetricsRepository,
	auditRepo *repository.AuditLogRepository,
) *ManagerCoordinator {
	return &ManagerCoordinator{
		factory:           factory,
		instanceRepo:      instanceRepo,
		metricsRepo:       metricsRepo,
		auditRepo:         auditRepo,
		managers:          make(map[uuid.UUID]ServiceManager),
		monitoringEnabled: true,
		monitorInterval:   30 * time.Second,
		stopCh:            make(chan struct{}),
	}
}

// GetManager gets or creates a manager for an instance
func (c *ManagerCoordinator) GetManager(ctx context.Context, instanceID uuid.UUID) (ServiceManager, error) {
	// Check if manager already exists
	c.managersMu.RLock()
	manager, exists := c.managers[instanceID]
	c.managersMu.RUnlock()

	if exists {
		return manager, nil
	}

	// Get instance from repository
	instance, err := c.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// Create manager
	manager, err = c.factory.Create(instance.Type)
	if err != nil {
		return nil, err
	}

	// Initialize manager
	if err := manager.Initialize(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to initialize manager: %w", err)
	}

	// Connect manager
	if err := manager.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect manager: %w", err)
	}

	// Store manager
	c.managersMu.Lock()
	c.managers[instanceID] = manager
	c.managersMu.Unlock()

	logrus.Infof("Created and connected manager for instance %s (type: %s)", instanceID, instance.Type)

	return manager, nil
}

// RemoveManager removes a manager for an instance
func (c *ManagerCoordinator) RemoveManager(ctx context.Context, instanceID uuid.UUID) error {
	c.managersMu.Lock()
	defer c.managersMu.Unlock()

	manager, exists := c.managers[instanceID]
	if !exists {
		return nil
	}

	// Disconnect manager
	if err := manager.Disconnect(ctx); err != nil {
		logrus.Errorf("Failed to disconnect manager for instance %s: %v", instanceID, err)
	}

	delete(c.managers, instanceID)
	logrus.Infof("Removed manager for instance %s", instanceID)

	return nil
}

// HealthCheck performs health check on an instance
func (c *ManagerCoordinator) HealthCheck(ctx context.Context, instanceID uuid.UUID) (*HealthStatus, error) {
	manager, err := c.GetManager(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	status, err := manager.HealthCheck(ctx)
	if err != nil {
		return nil, err
	}

	// Update instance health in database
	healthStr := "unknown"
	if status.Healthy {
		healthStr = "healthy"
	} else {
		healthStr = "unhealthy"
	}

	var healthMessage *string
	if status.Message != "" {
		healthMessage = &status.Message
	}

	if err := c.instanceRepo.UpdateHealth(ctx, instanceID, healthStr, healthMessage); err != nil {
		logrus.Errorf("Failed to update instance health: %v", err)
	}

	return status, nil
}

// CollectMetrics collects metrics from an instance
func (c *ManagerCoordinator) CollectMetrics(ctx context.Context, instanceID uuid.UUID) (*ServiceMetrics, error) {
	manager, err := c.GetManager(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	serviceMetrics, err := manager.CollectMetrics(ctx)
	if err != nil {
		return nil, err
	}

	// Store metrics in database
	timestamp, _ := time.Parse(time.RFC3339, serviceMetrics.Timestamp)
	for metricName, value := range serviceMetrics.Metrics {
		// Convert value to float64
		var floatValue float64
		switch v := value.(type) {
		case int:
			floatValue = float64(v)
		case int64:
			floatValue = float64(v)
		case float64:
			floatValue = v
		case string:
			// Skip string values for now
			continue
		default:
			continue
		}

		metric := &models.Metric{
			InstanceID: instanceID,
			MetricName: metricName,
			Value:      floatValue,
			Time:       timestamp,
		}

		if err := c.metricsRepo.Create(ctx, metric); err != nil {
			logrus.Errorf("Failed to store metric %s: %v", metricName, err)
		}
	}

	return serviceMetrics, nil
}

// GetInfo gets service information from an instance
func (c *ManagerCoordinator) GetInfo(ctx context.Context, instanceID uuid.UUID) (map[string]interface{}, error) {
	manager, err := c.GetManager(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	return manager.GetInfo(ctx)
}

// StartMonitoring starts periodic monitoring of all instances
func (c *ManagerCoordinator) StartMonitoring(ctx context.Context) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		ticker := time.NewTicker(c.monitorInterval)
		defer ticker.Stop()

		logrus.Infof("Started instance monitoring (interval: %v)", c.monitorInterval)

		for {
			select {
			case <-ticker.C:
				c.monitorAllInstances(ctx)
			case <-c.stopCh:
				logrus.Info("Stopped instance monitoring")
				return
			}
		}
	}()
}

// StopMonitoring stops the monitoring
func (c *ManagerCoordinator) StopMonitoring() {
	close(c.stopCh)
	c.wg.Wait()
}

// monitorAllInstances monitors all active instances
func (c *ManagerCoordinator) monitorAllInstances(ctx context.Context) {
	// Get all active instances
	filter := &repository.ListInstancesFilter{
		Status:   "running",
		Page:     1,
		PageSize: 1000, // Get all running instances
	}

	instances, _, err := c.instanceRepo.ListInstances(ctx, filter)
	if err != nil {
		logrus.Errorf("Failed to list instances for monitoring: %v", err)
		return
	}

	logrus.Debugf("Monitoring %d instances", len(instances))

	// Monitor each instance in parallel
	var wg sync.WaitGroup
	for _, instance := range instances {
		wg.Add(1)
		go func(inst *models.Instance) {
			defer wg.Done()
			c.monitorInstance(ctx, inst.ID)
		}(instance)
	}

	wg.Wait()
}

// monitorInstance monitors a single instance
func (c *ManagerCoordinator) monitorInstance(ctx context.Context, instanceID uuid.UUID) {
	// Health check
	if _, err := c.HealthCheck(ctx, instanceID); err != nil {
		logrus.Errorf("Health check failed for instance %s: %v", instanceID, err)
		return
	}

	// Collect metrics
	if _, err := c.CollectMetrics(ctx, instanceID); err != nil {
		logrus.Errorf("Metrics collection failed for instance %s: %v", instanceID, err)
	}
}

// DisconnectAll disconnects all managers
func (c *ManagerCoordinator) DisconnectAll(ctx context.Context) {
	c.managersMu.Lock()
	defer c.managersMu.Unlock()

	for instanceID, manager := range c.managers {
		if err := manager.Disconnect(ctx); err != nil {
			logrus.Errorf("Failed to disconnect manager for instance %s: %v", instanceID, err)
		}
	}

	c.managers = make(map[uuid.UUID]ServiceManager)
	logrus.Info("Disconnected all managers")
}

// SetMonitorInterval sets the monitoring interval
func (c *ManagerCoordinator) SetMonitorInterval(interval time.Duration) {
	c.monitorInterval = interval
}
