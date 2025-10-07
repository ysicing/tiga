package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Session represents a user session
type Session struct {
	ID           uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	UserID       uuid.UUID `gorm:"type:char(36);not null;index" json:"user_id"`
	Token        string    `gorm:"type:varchar(512);uniqueIndex;not null" json:"token"`
	RefreshToken string    `gorm:"type:varchar(512)" json:"refresh_token,omitempty"`

	// Session info
	IPAddress  string `gorm:"type:varchar(45)" json:"ip_address,omitempty"` // Changed from inet to varchar(45) for IPv6
	UserAgent  string `gorm:"type:text" json:"user_agent,omitempty"`
	DeviceType string `gorm:"type:varchar(32)" json:"device_type,omitempty"`

	// Timestamps
	ExpiresAt      time.Time `gorm:"not null;index" json:"expires_at"`
	CreatedAt      time.Time `gorm:"not null" json:"created_at"`
	LastActivityAt time.Time `gorm:"not null" json:"last_activity_at"`

	// Status
	IsActive bool `gorm:"default:true;index" json:"is_active"`

	// Associations
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName overrides the table name
func (Session) TableName() string {
	return "sessions"
}

// BeforeCreate hook
func (s *Session) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// IsExpired checks if the session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid checks if the session is valid (active and not expired)
func (s *Session) IsValid() bool {
	return s.IsActive && !s.IsExpired()
}
