package managers

import (
	"context"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
)

// HealthStatus represents the health status of a service
type HealthStatus struct {
	Healthy       bool                   `json:"healthy"`
	Message       string                 `json:"message,omitempty"`
	Details       map[string]interface{} `json:"details,omitempty"`
	ResponseTime  int64                  `json:"response_time_ms"`
	LastCheckTime string                 `json:"last_check_time"`
}

// ServiceMetrics represents metrics collected from a service
type ServiceMetrics struct {
	InstanceID uuid.UUID              `json:"instance_id"`
	Metrics    map[string]interface{} `json:"metrics"`
	Timestamp  string                 `json:"timestamp"`
}

// ServiceManager is the interface that all service managers must implement
type ServiceManager interface {
	// Initialize initializes the manager with instance configuration
	Initialize(ctx context.Context, instance *models.Instance) error

	// Connect establishes connection to the service
	Connect(ctx context.Context) error

	// Disconnect closes connection to the service
	Disconnect(ctx context.Context) error

	// HealthCheck checks the health of the service
	HealthCheck(ctx context.Context) (*HealthStatus, error)

	// CollectMetrics collects metrics from the service
	CollectMetrics(ctx context.Context) (*ServiceMetrics, error)

	// GetInfo returns service-specific information
	GetInfo(ctx context.Context) (map[string]interface{}, error)

	// ValidateConfig validates the service configuration
	ValidateConfig(config map[string]interface{}) error

	// Type returns the service type
	Type() string
}

// BaseManager provides common functionality for all service managers
type BaseManager struct {
	instance *models.Instance
	config   map[string]interface{}
}

// NewBaseManager creates a new base manager
func NewBaseManager() *BaseManager {
	return &BaseManager{}
}

// Initialize initializes the base manager
func (m *BaseManager) Initialize(ctx context.Context, instance *models.Instance) error {
	m.instance = instance
	m.config = instance.Config
	return nil
}

// GetInstance returns the associated instance
func (m *BaseManager) GetInstance() *models.Instance {
	return m.instance
}

// GetConfig returns the instance configuration
func (m *BaseManager) GetConfig() map[string]interface{} {
	return m.config
}

// GetConfigValue returns a configuration value
func (m *BaseManager) GetConfigValue(key string, defaultValue interface{}) interface{} {
	if val, ok := m.config[key]; ok {
		return val
	}
	return defaultValue
}

// GetConnectionString builds a connection string from the instance
func (m *BaseManager) GetConnectionString() (string, error) {
	if m.instance == nil {
		return "", ErrInstanceNotInitialized
	}

	connection := m.instance.Connection
	host, _ := connection["host"].(string)
	port, _ := connection["port"].(float64)

	if host == "" || port == 0 {
		return "", ErrInvalidConnection
	}

	return host + ":" + string(rune(int(port))), nil
}
