package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel contains common fields for all models with soft delete support
// Use this for models that need soft delete functionality
type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:char(36);primary_key" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // Soft delete
}

// BeforeCreate hook to auto-generate UUID
func (m *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// BaseModelWithoutSoftDelete contains common fields without soft delete
// Use this for append-only models like logs, metrics, or audit trails
type BaseModelWithoutSoftDelete struct {
	ID        uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate hook to auto-generate UUID
func (m *BaseModelWithoutSoftDelete) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// AppendOnlyModel contains only creation timestamp
// Use this for immutable records like audit logs or time-series data
type AppendOnlyModel struct {
	ID        uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// BeforeCreate hook to auto-generate UUID
func (m *AppendOnlyModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// Usage examples:
//
// 1. Standard model with soft delete (most common):
//    type User struct {
//        BaseModel
//        Username string `gorm:"uniqueIndex;not null" json:"username"`
//        Email    string `gorm:"uniqueIndex;not null" json:"email"`
//    }
//
// 2. Model without soft delete (sessions, temp data):
//    type Session struct {
//        BaseModelWithoutSoftDelete
//        Token     string    `gorm:"uniqueIndex;not null" json:"token"`
//        ExpiresAt time.Time `json:"expires_at"`
//    }
//
// 3. Append-only model (audit logs, metrics):
//    type AuditLog struct {
//        AppendOnlyModel
//        Action string `gorm:"type:varchar(128);not null" json:"action"`
//        UserID string `gorm:"type:char(36);index" json:"user_id"`
//    }
