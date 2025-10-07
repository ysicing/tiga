package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	Username  string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"type:varchar(255)" json:"-"`
	FullName  string    `gorm:"type:varchar(128)" json:"full_name"`
	AvatarURL string    `gorm:"type:text" json:"avatar_url"`

	Name string `gorm:"-" json:"name,omitempty"` // Maps to FullName at runtime

	// Authentication
	AuthType      string `gorm:"type:varchar(32);not null;default:'local'" json:"auth_type"`
	OAuthProvider string `gorm:"type:varchar(64)" json:"oauth_provider,omitempty"`
	OAuthID       string `gorm:"type:varchar(255)" json:"oauth_id,omitempty"`

	// Legacy compatibility fields
	Provider   string      `gorm:"type:varchar(50);default:'password'" json:"provider,omitempty"`
	Sub        string      `gorm:"type:varchar(255);index" json:"sub,omitempty"`
	OIDCGroups StringArray `gorm:"type:text" json:"oidc_groups,omitempty"`

	// Status
	Status        string `gorm:"type:varchar(32);not null;default:'active'" json:"status"`
	EmailVerified bool   `gorm:"default:false" json:"email_verified"`
	Enabled       bool   `gorm:"type:boolean;default:true" json:"enabled"`
	IsAdmin       bool   `gorm:"default:false" json:"is_admin"` // Simple admin flag for small teams

	// Metadata
	Metadata    JSONB `gorm:"type:text;default:'{}'" json:"metadata"`
	Preferences JSONB `gorm:"type:text;default:'{}'" json:"preferences"`

	// Timestamps
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	LastLoginAt *time.Time     `json:"last_login_at,omitempty"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // Soft delete

	// Associations (database relations)
	Instances []Instance `gorm:"foreignKey:OwnerID" json:"instances,omitempty"`
}

// TableName overrides the table name
func (User) TableName() string {
	return "users"
}

// BeforeCreate hook
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	// Sync legacy fields before create
	u.syncLegacyFields()
	return nil
}

// BeforeSave hook - sync legacy aliased fields to actual fields
func (u *User) BeforeSave(tx *gorm.DB) error {
	u.syncLegacyFields()
	return nil
}

// syncLegacyFields syncs legacy aliased fields to actual database fields
func (u *User) syncLegacyFields() {
	if u.Name != "" && u.FullName == "" {
		u.FullName = u.Name
	}

	if u.FullName != "" {
		u.Name = u.FullName
	}
}

// Key returns a unique identifier for the user (legacy compatibility)
func (u *User) Key() string {
	if u.Username != "" {
		return u.Username
	}
	if u.FullName != "" {
		return u.FullName
	}
	return u.ID.String()
}

// IsActive returns whether the user is active (legacy compatibility)
func (u *User) IsActive() bool {
	return u.Enabled && u.Status == "active" && u.DeletedAt.Time.IsZero()
}
