package models

import (
	"time"

	"github.com/google/uuid"
)

// DatabaseUser represents a database-level user account tied to a managed instance.
type DatabaseUser struct {
	BaseModel

	InstanceID uuid.UUID         `gorm:"type:char(36);not null;index:idx_db_user_instance,priority:1;index:idx_db_user_instance_username,priority:1,unique" json:"instance_id"`
	Instance   *DatabaseInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`

	Username    string `gorm:"type:varchar(100);not null;index:idx_db_user_instance_username,priority:2,unique" json:"username"`
	Password    string `gorm:"type:text" json:"-"` // encrypted credential blob
	Host        string `gorm:"type:varchar(255);default:'%'" json:"host"`
	Description string `gorm:"type:varchar(500)" json:"description"`

	IsActive    bool       `gorm:"not null;default:true;index:idx_db_user_active" json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`

	Permissions []PermissionPolicy `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName overrides the default table name.
func (DatabaseUser) TableName() string {
	return "db_users"
}
