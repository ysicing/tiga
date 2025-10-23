package models

import "time"

// Container represents a Docker container (non-persistent, from Agent)
type Container struct {
	ID      string `json:"id"`       // Container ID (12-character short ID)
	LongID  string `json:"long_id"`  // Full container ID
	Name    string `json:"name"`     // Container name
	Image   string `json:"image"`    // Image name (e.g., "nginx:latest")
	ImageID string `json:"image_id"` // Image ID

	// State
	State      string    `json:"state"`       // running, exited, paused, restarting, dead
	Status     string    `json:"status"`      // Human-readable status (e.g., "Up 2 hours")
	CreatedAt  time.Time `json:"created_at"`  // Container creation time
	StartedAt  time.Time `json:"started_at"`  // Last start time
	FinishedAt time.Time `json:"finished_at"` // Last finish time
	ExitCode   int       `json:"exit_code"`   // Exit code if exited

	// Network and ports
	Ports    []ContainerPort    `json:"ports"`
	Networks map[string]Network `json:"networks"`

	// Storage
	Mounts []ContainerMount `json:"mounts"`

	// Resource limits
	CPUShares  int64  `json:"cpu_shares,omitempty"`
	Memory     int64  `json:"memory,omitempty"` // Memory limit in bytes
	MemorySwap int64  `json:"memory_swap,omitempty"`

	// Restart policy
	RestartPolicy string `json:"restart_policy"` // no, always, unless-stopped, on-failure
	RestartCount  int    `json:"restart_count"`

	// Labels and command
	Labels  map[string]string `json:"labels"`
	Command []string          `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     []string          `json:"env,omitempty"`
}

// ContainerPort represents a port mapping
type ContainerPort struct {
	IP          string `json:"ip"`           // Host IP (e.g., "0.0.0.0")
	PrivatePort int    `json:"private_port"` // Container port
	PublicPort  int    `json:"public_port"`  // Host port
	Type        string `json:"type"`         // tcp, udp
}

// ContainerMount represents a volume or bind mount
type ContainerMount struct {
	Type        string `json:"type"`        // volume, bind, tmpfs
	Source      string `json:"source"`      // Source path or volume name
	Destination string `json:"destination"` // Container path
	Mode        string `json:"mode"`        // rw, ro
	RW          bool   `json:"rw"`          // Read-write flag
	Propagation string `json:"propagation,omitempty"`
}

// Network represents a container network
type Network struct {
	NetworkID  string `json:"network_id"`
	IPAddress  string `json:"ip_address"`
	Gateway    string `json:"gateway"`
	MacAddress string `json:"mac_address,omitempty"`
}

// ContainerStats represents real-time container resource usage
type ContainerStats struct {
	ContainerID  string    `json:"container_id"`
	Timestamp    int64     `json:"timestamp"`    // Unix timestamp
	// CPU
	CPUUsagePercent float64 `json:"cpu_usage_percent"` // 0-100% per core
	CPUUsageNano    uint64  `json:"cpu_usage_nano"`
	// Memory
	MemoryUsage        uint64  `json:"memory_usage"`         // Current usage in bytes
	MemoryLimit        uint64  `json:"memory_limit"`         // Limit in bytes
	MemoryUsagePercent float64 `json:"memory_usage_percent"` // 0-100%
	// Network
	NetworkRxBytes uint64 `json:"network_rx_bytes"` // Total received
	NetworkTxBytes uint64 `json:"network_tx_bytes"` // Total sent
	// Block I/O
	BlockReadBytes  uint64 `json:"block_read_bytes"`
	BlockWriteBytes uint64 `json:"block_write_bytes"`
	// PIDs
	PIDsCurrent uint64 `json:"pids_current"`
}

// IsRunning checks if the container is running
func (c *Container) IsRunning() bool {
	return c.State == "running"
}

// IsExited checks if the container has exited
func (c *Container) IsExited() bool {
	return c.State == "exited"
}

// IsPaused checks if the container is paused
func (c *Container) IsPaused() bool {
	return c.State == "paused"
}

// CanStart checks if the container can be started
func (c *Container) CanStart() bool {
	return c.State == "exited" || c.State == "created"
}

// CanStop checks if the container can be stopped
func (c *Container) CanStop() bool {
	return c.State == "running" || c.State == "paused"
}

// CanPause checks if the container can be paused
func (c *Container) CanPause() bool {
	return c.State == "running"
}

// CanUnpause checks if the container can be unpaused
func (c *Container) CanUnpause() bool {
	return c.State == "paused"
}

// GetMainPort returns the main exposed port (first public port)
func (c *Container) GetMainPort() string {
	if len(c.Ports) == 0 {
		return ""
	}
	port := c.Ports[0]
	if port.PublicPort > 0 {
		return string(rune(port.PublicPort))
	}
	return ""
}
