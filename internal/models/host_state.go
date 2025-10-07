package models

import (
	"time"

	"gorm.io/gorm"
)

// HostState represents real-time monitoring metrics snapshot
// Updated every 30 seconds by Agent
type HostState struct {
	gorm.Model

	HostNodeID uint      `gorm:"index:idx_host_timestamp,priority:1;not null" json:"host_node_id"`
	Timestamp  time.Time `gorm:"index:idx_host_timestamp,priority:2;index;not null" json:"timestamp"`

	// CPU and Load
	CPUUsage float64 `json:"cpu_usage"` // CPU usage percentage
	Load1    float64 `json:"load_1"`    // 1-minute load average
	Load5    float64 `json:"load_5"`    // 5-minute load average
	Load15   float64 `json:"load_15"`   // 15-minute load average

	// Memory
	MemUsed  uint64  `json:"mem_used"`  // Used memory in bytes
	MemUsage float64 `json:"mem_usage"` // Memory usage percentage
	SwapUsed uint64  `json:"swap_used"` // Used swap in bytes

	// Disk
	DiskUsed  uint64  `json:"disk_used"`  // Used disk in bytes
	DiskUsage float64 `json:"disk_usage"` // Disk usage percentage

	// Network
	NetInTransfer  uint64 `json:"net_in_transfer"`  // Total inbound traffic in bytes
	NetOutTransfer uint64 `json:"net_out_transfer"` // Total outbound traffic in bytes
	NetInSpeed     uint64 `json:"net_in_speed"`     // Inbound speed in bytes/sec
	NetOutSpeed    uint64 `json:"net_out_speed"`    // Outbound speed in bytes/sec

	// Connections and Processes
	TCPConnCount uint64 `json:"tcp_conn_count"` // Number of TCP connections
	UDPConnCount uint64 `json:"udp_conn_count"` // Number of UDP connections
	ProcessCount uint64 `json:"process_count"`  // Number of processes

	// System Uptime
	Uptime uint64 `json:"uptime"` // System uptime in seconds

	// Traffic statistics
	TrafficSent      uint64 `json:"traffic_sent"`       // Total bytes sent
	TrafficRecv      uint64 `json:"traffic_recv"`       // Total bytes received
	TrafficDeltaSent uint64 `json:"traffic_delta_sent"` // Delta bytes sent
	TrafficDeltaRecv uint64 `json:"traffic_delta_recv"` // Delta bytes received

	// Temperature and GPU (optional)
	Temperatures string  `gorm:"type:text" json:"temperatures,omitempty"` // Temperature sensors (JSON array)
	GPUUsage     float64 `json:"gpu_usage,omitempty"`                     // GPU usage percentage

	// Relationship
	HostNode *HostNode `gorm:"foreignKey:HostNodeID" json:"-"`
}

// TableName specifies the table name for HostState
func (HostState) TableName() string {
	return "host_states"
}

// BeforeCreate sets the timestamp if not provided
func (h *HostState) BeforeCreate(tx *gorm.DB) error {
	if h.Timestamp.IsZero() {
		h.Timestamp = time.Now()
	}
	return nil
}
