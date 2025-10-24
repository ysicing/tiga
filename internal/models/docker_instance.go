package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DockerInstance represents a Docker daemon instance managed through Agent
type DockerInstance struct {
	BaseModel
	Name        string    `gorm:"not null;index;uniqueIndex:idx_docker_instance_name_agent" json:"name"`
	Description string    `json:"description"`
	AgentID     uuid.UUID `gorm:"type:uuid;not null;index:idx_docker_instance_agent;uniqueIndex:idx_docker_instance_name_agent" json:"agent_id"`
	HostID      uuid.UUID `gorm:"type:uuid;index" json:"host_id,omitempty"` // Optional: associated host node

	// Health and connection status
	HealthStatus    string    `gorm:"not null;index;default:'unknown'" json:"status"` // unknown, online, offline, archived
	LastConnectedAt time.Time `json:"last_connected_at"`
	LastHealthCheck time.Time `json:"last_health_check"`

	// Docker daemon information (fetched from Agent)
	DockerVersion   string `json:"docker_version"`
	APIVersion      string `json:"api_version"`
	MinAPIVersion   string `json:"min_api_version"`
	StorageDriver   string `json:"storage_driver"`
	OperatingSystem string `json:"operating_system"`
	Architecture    string `json:"architecture"`
	KernelVersion   string `json:"kernel_version"`
	MemTotal        int64  `json:"mem_total"`
	NCPU            int    `json:"n_cpu"`

	// Resource statistics (updated by health checks)
	ContainerCount int `gorm:"default:0" json:"container_count"`
	ImageCount     int `gorm:"default:0" json:"image_count"`
	VolumeCount    int `gorm:"default:0" json:"volume_count"`
	NetworkCount   int `gorm:"default:0" json:"network_count"`

	// Metadata
	Tags []string `gorm:"type:jsonb;serializer:json" json:"tags"`
}

// TableName specifies the table name for DockerInstance
func (DockerInstance) TableName() string {
	return "docker_instances"
}

// BeforeCreate hook to set default values
func (d *DockerInstance) BeforeCreate(tx *gorm.DB) error {
	// Call BaseModel's BeforeCreate to generate UUID
	if err := d.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}

	if d.HealthStatus == "" {
		d.HealthStatus = "unknown"
	}
	return nil
}

// IsOnline checks if the instance is currently online
func (d *DockerInstance) IsOnline() bool {
	return d.HealthStatus == "online"
}

// IsOffline checks if the instance is offline
func (d *DockerInstance) IsOffline() bool {
	return d.HealthStatus == "offline"
}

// IsArchived checks if the instance is archived
func (d *DockerInstance) IsArchived() bool {
	return d.HealthStatus == "archived"
}

// CanOperate checks if operations can be performed on this instance
func (d *DockerInstance) CanOperate() bool {
	return d.IsOnline()
}

// MarkOnline marks the instance as online
func (d *DockerInstance) MarkOnline(db *gorm.DB) error {
	now := time.Now()
	return db.Model(d).Updates(map[string]interface{}{
		"health_status":     "online",
		"last_connected_at": now,
		"last_health_check": now,
	}).Error
}

// MarkOffline marks the instance as offline
func (d *DockerInstance) MarkOffline(db *gorm.DB) error {
	return db.Model(d).Updates(map[string]interface{}{
		"health_status":     "offline",
		"last_health_check": time.Now(),
	}).Error
}

// MarkArchived marks the instance as archived (soft delete on Agent deletion)
func (d *DockerInstance) MarkArchived(db *gorm.DB) error {
	return db.Model(d).Update("health_status", "archived").Error
}

// UpdateHealthStatus updates health status and statistics
func (d *DockerInstance) UpdateHealthStatus(db *gorm.DB, status string, containerCount, imageCount, volumeCount, networkCount int) error {
	updates := map[string]interface{}{
		"health_status":     status,
		"last_health_check": time.Now(),
		"container_count":   containerCount,
		"image_count":       imageCount,
		"volume_count":      volumeCount,
		"network_count":     networkCount,
	}
	if status == "online" {
		updates["last_connected_at"] = time.Now()
	}
	return db.Model(d).Updates(updates).Error
}

// UpdateDockerInfo updates Docker daemon information
func (d *DockerInstance) UpdateDockerInfo(db *gorm.DB, info map[string]interface{}) error {
	return db.Model(d).Updates(info).Error
}
