package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Health status constants
const (
	ClusterHealthUnknown     = "unknown"
	ClusterHealthHealthy     = "healthy"
	ClusterHealthWarning     = "warning"
	ClusterHealthError       = "error"
	ClusterHealthUnavailable = "unavailable"
)

// Cluster represents a Kubernetes cluster
type Cluster struct {
	ID          uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	Name        string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`

	// Configuration stored as encrypted string
	Config        string `gorm:"type:text" json:"config"` // Encrypted kubeconfig
	PrometheusURL string `gorm:"type:varchar(512)" json:"prometheus_url,omitempty"`

	// Cluster settings
	InCluster bool `gorm:"type:boolean;default:false" json:"in_cluster"`
	IsDefault bool `gorm:"type:boolean;default:false" json:"is_default"`
	Enable    bool `gorm:"type:boolean;default:true" json:"enable"`

	// Health and Statistics (added in Phase 0)
	HealthStatus    string     `gorm:"column:health_status;type:varchar(20);default:'unknown';index" json:"health_status"`
	LastConnectedAt *time.Time `gorm:"column:last_connected_at" json:"last_connected_at,omitempty"`
	NodeCount       int        `gorm:"column:node_count;default:0" json:"node_count"`
	PodCount        int        `gorm:"column:pod_count;default:0" json:"pod_count"`

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
