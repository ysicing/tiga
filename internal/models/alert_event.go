package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AlertEvent represents an alert trigger event
type AlertEvent struct {
	ID         uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	AlertID    uuid.UUID `gorm:"type:char(36);not null;index" json:"alert_id"`
	InstanceID uuid.UUID `gorm:"type:char(36);not null;index" json:"instance_id"`

	// Event information
	Status  string `gorm:"type:varchar(32);not null;index" json:"status"` // firing, resolved
	Message string `gorm:"type:text;not null" json:"message"`
	Details JSONB  `gorm:"type:text" json:"details"`

	// Timestamps
	StartedAt  time.Time  `gorm:"not null;index" json:"started_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`

	// Notification status
	Notified           bool       `gorm:"default:false" json:"notified"`
	NotificationSentAt *time.Time `json:"notification_sent_at,omitempty"`

	// Associations
	Alert    *Alert    `gorm:"foreignKey:AlertID" json:"alert,omitempty"`
	Instance *Instance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
}

// TableName overrides the table name
func (AlertEvent) TableName() string {
	return "alert_events"
}

// BeforeCreate hook
func (ae *AlertEvent) BeforeCreate(tx *gorm.DB) error {
	if ae.ID == uuid.Nil {
		ae.ID = uuid.New()
	}
	return nil
}

// IsResolved checks if the alert event is resolved
func (ae *AlertEvent) IsResolved() bool {
	return ae.Status == "resolved" && ae.ResolvedAt != nil
}

// IsFiring checks if the alert event is currently firing
func (ae *AlertEvent) IsFiring() bool {
	return ae.Status == "firing" && ae.ResolvedAt == nil
}

// Duration calculates the duration of the alert event
func (ae *AlertEvent) Duration() time.Duration {
	if ae.ResolvedAt != nil {
		return ae.ResolvedAt.Sub(ae.StartedAt)
	}
	return time.Since(ae.StartedAt)
}
