package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Event represents a system event
type Event struct {
	ID         uuid.UUID  `gorm:"type:char(36);primary_key" json:"id"`
	EventType  string     `gorm:"type:varchar(64);not null;index" json:"event_type"` // instance.created, instance.health_changed
	InstanceID *uuid.UUID `gorm:"type:char(36);index" json:"instance_id,omitempty"`
	UserID     *uuid.UUID `gorm:"type:char(36);index" json:"user_id,omitempty"`

	// Event data
	Payload JSONB `gorm:"type:text;not null" json:"payload"`

	// Priority
	Priority string `gorm:"type:varchar(32);default:'normal';index" json:"priority"` // low, normal, high, critical

	// Processing status
	Processed   bool       `gorm:"default:false;index" json:"processed"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`

	// Timestamp
	CreatedAt time.Time `gorm:"index" json:"created_at"`

	// Associations
	Instance *Instance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
	User     *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName overrides the table name
func (Event) TableName() string {
	return "events"
}

// BeforeCreate hook
func (e *Event) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

// MarkProcessed marks the event as processed
func (e *Event) MarkProcessed(tx *gorm.DB) error {
	now := time.Now()
	e.Processed = true
	e.ProcessedAt = &now
	return tx.Model(e).Updates(map[string]interface{}{
		"processed":    true,
		"processed_at": now,
	}).Error
}
