package models

import (
	"time"

	"github.com/google/uuid"
)

// PermissionPolicy defines the access granted to a database user for a specific database.
type PermissionPolicy struct {
	BaseModel

	UserID     uuid.UUID     `gorm:"type:char(36);not null;index:idx_db_permission_user" json:"user_id"`
	User       *DatabaseUser `gorm:"foreignKey:UserID" json:"user,omitempty"`
	DatabaseID uuid.UUID     `gorm:"type:char(36);not null;index:idx_db_permission_database" json:"database_id"`
	Database   *Database     `gorm:"foreignKey:DatabaseID" json:"database,omitempty"`

	Role      string     `gorm:"type:varchar(20);not null;index:idx_db_permission_role" json:"role"` // readonly|readwrite
	GrantedBy string     `gorm:"type:varchar(100)" json:"granted_by"`
	GrantedAt time.Time  `gorm:"not null" json:"granted_at"`
	RevokedAt *time.Time `gorm:"index" json:"revoked_at,omitempty"`
}

// TableName overrides the default table name.
func (PermissionPolicy) TableName() string {
	return "db_permissions"
}
