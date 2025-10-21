package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/managers"
)

// InstanceRepositoryInterface defines the repository interface needed by InstanceService
type InstanceRepositoryInterface interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Instance, error)
	UpdateHealth(ctx context.Context, id uuid.UUID, healthStatus string, healthMessage *string) error
}

// ManagerFactory is a function that creates a new ServiceManager instance
type ManagerFactory func() managers.ServiceManager

// InstanceService manages service instances and their health/metrics
type InstanceService struct {
	instanceRepo     InstanceRepositoryInterface
	managerRegistry  map[string]managers.ServiceManager
	managerFactories map[string]ManagerFactory // For testing: inject custom factories
}

// NewInstanceService creates a new instance service
func NewInstanceService(instanceRepo *repository.InstanceRepository) *InstanceService {
	return &InstanceService{
		instanceRepo:     instanceRepo,
		managerRegistry:  make(map[string]managers.ServiceManager),
		managerFactories: make(map[string]ManagerFactory),
	}
}

// RegisterManager registers a service manager for a specific service type
func (s *InstanceService) RegisterManager(serviceType string, manager managers.ServiceManager) {
	s.managerRegistry[serviceType] = manager
}

// getManagerForInstance gets the appropriate manager for an instance
func (s *InstanceService) getManagerForInstance(ctx context.Context, instance *models.Instance) (managers.ServiceManager, error) {
	managerTemplate, ok := s.managerRegistry[instance.Type]
	if !ok {
		return nil, fmt.Errorf("no manager registered for service type: %s", instance.Type)
	}

	// Create a new instance of the manager
	manager := s.cloneManager(managerTemplate)
	if manager == nil {
		return nil, fmt.Errorf("unsupported service type: %s", instance.Type)
	}

	// Initialize with instance config
	if err := manager.Initialize(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to initialize manager: %w", err)
	}

	return manager, nil
}

// cloneManager creates a new instance of a manager based on its type
func (s *InstanceService) cloneManager(template managers.ServiceManager) managers.ServiceManager {
	managerType := template.Type()

	// Check if a factory is registered (for testing)
	if factory, ok := s.managerFactories[managerType]; ok {
		return factory()
	}

	// Default: create new instances based on type
	switch managerType {
	case "minio":
		return managers.NewMinIOManager()
	case "mysql":
		return managers.NewMySQLManager()
	case "postgres", "postgresql":
		return managers.NewPostgreSQLManager()
	case "redis":
		return managers.NewRedisManager()
	case "docker":
		return managers.NewDockerManager()
	case "caddy":
		return managers.NewCaddyManager()
	case "kubernetes", "k8s":
		return managers.NewK8sManager()
	default:
		// Return nil for unknown types - caller should check
		return nil
	}
}

// GetHealthStatus retrieves health status for an instance
func (s *InstanceService) GetHealthStatus(ctx context.Context, instanceID uuid.UUID) (*managers.HealthStatus, error) {
	// Get instance from repository
	instance, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// Get manager for this instance
	manager, err := s.getManagerForInstance(ctx, instance)
	if err != nil {
		return nil, err
	}

	// Connect to service
	if err := manager.Connect(ctx); err != nil {
		return &managers.HealthStatus{
			Healthy:       false,
			Message:       fmt.Sprintf("Failed to connect: %v", err),
			LastCheckTime: getCurrentTimestamp(),
		}, nil
	}
	defer manager.Disconnect(ctx)

	// Perform health check
	health, err := manager.HealthCheck(ctx)
	if err != nil {
		return &managers.HealthStatus{
			Healthy:       false,
			Message:       fmt.Sprintf("Health check failed: %v", err),
			LastCheckTime: getCurrentTimestamp(),
		}, nil
	}

	// Update instance health in database
	healthStatus := "healthy"
	if !health.Healthy {
		healthStatus = "unhealthy"
	}
	healthMessage := health.Message
	_ = s.instanceRepo.UpdateHealth(ctx, instanceID, healthStatus, &healthMessage)

	return health, nil
}

// GetMetrics retrieves metrics for an instance
func (s *InstanceService) GetMetrics(ctx context.Context, instanceID uuid.UUID) (*managers.ServiceMetrics, error) {
	// Get instance from repository
	instance, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// Get manager for this instance
	manager, err := s.getManagerForInstance(ctx, instance)
	if err != nil {
		return nil, err
	}

	// Connect to service
	if err := manager.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer manager.Disconnect(ctx)

	// Collect metrics
	metrics, err := manager.CollectMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics: %w", err)
	}

	return metrics, nil
}

// getCurrentTimestamp returns current timestamp in ISO 8601 format
func getCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}
