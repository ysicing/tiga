package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID       uuid.UUID  `gorm:"type:char(36);primary_key" json:"id"`
	UserID   *uuid.UUID `gorm:"type:char(36);index" json:"user_id,omitempty"`
	Username string     `gorm:"type:varchar(64)" json:"username"` // Snapshot, prevents loss if user is deleted

	// Cluster context (Phase 4 enhancement)
	ClusterID   *uint  `gorm:"index" json:"cluster_id,omitempty"`
	ClusterName string `gorm:"type:varchar(255)" json:"cluster_name,omitempty"` // Snapshot

	// Operation
	Action       string     `gorm:"type:varchar(64);not null;index" json:"action"`        // create, update, delete, login, logout
	ResourceType string     `gorm:"type:varchar(64);not null;index" json:"resource_type"` // user, instance, role, etc.
	ResourceID   *uuid.UUID `gorm:"type:char(36);index" json:"resource_id,omitempty"`
	ResourceName string     `gorm:"type:varchar(255)" json:"resource_name,omitempty"`

	// Details
	Description string `gorm:"type:text" json:"description"`
	Changes     JSONB  `gorm:"type:text" json:"changes,omitempty"` // Change details

	// Request information
	IPAddress string `gorm:"type:varchar(45)" json:"ip_address,omitempty"` // Changed from inet to varchar(45) for IPv6
	UserAgent string `gorm:"type:text" json:"user_agent,omitempty"`
	RequestID string `gorm:"type:varchar(128)" json:"request_id,omitempty"` // Correlation ID

	// Result
	Status       string `gorm:"type:varchar(32);not null;index" json:"status"` // success, failure
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	// Timestamp
	CreatedAt time.Time `gorm:"index" json:"created_at"`

	// Associations
	User    *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Cluster *Cluster `gorm:"foreignKey:ClusterID" json:"cluster,omitempty"`
}

// TableName overrides the table name
func (AuditLog) TableName() string {
	return "audit_logs"
}

// BeforeCreate hook
func (al *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if al.ID == uuid.Nil {
		al.ID = uuid.New()
	}
	return nil
}

// IsSuccess checks if the operation was successful
func (al *AuditLog) IsSuccess() bool {
	return al.Status == "success"
}
