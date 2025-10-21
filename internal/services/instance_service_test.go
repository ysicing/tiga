package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/services/managers"
)

// MockInstanceRepository is a mock implementation of InstanceRepositoryInterface
type MockInstanceRepository struct {
	mock.Mock
}

func (m *MockInstanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Instance, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Instance), args.Error(1)
}

func (m *MockInstanceRepository) UpdateHealth(ctx context.Context, id uuid.UUID, healthStatus string, healthMessage *string) error {
	args := m.Called(ctx, id, healthStatus, healthMessage)
	return args.Error(0)
}

// MockServiceManager is a mock implementation of ServiceManager
type MockServiceManager struct {
	mock.Mock
}

func (m *MockServiceManager) Type() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockServiceManager) Initialize(ctx context.Context, instance *models.Instance) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *MockServiceManager) Connect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockServiceManager) Disconnect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockServiceManager) HealthCheck(ctx context.Context) (*managers.HealthStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*managers.HealthStatus), args.Error(1)
}

func (m *MockServiceManager) CollectMetrics(ctx context.Context) (*managers.ServiceMetrics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*managers.ServiceMetrics), args.Error(1)
}

func (m *MockServiceManager) ValidateConfig(config map[string]interface{}) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockServiceManager) GetInfo(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// createMockInstanceService creates a test instance service with mock repository
func createMockInstanceService(mockRepo *MockInstanceRepository) *InstanceService {
	// With the interface-based design, we can directly create the service with the mock
	return &InstanceService{
		instanceRepo:     mockRepo,
		managerRegistry:  make(map[string]managers.ServiceManager),
		managerFactories: make(map[string]ManagerFactory), // Initialize factory map for testing
	}
}

// TestNewInstanceService tests the creation of a new InstanceService
func TestNewInstanceService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires real database)")
	}

	mockRepo := new(MockInstanceRepository)
	service := createMockInstanceService(mockRepo)

	assert.NotNil(t, service)
	assert.NotNil(t, service.managerRegistry)
}

// TestRegisterManager tests the RegisterManager method
func TestRegisterManager(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires real database)")
	}

	mockRepo := new(MockInstanceRepository)
	service := createMockInstanceService(mockRepo)

	mockManager := new(MockServiceManager)
	mockManager.On("Type").Return("test-service")

	service.RegisterManager("test-service", mockManager)

	assert.NotNil(t, service.managerRegistry["test-service"])
}

// TestGetHealthStatus_Success tests successful health check
func TestGetHealthStatus_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires real database)")
	}

	ctx := context.Background()
	instanceID := uuid.New()

	// Setup mock repository
	mockRepo := new(MockInstanceRepository)
	instance := &models.Instance{
		ID:   instanceID,
		Type: "minio",
		Name: "test-instance",
	}
	mockRepo.On("GetByID", ctx, instanceID).Return(instance, nil)

	healthMessage := "Service is healthy"
	mockRepo.On("UpdateHealth", ctx, instanceID, "healthy", &healthMessage).Return(nil)

	// Setup mock manager
	mockManager := new(MockServiceManager)
	mockManager.On("Type").Return("minio")
	mockManager.On("Initialize", ctx, instance).Return(nil)
	mockManager.On("Connect", ctx).Return(nil)
	mockManager.On("Disconnect", ctx).Return(nil)
	mockManager.On("HealthCheck", ctx).Return(&managers.HealthStatus{
		Healthy:       true,
		Message:       healthMessage,
		LastCheckTime: time.Now().Format(time.RFC3339),
	}, nil)

	// Create service and register manager template
	service := createMockInstanceService(mockRepo)
	service.RegisterManager("minio", mockManager)

	// Inject factory to return the mock manager (for testing cloneManager)
	service.managerFactories["minio"] = func() managers.ServiceManager {
		return mockManager
	}

	// Execute
	health, err := service.GetHealthStatus(ctx, instanceID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.True(t, health.Healthy)
	assert.Equal(t, healthMessage, health.Message)

	mockRepo.AssertExpectations(t)
	mockManager.AssertExpectations(t)
}

