package collector

import (
	"runtime"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
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
}
