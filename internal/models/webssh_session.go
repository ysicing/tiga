package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WebSSHSession represents an active or historical WebSSH session
type WebSSHSession struct {
	BaseModel

	SessionID string `gorm:"uniqueIndex;not null" json:"session_id"` // Unique session identifier

	// Session details
	UserID     uuid.UUID `gorm:"type:char(36);index;not null" json:"user_id"`
	HostNodeID uuid.UUID `gorm:"type:char(36);index;not null" json:"host_node_id"`
	ClientIP   string    `json:"client_ip"`

	// Terminal configuration
	Cols int `json:"cols"`
	Rows int `json:"rows"`

	// Session lifecycle
	Status     string     `gorm:"index;not null" json:"status"` // active/closed
	StartTime  time.Time  `gorm:"not null" json:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	LastActive time.Time  `gorm:"index" json:"last_active"`

	// Audit information
	SSHUser      string `json:"ssh_user"`
	SSHPort      int    `json:"ssh_port"`
	CommandCount int    `gorm:"default:0" json:"command_count"` // Number of commands executed
	AuditLog     string `gorm:"type:text" json:"-"`             // Full terminal log (encrypted)
	CloseReason  string `json:"close_reason,omitempty"`

	// Relationships
	HostNode *HostNode `gorm:"foreignKey:HostNodeID" json:"-"`
	User     *User     `gorm:"foreignKey:UserID" json:"-"`
}

// TableName specifies the table name for WebSSHSession
func (WebSSHSession) TableName() string {
	return "webssh_sessions"
}

// BeforeCreate sets initial values
func (w *WebSSHSession) BeforeCreate(tx *gorm.DB) error {
	if w.StartTime.IsZero() {
		w.StartTime = time.Now()
	}
	if w.LastActive.IsZero() {
		w.LastActive = time.Now()
	}
	if w.Status == "" {
		w.Status = "active"
	}
	return nil
}

// IsActive checks if the session is still active
func (w *WebSSHSession) IsActive() bool {
	return w.Status == "active" && w.EndTime == nil
}

// Close marks the session as closed
func (w *WebSSHSession) Close(reason string) {
	now := time.Now()
	w.Status = "closed"
	w.EndTime = &now
	w.CloseReason = reason
}
