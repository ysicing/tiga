package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// HostActivityLog records host-related activities and events
type HostActivityLog struct {
	BaseModel

	HostNodeID uuid.UUID  `gorm:"type:char(36);index;not null" json:"host_node_id"`
	UserID     *uuid.UUID `gorm:"type:char(36);index" json:"user_id,omitempty"` // Optional: for user-initiated actions

	// Activity details
	Action      string `gorm:"type:varchar(100);not null;index" json:"action"`     // terminal_created, terminal_closed, agent_connected, agent_disconnected, node_edited, etc.
	ActionType  string `gorm:"type:varchar(50);not null;index" json:"action_type"` // terminal, agent, system, user
	Description string `gorm:"type:text" json:"description"`

	// Additional context (JSON)
	Metadata string `gorm:"type:text" json:"metadata,omitempty"` // JSON string for additional data (session_id, etc.)

	// IP and User-Agent for user actions
	ClientIP  string `json:"client_ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`

	// Timestamp
	CreatedAt time.Time `gorm:"not null;index" json:"created_at"`

	// Relationships
	HostNode *HostNode `gorm:"foreignKey:HostNodeID" json:"-"`
	User     *User     `gorm:"foreignKey:UserID" json:"-"`
}

// TableName specifies the table name for HostActivityLog
func (HostActivityLog) TableName() string {
	return "host_activity_logs"
}

// BeforeCreate sets initial values
func (h *HostActivityLog) BeforeCreate(tx *gorm.DB) error {
	// Call BaseModel's BeforeCreate to generate UUID
	if err := h.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}

	if h.CreatedAt.IsZero() {
		h.CreatedAt = time.Now()
	}
	return nil
}

// Activity action constants
const (
	// Terminal actions
	ActivityTerminalCreated = "terminal_created"
	ActivityTerminalClosed  = "terminal_closed"
	ActivityTerminalReplay  = "terminal_replay"

	// Agent actions
	ActivityAgentConnected    = "agent_connected"
	ActivityAgentDisconnected = "agent_disconnected"
	ActivityAgentReconnected  = "agent_reconnected"

	// Node management actions
	ActivityNodeCreated = "node_created"
	ActivityNodeUpdated = "node_updated"
	ActivityNodeDeleted = "node_deleted"

	// System actions
	ActivitySystemAlert = "system_alert"
	ActivitySystemError = "system_error"
)

// Activity type constants
const (
	ActivityTypeTerminal = "terminal"
	ActivityTypeAgent    = "agent"
	ActivityTypeSystem   = "system"
	ActivityTypeUser     = "user"
)
