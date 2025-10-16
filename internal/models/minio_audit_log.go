package models

import (
	"time"

	"github.com/google/uuid"
)

// MinIOAuditLog stores MinIO-specific audit entries
type MinIOAuditLog struct {
	AppendOnlyModel

	InstanceID    uuid.UUID  `gorm:"type:char(36);index;not null" json:"instance_id"`
	OperationType string     `gorm:"type:varchar(64);index;not null" json:"operation_type"`
	ResourceType  string     `gorm:"type:varchar(64);index;not null" json:"resource_type"`
	ResourceName  string     `gorm:"type:varchar(255);index;not null" json:"resource_name"`
	Action        string     `gorm:"type:varchar(64);index;not null" json:"action"`
	OperatorID    *uuid.UUID `gorm:"type:char(36);index" json:"operator_id,omitempty"`
	OperatorName  string     `gorm:"type:varchar(128)" json:"operator_name,omitempty"`
	ClientIP      string     `gorm:"type:varchar(64)" json:"client_ip,omitempty"`
	Status        string     `gorm:"type:varchar(32);index;not null" json:"status"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message,omitempty"`
	Details       JSONB      `gorm:"type:text" json:"details"`
}

func (MinIOAuditLog) TableName() string { return "minio_audit_logs" }

// GetID returns the audit log ID as a string (implements audit.AuditLog interface).
func (l *MinIOAuditLog) GetID() string {
	return l.ID.String()
}

// SetCreatedAt sets the creation timestamp (implements audit.AuditLog interface).
func (l *MinIOAuditLog) SetCreatedAt(t time.Time) {
	l.CreatedAt = t
}
