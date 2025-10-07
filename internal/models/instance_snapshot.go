package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InstanceSnapshot represents a snapshot of instance configuration
type InstanceSnapshot struct {
	ID         uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	InstanceID uuid.UUID `gorm:"type:char(36);not null;index" json:"instance_id"`

	// Snapshot content
	Snapshot JSONB `gorm:"type:text;not null" json:"snapshot"` // Complete instance configuration

	// Change information
	ChangeType    string      `gorm:"type:varchar(32);not null" json:"change_type"` // created, updated, deleted
	ChangedFields StringArray `gorm:"type:text" json:"changed_fields"`
	ChangeReason  string      `gorm:"type:text" json:"change_reason,omitempty"`

	// Creator
	CreatedBy *uuid.UUID     `gorm:"type:char(36)" json:"created_by,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // Soft delete

	// Associations
	Instance *Instance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
}

// TableName overrides the table name
func (InstanceSnapshot) TableName() string {
	return "instance_snapshots"
}

// BeforeCreate hook
func (is *InstanceSnapshot) BeforeCreate(tx *gorm.DB) error {
	if is.ID == uuid.Nil {
		is.ID = uuid.New()
	}
	return nil
}
