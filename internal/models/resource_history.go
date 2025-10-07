package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ResourceHistory records Kubernetes resource operation history
type ResourceHistory struct {
	ID          uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	ClusterName string    `gorm:"type:varchar(100);not null;index" json:"cluster_name"`

	// Resource identification
	ResourceType string `gorm:"type:varchar(50);not null;index" json:"resource_type"`
	ResourceName string `gorm:"type:varchar(255);not null;index" json:"resource_name"`
	Namespace    string `gorm:"type:varchar(100);index" json:"namespace"`

	// Operation details
	OperationType string `gorm:"type:varchar(50);not null;index" json:"operation_type"` // create, update, delete, apply

	// Resource content
	ResourceYAML string `gorm:"type:text" json:"resource_yaml"`
	PreviousYAML string `gorm:"type:text" json:"previous_yaml"`

	// Operation result
	Success      bool   `gorm:"type:boolean;default:true" json:"success"`
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	// Operator information
	OperatorID   uuid.UUID `gorm:"type:char(36);not null;index" json:"operator_id"`
	OperatorName string    `gorm:"type:varchar(255)" json:"operator_name"` // Cached operator name

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // Soft delete

	// Association
	Operator *User `gorm:"foreignKey:OperatorID" json:"operator,omitempty"`
}

// TableName overrides the table name
func (ResourceHistory) TableName() string {
	return "resource_histories"
}

// BeforeCreate hook
func (rh *ResourceHistory) BeforeCreate(tx *gorm.DB) error {
	if rh.ID == uuid.Nil {
		rh.ID = uuid.New()
	}
	return nil
}

// AfterMigrate creates composite index for efficient querying
func (ResourceHistory) AfterMigrate(tx *gorm.DB) error {
	// Create composite index for common lookup pattern
	return tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_resource_histories_lookup
		ON resource_histories (cluster_name, resource_type, resource_name, namespace, created_at DESC)
	`).Error
}
