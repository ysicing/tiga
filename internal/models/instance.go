package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Instance represents a managed instance (MinIO, MySQL, Redis, Docker, K8s, Caddy, etc.)
type Instance struct {
	ID          uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	Name        string    `gorm:"type:varchar(128);not null" json:"name"`
	DisplayName string    `gorm:"type:varchar(255)" json:"display_name"`
	Description string    `gorm:"type:text" json:"description"`
	Type        string    `gorm:"type:varchar(32);not null;index" json:"type"` // minio, mysql, postgresql, redis, docker, k8s, caddy

	// Connection information (encrypted)
	Connection JSONB `gorm:"type:text;not null" json:"connection"`

	// Status
	Status          string     `gorm:"type:varchar(32);not null;default:'unknown';index" json:"status"` // running, stopped, error, unknown, provisioning
	Health          string     `gorm:"type:varchar(32);not null;default:'unknown';index" json:"health"` // healthy, unhealthy, degraded, unknown
	HealthMessage   string     `gorm:"type:text" json:"health_message,omitempty"`
	LastHealthCheck *time.Time `json:"last_health_check,omitempty"`

	// Version
	Version string `gorm:"type:varchar(64)" json:"version,omitempty"`

	// Classification and labels
	Tags        StringArray `gorm:"type:text" json:"tags"`
	Labels      JSONB       `gorm:"type:text" json:"labels"`
	Environment string      `gorm:"type:varchar(32);index" json:"environment,omitempty"` // dev, staging, prod

	// Configuration
	Config            JSONB `gorm:"type:text" json:"config"`
	HealthCheckConfig JSONB `gorm:"type:text" json:"health_check_config"`

	// Ownership
	OwnerID uuid.UUID `gorm:"type:char(36);not null;index" json:"owner_id"`
	Owner   *User     `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Team    string    `gorm:"type:varchar(64)" json:"team,omitempty"`

	// Metadata
	Metadata JSONB `gorm:"type:text" json:"metadata"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // Soft delete
}

// TableName overrides the table name
func (Instance) TableName() string {
	return "instances"
}

// BeforeCreate hook
func (i *Instance) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

// IsHealthy checks if the instance is healthy
func (i *Instance) IsHealthy() bool {
	return i.Health == "healthy"
}

// IsRunning checks if the instance is running
func (i *Instance) IsRunning() bool {
	return i.Status == "running"
}
