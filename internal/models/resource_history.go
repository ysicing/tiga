package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ResourceHistory records Kubernetes resource operation history
type ResourceHistory struct {
	ID        uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	ClusterID uuid.UUID `gorm:"type:char(36);not null;index" json:"cluster_id"`

	// Resource identification
	ResourceType string `gorm:"type:varchar(50);not null;index" json:"resource_type"` // e.g., "Pod", "CloneSet"
	ResourceName string `gorm:"type:varchar(255);not null;index" json:"resource_name"`
	Namespace    string `gorm:"type:varchar(100);index" json:"namespace"`

	// CRD support: API group and version
	APIGroup   string `gorm:"type:varchar(100);index" json:"api_group"`  // e.g., "apps.kruise.io", "" for core
	APIVersion string `gorm:"type:varchar(50);index" json:"api_version"` // e.g., "v1alpha1", "v1"

	// Operation details
	OperationType string `gorm:"type:varchar(50);not null;index" json:"operation_type"` // create, update, delete, apply, scale, restart

	// Resource content
	ResourceYAML string `gorm:"type:text" json:"resource_yaml"`
	PreviousYAML string `gorm:"type:text" json:"previous_yaml,omitempty"`

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

	// Associations
	Cluster  *Cluster `gorm:"foreignKey:ClusterID" json:"cluster,omitempty"`
	Operator *User    `gorm:"foreignKey:OperatorID" json:"operator,omitempty"`
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

// AfterMigrate creates composite indexes for efficient querying
func (ResourceHistory) AfterMigrate(tx *gorm.DB) error {
	// Create composite index for common lookup pattern (by cluster and resource)
	if err := tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_resource_histories_cluster_resource
		ON resource_histories (cluster_id, resource_type, namespace, resource_name, created_at DESC)
	`).Error; err != nil {
		return err
	}

	// Create composite index for CRD lookup (by API group)
	if err := tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_resource_histories_api_group
		ON resource_histories (cluster_id, api_group, api_version, resource_type, created_at DESC)
	`).Error; err != nil {
		return err
	}

	// Create composite index for operation type lookup
	return tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_resource_histories_operation
		ON resource_histories (cluster_id, operation_type, created_at DESC)
	`).Error
}
