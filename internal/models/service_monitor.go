package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProbeType represents the type of service probe
type ProbeType string

const (
	ProbeTypeHTTP ProbeType = "HTTP"
	ProbeTypeTCP  ProbeType = "TCP"
	ProbeTypeICMP ProbeType = "ICMP"
)

// ProbeStrategy represents how to select probe nodes
type ProbeStrategy string

const (
	ProbeStrategyServer  ProbeStrategy = "server"  // Server-side probing
	ProbeStrategyInclude ProbeStrategy = "include" // Include specific nodes
	ProbeStrategyExclude ProbeStrategy = "exclude" // Exclude specific nodes
	ProbeStrategyGroup   ProbeStrategy = "group"   // Probe from a node group
)

// ServiceMonitor represents a service health check configuration
type ServiceMonitor struct {
	BaseModel

	// Basic information
	Name     string    `gorm:"not null" json:"name"`
	Type     ProbeType `gorm:"not null;index" json:"type"`
	Target   string    `gorm:"not null" json:"target"`   // URL/IP:Port/IP
	Interval int       `gorm:"not null" json:"interval"` // Probe interval in seconds
	Timeout  int       `gorm:"not null" json:"timeout"`  // Timeout in seconds
	Enabled  bool      `gorm:"default:true;index" json:"enabled"`

	// Probe node selection strategy
	ProbeStrategy ProbeStrategy `gorm:"default:'server';index" json:"probe_strategy"`      // server/include/exclude/group
	ProbeNodeIDs  string        `gorm:"type:text" json:"probe_node_ids,omitempty"`         // JSON array of node UUIDs
	ProbeGroupName string       `gorm:"type:varchar(100);index" json:"probe_group_name,omitempty"` // Node group name for group strategy
	HostNodeID    *uuid.UUID    `gorm:"type:char(36);index" json:"host_node_id,omitempty"` // Legacy: single executor host (deprecated, use ProbeStrategy)

	// HTTP-specific configuration
	HTTPMethod   string `json:"http_method,omitempty"`                   // GET/POST
	HTTPHeaders  string `gorm:"type:text" json:"http_headers,omitempty"` // JSON map
	HTTPBody     string `gorm:"type:text" json:"http_body,omitempty"`
	ExpectStatus int    `json:"expect_status,omitempty"` // Expected HTTP status code
	ExpectBody   string `json:"expect_body,omitempty"`   // Expected response body substring

	// TCP-specific configuration
	TCPSend   string `json:"tcp_send,omitempty"`   // Data to send
	TCPExpect string `json:"tcp_expect,omitempty"` // Expected response

	// Alert configuration
	NotifyOnFailure   bool `gorm:"default:true" json:"notify_on_failure"`
	FailureThreshold  int  `gorm:"default:3" json:"failure_threshold"`  // Consecutive failures before alert
	RecoveryThreshold int  `gorm:"default:1" json:"recovery_threshold"` // Consecutive successes before recovery

	// Runtime status (not persisted)
	Status        string     `gorm:"-" json:"status,omitempty"` // online/offline/unknown
	LastCheckTime *time.Time `gorm:"-" json:"last_check_time,omitempty"`
	Uptime24h     float64    `gorm:"-" json:"uptime_24h,omitempty"` // 24h uptime percentage

	// Relationships
	HostNode     *HostNode             `gorm:"foreignKey:HostNodeID" json:"-"`
	ProbeResults []ServiceProbeResult  `gorm:"foreignKey:ServiceMonitorID" json:"-"`
	Availability []ServiceAvailability `gorm:"foreignKey:ServiceMonitorID" json:"-"`
	AlertRules   []MonitorAlertRule    `gorm:"foreignKey:TargetID" json:"-"`
}

// TableName specifies the table name for ServiceMonitor
func (ServiceMonitor) TableName() string {
	return "service_monitors"
}

// ServiceStatus represents the health status of a service
type ServiceStatus string

const (
	ServiceStatusGood            ServiceStatus = "Good"
	ServiceStatusLowAvailability ServiceStatus = "LowAvailability"
	ServiceStatusDown            ServiceStatus = "Down"
	ServiceStatusUnknown         ServiceStatus = "Unknown"
)

// GetStatusCode calculates the service status based on uptime percentage
// Uses the same thresholds as Nezha:
// - >= 95%: Good
// - >= 80%: LowAvailability
// - < 80%: Down
func (s *ServiceMonitor) GetStatusCode(uptimePercent float64) ServiceStatus {
	if uptimePercent >= 95.0 {
		return ServiceStatusGood
	} else if uptimePercent >= 80.0 {
		return ServiceStatusLowAvailability
	}
	return ServiceStatusDown
}

// IsHealthy checks if service is healthy (>= 95% uptime)
func (s *ServiceMonitor) IsHealthy(uptimePercent float64) bool {
	return uptimePercent >= 95.0
}

// BeforeCreate validates the model before creation
func (s *ServiceMonitor) BeforeCreate(tx *gorm.DB) error {
	// Call BaseModel's BeforeCreate to generate UUID
	if err := s.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}

	// Set default thresholds
	if s.FailureThreshold <= 0 {
		s.FailureThreshold = 3
	}
	if s.RecoveryThreshold <= 0 {
		s.RecoveryThreshold = 1
	}
	if s.Interval <= 0 {
		s.Interval = 60
	}
	if s.Timeout <= 0 {
		s.Timeout = 5
	}
	return nil
}