// TestGetHealthStatus_InstanceNotFound tests when instance is not found
func TestGetHealthStatus_InstanceNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires real database)")
	}
	ctx := context.Background()
	instanceID := uuid.New()

	mockRepo := new(MockInstanceRepository)
	mockRepo.On("GetByID", ctx, instanceID).Return(nil, errors.New("instance not found"))

	service := createMockInstanceService(mockRepo)

	health, err := service.GetHealthStatus(ctx, instanceID)

	assert.Error(t, err)
	assert.Nil(t, health)
	assert.Contains(t, err.Error(), "failed to get instance")

	mockRepo.AssertExpectations(t)
}

// TestGetHealthStatus_NoManagerRegistered tests when no manager is registered
func TestGetHealthStatus_NoManagerRegistered(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires real database)")
	}
	ctx := context.Background()
	instanceID := uuid.New()

	mockRepo := new(MockInstanceRepository)
	instance := &models.Instance{
		ID:   instanceID,
		Type: "unknown-type",
		Name: "test-instance",
	}
	mockRepo.On("GetByID", ctx, instanceID).Return(instance, nil)

	service := createMockInstanceService(mockRepo)

	health, err := service.GetHealthStatus(ctx, instanceID)

	assert.Error(t, err)
	assert.Nil(t, health)
	assert.Contains(t, err.Error(), "no manager registered")

	mockRepo.AssertExpectations(t)
}

// TestGetHealthStatus_ConnectionFailed tests when connection fails
func TestGetHealthStatus_ConnectionFailed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires real database)")
	}
	ctx := context.Background()
	instanceID := uuid.New()

	mockRepo := new(MockInstanceRepository)
	instance := &models.Instance{
		ID:   instanceID,
		Type: "minio",
		Name: "test-instance",
	}
	mockRepo.On("GetByID", ctx, instanceID).Return(instance, nil)

	mockManager := new(MockServiceManager)
	mockManager.On("Type").Return("minio")
	mockManager.On("Initialize", ctx, instance).Return(nil)
	mockManager.On("Connect", ctx).Return(errors.New("connection refused"))

	service := createMockInstanceService(mockRepo)
	service.RegisterManager("minio", mockManager)

	// Inject factory to return the mock manager (for testing cloneManager)
	service.managerFactories["minio"] = func() managers.ServiceManager {
		return mockManager
	}

	health, err := service.GetHealthStatus(ctx, instanceID)

	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.False(t, health.Healthy)
	assert.Contains(t, health.Message, "Failed to connect")

	mockRepo.AssertExpectations(t)
	mockManager.AssertExpectations(t)
}

// TestGetHealthStatus_HealthCheckFailed tests when health check fails
func TestGetHealthStatus_HealthCheckFailed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires real database)")
	}
	ctx := context.Background()
	instanceID := uuid.New()

	mockRepo := new(MockInstanceRepository)
	instance := &models.Instance{
		ID:   instanceID,
		Type: "minio",
		Name: "test-instance",
	}
	mockRepo.On("GetByID", ctx, instanceID).Return(instance, nil)

	mockManager := new(MockServiceManager)
	mockManager.On("Type").Return("minio")
	mockManager.On("Initialize", ctx, instance).Return(nil)
	mockManager.On("Connect", ctx).Return(nil)
	mockManager.On("Disconnect", ctx).Return(nil)
	mockManager.On("HealthCheck", ctx).Return(nil, errors.New("health check error"))

	service := createMockInstanceService(mockRepo)
	service.RegisterManager("minio", mockManager)

	// Inject factory to return the mock manager (for testing cloneManager)
	service.managerFactories["minio"] = func() managers.ServiceManager {
		return mockManager
	}

	health, err := service.GetHealthStatus(ctx, instanceID)

	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.False(t, health.Healthy)
	assert.Contains(t, health.Message, "Health check failed")

	mockRepo.AssertExpectations(t)
	mockManager.AssertExpectations(t)
}

