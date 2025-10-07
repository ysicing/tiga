package collector

import (
	"os"
	"runtime"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

// SystemCollector collects system information for HostInfo
type SystemCollector struct {
	agentVersion string
}

// NewSystemCollector creates a new system information collector
func NewSystemCollector(agentVersion string) *SystemCollector {
	return &SystemCollector{
		agentVersion: agentVersion,
	}
}

// CollectHostInfo collects static host information
func (c *SystemCollector) CollectHostInfo() (*HostInfo, error) {
	info := &HostInfo{
		AgentVersion: c.agentVersion,
	}

	// Platform and OS information
	hostInfo, err := host.Info()
	if err != nil {
		return nil, err
	}

	info.Platform = hostInfo.Platform
	info.PlatformVersion = hostInfo.PlatformVersion
	info.Kernel = hostInfo.KernelVersion
	info.Arch = runtime.GOARCH
	info.Virtualization = hostInfo.VirtualizationSystem
	info.BootTime = hostInfo.BootTime

	// CPU information
	info.CPUModel = getCPUModel()
	info.CPUCores = uint32(runtime.NumCPU())
	info.CPUThreads = uint32(runtime.GOMAXPROCS(0))

	// Memory information
	vmStat, err := mem.VirtualMemory()
	if err == nil {
		info.MemTotal = vmStat.Total
	}

	// Swap information
	swapStat, err := mem.SwapMemory()
	if err == nil {
		info.SwapTotal = swapStat.Total
	}

	// Disk information (root filesystem)
	info.DiskTotal = getTotalDiskSpace()

	// SSH configuration detection
	sshInfo := detectSSHConfig()
	info.SSHEnabled = sshInfo.Enabled
	info.SSHPort = sshInfo.Port
	info.SSHUser = sshInfo.DefaultUser

	return info, nil
}

// CollectUptime returns system uptime in seconds
func (c *SystemCollector) CollectUptime() (uint64, error) {
	uptime, err := host.Uptime()
	if err != nil {
		return 0, err
	}
	return uptime, nil
}

// CollectBootTime returns system boot time as Unix timestamp
func (c *SystemCollector) CollectBootTime() (uint64, error) {
	bootTime, err := host.BootTime()
	if err != nil {
		return 0, err
	}
	return bootTime, nil
}

// HostInfo represents static host information
type HostInfo struct {
	Platform        string
	PlatformVersion string
	Kernel          string
	Arch            string
	Virtualization  string
	CPUModel        string
	CPUCores        uint32
	CPUThreads      uint32
	MemTotal        uint64
	DiskTotal       uint64
	SwapTotal       uint64
	AgentVersion    string
	BootTime        uint64
	GPUModel        string // Optional

	// SSH configuration (detected by Agent)
	SSHEnabled  bool
	SSHPort     int
	SSHUser     string
}

// SSHInfo represents SSH service detection result
type SSHInfo struct {
	Enabled     bool
	Port        int
	DefaultUser string
}

// detectSSHConfig detects if SSH service is running and its configuration
func detectSSHConfig() SSHInfo {
	info := SSHInfo{
		Enabled:     false,
		Port:        22,
		DefaultUser: "root",
	}

	// Check if sshd process is running
	processes, err := process.Processes()
	if err != nil {
		return info
	}

	for _, proc := range processes {
		name, err := proc.Name()
		if err != nil {
			continue
		}
		// Check for sshd (Unix/Linux) or sshd.exe (Windows)
		if name == "sshd" || name == "sshd.exe" {
			info.Enabled = true
			break
		}
	}

	// If SSH is not running, return disabled
	if !info.Enabled {
		return info
	}

	// Try to detect SSH port from common config files
	// Default to 22 if can't detect
	info.Port = detectSSHPort()

	// Detect default user based on OS
	info.DefaultUser = detectDefaultSSHUser()

	return info
}

// detectSSHPort tries to detect SSH port from sshd_config
func detectSSHPort() int {
	// Common sshd_config locations
	configPaths := []string{
		"/etc/ssh/sshd_config",
		"/usr/local/etc/ssh/sshd_config",
	}

	for _, path := range configPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Simple parsing - look for "Port" directive
		lines := string(data)
		// This is a simplified parser, production code should be more robust
		_ = lines
		// For now, just return default port
		// TODO: Implement proper config parsing
	}

	return 22 // Default SSH port
}

// detectDefaultSSHUser detects the default SSH user based on OS
func detectDefaultSSHUser() string {
	switch runtime.GOOS {
	case "linux":
		return "root"
	case "darwin":
		// macOS typically uses the current user
		if user := os.Getenv("USER"); user != "" {
			return user
		}
		return "root"
	case "windows":
		return "Administrator"
	default:
		return "root"
	}
}
