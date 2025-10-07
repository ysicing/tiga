package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OAuthProvider represents an OAuth provider configuration
type OAuthProvider struct {
	ID          uuid.UUID `gorm:"type:char(36);primary_key" json:"id"`
	Name        string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"name"`
	DisplayName string    `gorm:"type:varchar(128);not null" json:"display_name"`
	Type        string    `gorm:"type:varchar(32);not null" json:"type"` // oauth2, oidc, ldap, saml

	// OAuth configuration
	ClientID     string `gorm:"type:varchar(255);not null" json:"client_id"`
	ClientSecret string `gorm:"type:varchar(512);not null" json:"-"` // Encrypted
	AuthURL      string `gorm:"type:text;not null" json:"auth_url"`
	TokenURL     string `gorm:"type:text;not null" json:"token_url"`
	UserInfoURL  string `gorm:"type:text" json:"user_info_url,omitempty"`
	Scopes       string `gorm:"type:text" json:"scopes"`                   // Comma-separated for compatibility
	Issuer       string `gorm:"type:varchar(255)" json:"issuer,omitempty"` // OIDC issuer

	// Additional configuration
	Config JSONB `gorm:"type:text;default:'{}'" json:"config"`

	// Auto-generated redirect URL (runtime only)
	RedirectURL string `gorm:"-" json:"-"`

	// Status
	Enabled  bool `gorm:"default:true;index" json:"enabled"`
	IsSystem bool `gorm:"default:false" json:"is_system"` // System predefined

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // Soft delete
}

// TableName overrides the table name
func (OAuthProvider) TableName() string {
	return "oauth_providers"
}

// BeforeCreate hook
func (o *OAuthProvider) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}
