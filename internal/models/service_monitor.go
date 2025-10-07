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

// ServiceMonitor represents a service health check configuration
type ServiceMonitor struct {
	BaseModel

	// Basic information
	Name       string     `gorm:"not null" json:"name"`
	Type       ProbeType  `gorm:"not null;index" json:"type"`
	Target     string     `gorm:"not null" json:"target"`         // URL/IP:Port/IP
	Interval   int        `gorm:"not null" json:"interval"`       // Probe interval in seconds
	Timeout    int        `gorm:"not null" json:"timeout"`        // Timeout in seconds
	HostNodeID *uuid.UUID `gorm:"type:char(36);index" json:"host_node_id,omitempty"` // Executor host
	Enabled    bool       `gorm:"default:true;index" json:"enabled"`

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
