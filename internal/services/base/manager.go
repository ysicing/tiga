package base

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
)

// ServiceManager defines the unified interface for all service managers
// All service types (MinIO, MySQL, Redis, Docker, K8s, Caddy) implement this interface
type ServiceManager interface {
	// Lifecycle management
	Create(ctx context.Context, config *InstanceConfig) (*models.Instance, error)
	Update(ctx context.Context, id uuid.UUID, config *InstanceConfig) error
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*models.Instance, error)
	List(ctx context.Context, filter *Filter) ([]*models.Instance, error)

	// Health check
	HealthCheck(ctx context.Context, id uuid.UUID) (*HealthStatus, error)

	// Metrics collection
	GetMetrics(ctx context.Context, id uuid.UUID, timeRange TimeRange) (*MetricsResult, error)

	// Service type identification
	Type() string
}

// InstanceConfig represents the configuration for creating/updating an instance
type InstanceConfig struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`

	// Connection details (will be encrypted in database)
	Connection ConnectionConfig `json:"connection"`

	// Configuration
	Config            map[string]interface{} `json:"config,omitempty"`
	HealthCheckConfig map[string]interface{} `json:"health_check_config,omitempty"`

	// Classification
	Tags        []string          `json:"tags,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Environment string            `json:"environment,omitempty"` // dev, staging, prod
	Team        string            `json:"team,omitempty"`

	// Ownership
	OwnerID uuid.UUID `json:"owner_id"`
}

// ConnectionConfig represents connection information for an instance
type ConnectionConfig struct {
	Host     string                 `json:"host"`
	Port     int                    `json:"port,omitempty"`
	Protocol string                 `json:"protocol,omitempty"` // http, https, tcp, etc.
	Username string                 `json:"username,omitempty"`
	Password string                 `json:"password,omitempty"` // Will be encrypted
	Database string                 `json:"database,omitempty"` // For database instances
	TLS      *TLSConfig             `json:"tls,omitempty"`
	Extra    map[string]interface{} `json:"extra,omitempty"` // Type-specific connection params
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled            bool   `json:"enabled"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
	CACert             string `json:"ca_cert,omitempty"`
	ClientCert         string `json:"client_cert,omitempty"`
	ClientKey          string `json:"client_key,omitempty"`
}

// Filter represents query filters for listing instances
type Filter struct {
	Type        string            `json:"type,omitempty"`
	Status      string            `json:"status,omitempty"`
	Health      string            `json:"health,omitempty"`
	Environment string            `json:"environment,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	OwnerID     *uuid.UUID        `json:"owner_id,omitempty"`
	Team        string            `json:"team,omitempty"`
	Search      string            `json:"search,omitempty"` // Full-text search

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Sorting
	SortBy    string `json:"sort_by,omitempty"`    // created_at, updated_at, name
	SortOrder string `json:"sort_order,omitempty"` // asc, desc
}

// HealthStatus represents the health status of an instance
type HealthStatus struct {
	Healthy   bool                   `json:"healthy"`
	Status    string                 `json:"status"` // healthy, unhealthy, degraded, unknown
	Message   string                 `json:"message,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	CheckedAt time.Time              `json:"checked_at"`
	Latency   time.Duration          `json:"latency,omitempty"` // Health check latency
}

// TimeRange represents a time range for metrics queries
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Step  string    `json:"step,omitempty"` // Aggregation interval: 1m, 5m, 1h, etc.
}

// MetricsResult represents metrics data for an instance
type MetricsResult struct {
	InstanceID uuid.UUID             `json:"instance_id"`
	TimeRange  TimeRange             `json:"time_range"`
	Metrics    map[string]MetricData `json:"metrics"` // metric_name -> data
}

// MetricData represents a single metric's data points
type MetricData struct {
	Name   string      `json:"name"`
	Type   string      `json:"type"` // gauge, counter, histogram
	Unit   string      `json:"unit,omitempty"`
	Points []DataPoint `json:"points"`
}

// DataPoint represents a single metric data point
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// InstanceStatus represents valid instance status values
const (
	StatusRunning      = "running"
	StatusStopped      = "stopped"
	StatusError        = "error"
	StatusUnknown      = "unknown"
	StatusProvisioning = "provisioning"
)

// HealthStatusConstants represents valid health status values
const (
	HealthHealthy   = "healthy"
	HealthUnhealthy = "unhealthy"
	HealthDegraded  = "degraded"
	HealthUnknown   = "unknown"
)

// MetricType represents valid metric types
const (
	MetricTypeGauge     = "gauge"
	MetricTypeCounter   = "counter"
	MetricTypeHistogram = "histogram"
)