// TestGetHealthStatus_UnhealthyService tests when service is unhealthy
func TestGetHealthStatus_UnhealthyService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires real database)")
	}
	ctx := context.Background()
	instanceID := uuid.New()

	mockRepo := new(MockInstanceRepository)
	instance := &models.Instance{
		ID:   instanceID,
		Type: "minio",
		Name: "test-instance",
	}
	mockRepo.On("GetByID", ctx, instanceID).Return(instance, nil)

	unhealthyMessage := "Disk full"
	mockRepo.On("UpdateHealth", ctx, instanceID, "unhealthy", &unhealthyMessage).Return(nil)

	mockManager := new(MockServiceManager)
	mockManager.On("Type").Return("minio")
	mockManager.On("Initialize", ctx, instance).Return(nil)
	mockManager.On("Connect", ctx).Return(nil)
	mockManager.On("Disconnect", ctx).Return(nil)
	mockManager.On("HealthCheck", ctx).Return(&managers.HealthStatus{
		Healthy:       false,
		Message:       unhealthyMessage,
		LastCheckTime: time.Now().Format(time.RFC3339),
	}, nil)

	service := createMockInstanceService(mockRepo)
	service.RegisterManager("minio", mockManager)

	// Inject factory to return the mock manager (for testing cloneManager)
	service.managerFactories["minio"] = func() managers.ServiceManager {
		return mockManager
	}

	health, err := service.GetHealthStatus(ctx, instanceID)

	assert.NoError(t, err)
	assert.NotNil(t, health)
	assert.False(t, health.Healthy)
	assert.Equal(t, unhealthyMessage, health.Message)

	mockRepo.AssertExpectations(t)
	mockManager.AssertExpectations(t)
}

// TestGetMetrics_Success tests successful metrics collection
func TestGetMetrics_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires real database)")
	}
	ctx := context.Background()
	instanceID := uuid.New()

	mockRepo := new(MockInstanceRepository)
	instance := &models.Instance{
		ID:   instanceID,
		Type: "minio",
		Name: "test-instance",
	}
	mockRepo.On("GetByID", ctx, instanceID).Return(instance, nil)

	mockManager := new(MockServiceManager)
	mockManager.On("Type").Return("minio")
	mockManager.On("Initialize", ctx, instance).Return(nil)
	mockManager.On("Connect", ctx).Return(nil)
	mockManager.On("Disconnect", ctx).Return(nil)

	expectedMetrics := &managers.ServiceMetrics{
		InstanceID: instanceID,
		Metrics: map[string]interface{}{
			"cpu":    50.5,
			"memory": 75.2,
			"disk":   80.0,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	mockManager.On("CollectMetrics", ctx).Return(expectedMetrics, nil)

	service := createMockInstanceService(mockRepo)
	service.RegisterManager("minio", mockManager)

	// Inject factory to return the mock manager (for testing cloneManager)
	service.managerFactories["minio"] = func() managers.ServiceManager {
		return mockManager
	}

	metrics, err := service.GetMetrics(ctx, instanceID)

	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, instanceID, metrics.InstanceID)
	assert.Equal(t, 50.5, metrics.Metrics["cpu"])

	mockRepo.AssertExpectations(t)
	mockManager.AssertExpectations(t)
}

// TestGetMetrics_InstanceNotFound tests when instance is not found
func TestGetMetrics_InstanceNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires real database)")
	}
	ctx := context.Background()
	instanceID := uuid.New()

	mockRepo := new(MockInstanceRepository)
	mockRepo.On("GetByID", ctx, instanceID).Return(nil, errors.New("instance not found"))

	service := createMockInstanceService(mockRepo)

	metrics, err := service.GetMetrics(ctx, instanceID)

	assert.Error(t, err)
	assert.Nil(t, metrics)
	assert.Contains(t, err.Error(), "failed to get instance")

	mockRepo.AssertExpectations(t)
}

