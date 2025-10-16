package models

import (
	"time"

	"github.com/google/uuid"
)

// DatabaseAuditLog records high-level database management operations for compliance tracking.
type DatabaseAuditLog struct {
	AppendOnlyModel

	InstanceID *uuid.UUID        `gorm:"type:char(36);index:idx_db_audit_instance" json:"instance_id,omitempty"`
	Instance   *DatabaseInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`

	Operator   string `gorm:"type:varchar(100);not null;index:idx_db_audit_operator" json:"operator"`
	Action     string `gorm:"type:varchar(50);not null;index:idx_db_audit_action" json:"action"`
	TargetType string `gorm:"type:varchar(50)" json:"target_type"`
	TargetName string `gorm:"type:varchar(255)" json:"target_name"`
	Details    string `gorm:"type:text" json:"details"`

	Success      bool   `gorm:"not null;index:idx_db_audit_success" json:"success"`
	ErrorMessage string `gorm:"type:text" json:"error_msg,omitempty"`
	ClientIP     string `gorm:"type:varchar(50)" json:"client_ip"`
}

// TableName overrides the default table name.
func (DatabaseAuditLog) TableName() string {
	return "db_audit_logs"
}

// GetID returns the audit log ID as a string (implements audit.AuditLog interface).
func (l *DatabaseAuditLog) GetID() string {
	return l.ID.String()
}

// SetCreatedAt sets the creation timestamp (implements audit.AuditLog interface).
func (l *DatabaseAuditLog) SetCreatedAt(t time.Time) {
	l.CreatedAt = t
}
