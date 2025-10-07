package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SystemConfig represents a system configuration setting
type SystemConfig struct {
	Key         string `gorm:"primaryKey;type:varchar(128)" json:"key"`
	Value       JSONB  `gorm:"type:text;not null" json:"value"`
	Description string `gorm:"type:text" json:"description"`

	// Type and validation
	ValueType       string `gorm:"type:varchar(32);not null" json:"value_type"` // string, number, boolean, json
	ValidationRules JSONB  `gorm:"type:text" json:"validation_rules,omitempty"`

	// Sensitive data flag
	IsSensitive bool `gorm:"default:false" json:"is_sensitive"`

	// Update information
	UpdatedBy *uuid.UUID `gorm:"type:char(36)" json:"updated_by,omitempty"`
	UpdatedAt time.Time  `gorm:"not null" json:"updated_at"`

	// Associations
	UpdatedByUser *User `gorm:"foreignKey:UpdatedBy" json:"updated_by_user,omitempty"`
}

// TableName overrides the table name
func (SystemConfig) TableName() string {
	return "system_configs"
}

// BeforeSave hook
func (sc *SystemConfig) BeforeSave(tx *gorm.DB) error {
	sc.UpdatedAt = time.Now()
	return nil
}
