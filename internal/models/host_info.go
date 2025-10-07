package models

import "github.com/google/uuid"

// HostInfo represents static hardware and system information of a host
// This data is reported by Agent once during registration and updated when hardware changes
type HostInfo struct {
	BaseModel

	HostNodeID uuid.UUID `gorm:"type:char(36);uniqueIndex;not null" json:"host_node_id"` // One-to-one with HostNode

	// System information
	Platform        string `json:"platform"`         // Operating system (linux/windows/darwin)
	PlatformVersion string `json:"platform_version"` // OS version (e.g., "Ubuntu 22.04")
	Arch            string `json:"arch"`             // Architecture (amd64/arm64)
	Virtualization  string `json:"virtualization"`   // Virtualization type (kvm/docker/none)

	// Hardware information
	CPUModel  string `json:"cpu_model"`  // CPU model name
	CPUCores  int    `json:"cpu_cores"`  // Number of CPU cores
	MemTotal  uint64 `json:"mem_total"`  // Total memory in bytes
	DiskTotal uint64 `json:"disk_total"` // Total disk space in bytes
	SwapTotal uint64 `json:"swap_total"` // Total swap space in bytes

	// Agent information
	AgentVersion string `json:"agent_version"` // Agent version
	BootTime     uint64 `json:"boot_time"`     // System boot time (Unix timestamp)

	// SSH configuration (reported by Agent)
	SSHEnabled bool   `json:"ssh_enabled"` // Whether SSH service is running
	SSHPort    int    `json:"ssh_port"`    // SSH listening port
	SSHUser    string `json:"ssh_user"`    // Default SSH user

	// GPU information (optional)
	GPUModel string `json:"gpu_model,omitempty"` // GPU model name

	// Relationship
	HostNode *HostNode `gorm:"foreignKey:HostNodeID" json:"-"`
}

// TableName specifies the table name for HostInfo
func (HostInfo) TableName() string {
	return "host_info"
}
