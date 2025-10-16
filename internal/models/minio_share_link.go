package models

import (
	"time"

	"github.com/google/uuid"
)

// MinIOShareLink records generated share links
type MinIOShareLink struct {
	BaseModel

	InstanceID  uuid.UUID  `gorm:"type:char(36);index;not null" json:"instance_id"`
	BucketName  string     `gorm:"type:varchar(255);index;not null" json:"bucket_name"`
	ObjectKey   string     `gorm:"type:varchar(2048);index;not null" json:"object_key"`
	Token       string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"token"`
	ExpiresAt   time.Time  `gorm:"index;not null" json:"expires_at"`
	Status      string     `gorm:"type:varchar(32);index;not null;default:'active'" json:"status"`
	CreatedBy   *uuid.UUID `gorm:"type:char(36);index" json:"created_by,omitempty"`
	AccessCount int        `gorm:"default:0" json:"access_count"`
}

func (MinIOShareLink) TableName() string { return "minio_share_links" }
