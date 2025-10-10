package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MonitorAlertStatus represents the status of a monitor alert event
type MonitorAlertStatus string

const (
	AlertStatusFiring       MonitorAlertStatus = "firing"
	AlertStatusAcknowledged MonitorAlertStatus = "acknowledged"
	AlertStatusResolved     MonitorAlertStatus = "resolved"
)

// MonitorAlertEvent represents an alert event triggered by a monitor alert rule
type MonitorAlertEvent struct {
	BaseModel

	RuleID uuid.UUID `gorm:"type:char(36);index:idx_rule_status,priority:1;not null" json:"rule_id"`

	// Event details
	Status   MonitorAlertStatus `gorm:"index:idx_rule_status,priority:2;index;not null" json:"status"`
	Severity AlertSeverity      `gorm:"index;not null" json:"severity"`
	Message  string             `gorm:"type:text" json:"message"`

	// Context data (JSON: actual metric values that triggered the alert)
	// Example: {"cpu_usage": 85.5, "load_5": 12.3}
	Context string `gorm:"type:text" json:"context"`

	// Timeline
	TriggeredAt time.Time  `gorm:"index;not null" json:"triggered_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`

	// Acknowledgment
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedBy *uuid.UUID `gorm:"type:char(36)" json:"acknowledged_by,omitempty"` // User ID
	AckNote        string     `gorm:"type:text" json:"ack_note,omitempty"`

	// Resolution
	ResolvedBy *uuid.UUID `gorm:"type:char(36)" json:"resolved_by,omitempty"` // User ID
	ResNote    string     `gorm:"type:text" json:"res_note,omitempty"`

	// Notification tracking (JSON array of sent notifications)
	// Example: [{"channel": "email", "sent_at": "2025-10-07T10:00:00Z", "success": true}]
	Notifications string `gorm:"type:text" json:"notifications,omitempty"`

	// Relationships
	Rule             *MonitorAlertRule `gorm:"foreignKey:RuleID" json:"rule,omitempty"`
	AcknowledgedUser *User             `gorm:"foreignKey:AcknowledgedBy" json:"-"`
	ResolvedUser     *User             `gorm:"foreignKey:ResolvedBy" json:"-"`
}

// TableName specifies the table name for MonitorAlertEvent
func (MonitorAlertEvent) TableName() string {
	return "monitor_alert_events"
}

// BeforeCreate sets initial values
func (a *MonitorAlertEvent) BeforeCreate(tx *gorm.DB) error {
	// Call BaseModel's BeforeCreate to generate UUID
	if err := a.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}

	if a.TriggeredAt.IsZero() {
		a.TriggeredAt = time.Now()
	}
	if a.Status == "" {
		a.Status = AlertStatusFiring
	}
	return nil
}

// Acknowledge marks the event as acknowledged
func (a *MonitorAlertEvent) Acknowledge(userID uuid.UUID, note string) {
	now := time.Now()
	a.Status = AlertStatusAcknowledged
	a.AcknowledgedAt = &now
	a.AcknowledgedBy = &userID
	a.AckNote = note
}

// Resolve marks the event as resolved
func (a *MonitorAlertEvent) Resolve(userID uuid.UUID, note string) {
	now := time.Now()
	a.Status = AlertStatusResolved
	a.ResolvedAt = &now
	a.ResolvedBy = &userID
	a.ResNote = note
}

// IsFiring checks if the event is currently firing
func (a *MonitorAlertEvent) IsFiring() bool {
	return a.Status == AlertStatusFiring
}
