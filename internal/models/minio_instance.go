package models

import (
	"time"

	"github.com/google/uuid"
)

// MinIOInstance stores MinIO-specific runtime metadata for a generic Instance
// It references the primary instances table via InstanceID
type MinIOInstance struct {
	BaseModel

	InstanceID  uuid.UUID  `gorm:"type:char(36);uniqueIndex;not null" json:"instance_id"`
	Version     string     `gorm:"type:varchar(64)" json:"version,omitempty"`
	TotalSize   int64      `gorm:"default:0" json:"total_size_bytes"`
	UsedSize    int64      `gorm:"default:0" json:"used_size_bytes"`
	LastChecked *time.Time `json:"last_checked,omitempty"`

	Instance *Instance `gorm:"foreignKey:InstanceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"instance,omitempty"`
}

func (MinIOInstance) TableName() string { return "minio_instances" }
