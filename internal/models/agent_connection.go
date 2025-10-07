package models

import (
	"time"

	"gorm.io/gorm"
)

// AgentConnectionStatus represents the connection status of an agent
type AgentConnectionStatus string

const (
	AgentStatusOnline      AgentConnectionStatus = "online"
	AgentStatusOffline     AgentConnectionStatus = "offline"
	AgentStatusConnecting  AgentConnectionStatus = "connecting"
	AgentStatusDisconnected AgentConnectionStatus = "disconnected"
)

// AgentConnection represents the connection state and metadata of a connected agent
type AgentConnection struct {
	gorm.Model

	HostNodeID uint `gorm:"uniqueIndex;not null" json:"host_node_id"` // One-to-one with HostNode

	// Connection status
	Status       AgentConnectionStatus `gorm:"index;not null" json:"status"`
	ConnectedAt  *time.Time            `json:"connected_at,omitempty"`
	LastHeartbeat time.Time            `gorm:"index;not null" json:"last_heartbeat"`

	// Agent information
	AgentVersion string `json:"agent_version"`
	IPAddress    string `json:"ip_address"`

	// Connection statistics
	HeartbeatCount   int64  `gorm:"default:0" json:"heartbeat_count"`
	ReconnectCount   int    `gorm:"default:0" json:"reconnect_count"`
	LastDisconnectAt *time.Time `json:"last_disconnect_at,omitempty"`
	DisconnectReason string `json:"disconnect_reason,omitempty"`

	// Performance metrics
	AvgLatency       int `json:"avg_latency"` // Average heartbeat latency in milliseconds
	LastLatency      int `json:"last_latency"`
	PacketLoss       float64 `json:"packet_loss"` // Packet loss percentage

	// Relationship
	HostNode *HostNode `gorm:"foreignKey:HostNodeID" json:"-"`
}

// TableName specifies the table name for AgentConnection
func (AgentConnection) TableName() string {
	return "agent_connections"
}

// BeforeCreate sets initial values
func (a *AgentConnection) BeforeCreate(tx *gorm.DB) error {
	if a.LastHeartbeat.IsZero() {
		a.LastHeartbeat = time.Now()
	}
	if a.Status == "" {
		a.Status = AgentStatusConnecting
	}
	return nil
}

// UpdateHeartbeat updates the last heartbeat time and status
func (a *AgentConnection) UpdateHeartbeat() {
	a.LastHeartbeat = time.Now()
	a.Status = AgentStatusOnline
	a.HeartbeatCount++
}

// MarkOffline marks the connection as offline
func (a *AgentConnection) MarkOffline(reason string) {
	now := time.Now()
	a.Status = AgentStatusOffline
	a.LastDisconnectAt = &now
	a.DisconnectReason = reason
}

// IsOnline checks if the agent is currently online
func (a *AgentConnection) IsOnline() bool {
	return a.Status == AgentStatusOnline
}

// IsStale checks if the heartbeat is stale (older than 90 seconds)
func (a *AgentConnection) IsStale() bool {
	return time.Since(a.LastHeartbeat) > 90*time.Second
}
