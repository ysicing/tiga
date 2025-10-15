package models

import (
	"github.com/google/uuid"
)

// BucketPermission represents a permission granted to a MinIO user for a bucket or prefix
type BucketPermission struct {
	BaseModel

	InstanceID  uuid.UUID  `gorm:"type:char(36);index;not null" json:"instance_id"`
	UserID      uuid.UUID  `gorm:"type:char(36);index;not null" json:"user_id"`
	BucketName  string     `gorm:"type:varchar(255);index;not null" json:"bucket_name"`
	Prefix      string     `gorm:"type:varchar(1024);index" json:"prefix"`
	Permission  string     `gorm:"type:varchar(16);index;not null" json:"permission"` // readonly, writeonly, readwrite
	GrantedBy   *uuid.UUID `gorm:"type:char(36);index" json:"granted_by,omitempty"`
	Description string     `gorm:"type:text" json:"description,omitempty"`

	User *MinIOUser `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user,omitempty"`
}

func (BucketPermission) TableName() string { return "bucket_permissions" }
