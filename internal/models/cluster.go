package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Cluster represents a Kubernetes cluster
type Cluster struct {
	ID          uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	Name        string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`

	// Configuration stored as encrypted string
	Config        string `gorm:"type:text" json:"config"` // Encrypted kubeconfig
	PrometheusURL string `gorm:"type:varchar(255)" json:"prometheus_url,omitempty"`

	// Cluster settings
	InCluster bool `gorm:"type:boolean;default:false" json:"in_cluster"`
	IsDefault bool `gorm:"type:boolean;default:false" json:"is_default"`
	Enable    bool `gorm:"type:boolean;default:true" json:"enable"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // Soft delete
}

// TableName overrides the table name
func (Cluster) TableName() string {
	return "clusters"
}

// BeforeCreate hook
func (c *Cluster) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
