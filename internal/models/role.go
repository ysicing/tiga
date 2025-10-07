package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role represents a user role
type Role struct {
	ID          uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	Name        string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"name"`
	DisplayName string    `gorm:"type:varchar(128);not null" json:"display_name"`
	Description string    `gorm:"type:text" json:"description"`

	// Permissions stored as JSONB array of permission objects
	// Format: [{"resource": "instance", "actions": ["create", "read", "update", "delete"]}]
	Permissions JSONB `gorm:"type:text;not null;default:'[]'" json:"permissions"`

	// Type
	IsSystem  bool `gorm:"default:false;index" json:"is_system"` // System predefined role
	IsDefault bool `gorm:"default:false" json:"is_default"`      // Default role for new users

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // Soft delete

	// Associations
	Users []User `gorm:"many2many:user_roles;" json:"users,omitempty"`
}

// TableName overrides the table name
func (Role) TableName() string {
	return "roles"
}

// BeforeCreate hook
func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// UserRole represents the many-to-many relationship between users and roles
type UserRole struct {
	UserID    uuid.UUID  `gorm:"type:char(36);not null;primaryKey" json:"user_id"`
	RoleID    uuid.UUID  `gorm:"type:char(36);not null;primaryKey" json:"role_id"`
	GrantedBy *uuid.UUID `gorm:"type:char(36)" json:"granted_by,omitempty"`
	GrantedAt time.Time  `gorm:"not null" json:"granted_at"`
	ExpiresAt *time.Time `gorm:"index" json:"expires_at,omitempty"`

	// Associations
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Role *Role `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

// TableName overrides the table name
func (UserRole) TableName() string {
	return "user_roles"
}

// IsExpired checks if the role assignment is expired
func (ur *UserRole) IsExpired() bool {
	if ur.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*ur.ExpiresAt)
}
