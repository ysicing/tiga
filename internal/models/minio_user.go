package models

import (
	"github.com/google/uuid"
)

// MinIOUser represents a MinIO access user under a specific instance
type MinIOUser struct {
	BaseModel

	InstanceID  uuid.UUID    `gorm:"type:char(36);index;not null" json:"instance_id"`
	Username    string       `gorm:"type:varchar(128);not null" json:"username"`
	AccessKey   string       `gorm:"type:varchar(128);not null" json:"access_key"`
	SecretKey   SecretString `gorm:"type:text;not null" json:"-"`
	Status      string       `gorm:"type:varchar(32);index;not null;default:'enabled'" json:"status"`
	Description string       `gorm:"type:text" json:"description,omitempty"`

	// Relations
	Instance *Instance `gorm:"foreignKey:InstanceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"instance,omitempty"`
}

func (MinIOUser) TableName() string { return "minio_users" }
