package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Backup represents a backup record
type Backup struct {
	ID         uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	InstanceID uuid.UUID `gorm:"type:char(36);not null;index" json:"instance_id"`

	// Backup information
	BackupType   string `gorm:"type:varchar(32);not null" json:"backup_type"`   // full, incremental, snapshot
	BackupMethod string `gorm:"type:varchar(32);not null" json:"backup_method"` // mysqldump, pg_dump, redis-rdb, manual

	// Storage
	StorageType string `gorm:"type:varchar(32);not null" json:"storage_type"` // minio, local, s3
	StoragePath string `gorm:"type:text;not null" json:"storage_path"`
	FileSize    *int64 `json:"file_size,omitempty"` // bytes

	// Status
	Status string `gorm:"type:varchar(32);not null;default:'in_progress';index" json:"status"` // in_progress, completed, failed

	// Metadata
	Metadata JSONB  `gorm:"type:text;default:'{}'" json:"metadata"`
	Checksum string `gorm:"type:varchar(128)" json:"checksum,omitempty"` // SHA256

	// Timestamps
	StartedAt   time.Time      `gorm:"not null;index" json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	ExpiresAt   *time.Time     `gorm:"index" json:"expires_at,omitempty"` // Expiration time
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // Soft delete

	// Creator
	CreatedBy *uuid.UUID `gorm:"type:char(36)" json:"created_by,omitempty"`

	// Associations
	Instance *Instance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
	Creator  *User     `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

// TableName overrides the table name
func (Backup) TableName() string {
	return "backups"
}

// BeforeCreate hook
func (b *Backup) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// IsCompleted checks if the backup is completed
func (b *Backup) IsCompleted() bool {
	return b.Status == "completed"
}

// IsFailed checks if the backup has failed
func (b *Backup) IsFailed() bool {
	return b.Status == "failed"
}

// IsExpired checks if the backup has expired
func (b *Backup) IsExpired() bool {
	if b.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*b.ExpiresAt)
}
