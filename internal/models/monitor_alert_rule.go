package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertType represents the type of resource being monitored
type AlertType string

const (
	AlertTypeHost    AlertType = "host"
	AlertTypeService AlertType = "service"
)

// MonitorAlertRule represents an alert rule configuration
type MonitorAlertRule struct {
	BaseModel

	// Basic information
	Name     string        `gorm:"not null" json:"name"`
	Type     AlertType     `gorm:"not null;index" json:"type"`              // host/service
	TargetID uuid.UUID     `gorm:"type:char(36);index;not null" json:"target_id"` // HostNode ID or ServiceMonitor ID
	Severity AlertSeverity `gorm:"not null;index" json:"severity"`

	// Condition expression (using antonmedv/expr)
	// Examples:
	// - Host: "cpu_usage > 80 && load_5 > 10"
	// - Service: "uptime_percentage < 99.9 && failed_checks > 10"
	Condition string `gorm:"type:text;not null" json:"condition"`

	// Trigger configuration
	Duration int  `gorm:"default:300" json:"duration"` // Condition must be true for this duration (seconds)
	Enabled  bool `gorm:"default:true;index" json:"enabled"`

	// Notification channels (JSON array: ["email", "webhook", "sms"])
	NotifyChannels string `gorm:"type:text" json:"notify_channels"`

	// Notification configuration (JSON map)
	// Example: {"email": ["admin@example.com"], "webhook": "https://hooks.example.com"}
	NotifyConfig string `gorm:"type:text" json:"notify_config"`

	// Runtime statistics (not persisted)
	LastTriggered *string `gorm:"-" json:"last_triggered,omitempty"`
	TriggerCount  int     `gorm:"-" json:"trigger_count,omitempty"`

	// Relationships
	Events []MonitorAlertEvent `gorm:"foreignKey:RuleID" json:"-"`
}

// TableName specifies the table name for MonitorAlertRule
func (MonitorAlertRule) TableName() string {
	return "monitor_alert_rules"
}

// BeforeCreate validates the model before creation
func (m *MonitorAlertRule) BeforeCreate(tx *gorm.DB) error {
	// Call BaseModel's BeforeCreate to generate UUID
	if err := m.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}

	if m.Duration <= 0 {
		m.Duration = 300 // Default 5 minutes
	}
	return nil
}