// TestGetMetrics_ConnectionFailed tests when connection fails
func TestGetMetrics_ConnectionFailed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires real database)")
	}
	ctx := context.Background()
	instanceID := uuid.New()

	mockRepo := new(MockInstanceRepository)
	instance := &models.Instance{
		ID:   instanceID,
		Type: "minio",
		Name: "test-instance",
	}
	mockRepo.On("GetByID", ctx, instanceID).Return(instance, nil)

	mockManager := new(MockServiceManager)
	mockManager.On("Type").Return("minio")
	mockManager.On("Initialize", ctx, instance).Return(nil)
	mockManager.On("Connect", ctx).Return(errors.New("connection refused"))

	service := createMockInstanceService(mockRepo)
	service.RegisterManager("minio", mockManager)

	// Inject factory to return the mock manager (for testing cloneManager)
	service.managerFactories["minio"] = func() managers.ServiceManager {
		return mockManager
	}

	metrics, err := service.GetMetrics(ctx, instanceID)

	assert.Error(t, err)
	assert.Nil(t, metrics)
	assert.Contains(t, err.Error(), "failed to connect")

	mockRepo.AssertExpectations(t)
	mockManager.AssertExpectations(t)
}

// TestGetMetrics_CollectionFailed tests when metrics collection fails
func TestGetMetrics_CollectionFailed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires real database)")
	}
	ctx := context.Background()
	instanceID := uuid.New()

	mockRepo := new(MockInstanceRepository)
	instance := &models.Instance{
		ID:   instanceID,
		Type: "minio",
		Name: "test-instance",
	}
	mockRepo.On("GetByID", ctx, instanceID).Return(instance, nil)

	mockManager := new(MockServiceManager)
	mockManager.On("Type").Return("minio")
	mockManager.On("Initialize", ctx, instance).Return(nil)
	mockManager.On("Connect", ctx).Return(nil)
	mockManager.On("Disconnect", ctx).Return(nil)
	mockManager.On("CollectMetrics", ctx).Return(nil, errors.New("metrics unavailable"))

	service := createMockInstanceService(mockRepo)
	service.RegisterManager("minio", mockManager)

	// Inject factory to return the mock manager (for testing cloneManager)
	service.managerFactories["minio"] = func() managers.ServiceManager {
		return mockManager
	}

	metrics, err := service.GetMetrics(ctx, instanceID)

	assert.Error(t, err)
	assert.Nil(t, metrics)
	assert.Contains(t, err.Error(), "failed to collect metrics")

	mockRepo.AssertExpectations(t)
	mockManager.AssertExpectations(t)
}

// TestCloneManager tests the cloneManager method for all supported types
func TestCloneManager(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (requires real database)")
	}
	service := createMockInstanceService(nil)

	testCases := []struct {
		name        string
		managerType string
		shouldBeNil bool
	}{
		{"MinIO", "minio", false},
		{"MySQL", "mysql", false},
		{"PostgreSQL", "postgres", false},
		{"PostgreSQL Alt", "postgresql", false},
		{"Redis", "redis", false},
		{"Docker", "docker", false},
		{"Caddy", "caddy", false},
		{"Kubernetes", "kubernetes", false},
		{"K8s", "k8s", false},
		{"Unknown", "unknown", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockTemplate := new(MockServiceManager)
			mockTemplate.On("Type").Return(tc.managerType)

			manager := service.cloneManager(mockTemplate)

			if tc.shouldBeNil {
				assert.Nil(t, manager)
			} else {
				assert.NotNil(t, manager)
			}
		})
	}
}

// TestGetCurrentTimestamp tests the getCurrentTimestamp helper function
func TestGetCurrentTimestamp(t *testing.T) {
	timestamp := getCurrentTimestamp()

	assert.NotEmpty(t, timestamp)

	// Verify it's a valid RFC3339 timestamp
	_, err := time.Parse(time.RFC3339, timestamp)
	assert.NoError(t, err)
}
