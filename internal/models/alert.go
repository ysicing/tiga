package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Alert represents an alert rule
type Alert struct {
	ID          uuid.UUID  `gorm:"type:char(36);primary_key" json:"id"`
	Name        string     `gorm:"type:varchar(128);not null" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	InstanceID  *uuid.UUID `gorm:"type:char(36);index:idx_instance_enabled" json:"instance_id,omitempty"`

	// Rule configuration
	RuleType   string `gorm:"type:varchar(32);not null" json:"rule_type"` // threshold, anomaly, rate
	RuleConfig JSONB  `gorm:"type:text;not null" json:"rule_config"`

	// Severity
	Severity string `gorm:"type:varchar(32);not null;index" json:"severity"` // critical, warning, info

	// Notification
	NotificationChannels StringArray `gorm:"type:text" json:"notification_channels"` // email, slack, webhook
	NotificationConfig   JSONB       `gorm:"type:text" json:"notification_config"`

	// Status
	Enabled   bool       `gorm:"default:true;index:idx_instance_enabled" json:"enabled"`
	Muted     bool       `gorm:"default:false" json:"muted"`
	MuteUntil *time.Time `json:"mute_until,omitempty"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // Soft delete

	// Associations
	Instance *Instance    `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
	Events   []AlertEvent `gorm:"foreignKey:AlertID" json:"events,omitempty"`
}

// TableName overrides the table name
func (Alert) TableName() string {
	return "alerts"
}

// BeforeCreate hook
func (a *Alert) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// IsMuted checks if the alert is currently muted
func (a *Alert) IsMuted() bool {
	if !a.Muted {
		return false
	}
	if a.MuteUntil == nil {
		return true
	}
	return time.Now().Before(*a.MuteUntil)
}

// IsActive checks if the alert is active (enabled and not muted)
func (a *Alert) IsActive() bool {
	return a.Enabled && !a.IsMuted()
}
