package collector

import (
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// Collector aggregates all metrics collectors
type Collector struct {
	system  *SystemCollector
	cpu     *CPUCollector
	memory  *MemoryCollector
	disk    *DiskCollector
	network *NetworkCollector
	traffic *TrafficCollector
}

// NewCollector creates a new aggregated collector
func NewCollector(agentVersion string) *Collector {
	return &Collector{
		system:  NewSystemCollector(agentVersion),
		cpu:     NewCPUCollector(),
		memory:  NewMemoryCollector(),
		disk:    NewDiskCollector("/"),
		network: NewNetworkCollector(),
		traffic: NewTrafficCollector(),
	}
}

// CollectHostInfo collects static host information
func (c *Collector) CollectHostInfo() (*HostInfo, error) {
	return c.system.CollectHostInfo()
}

// CollectHostState collects real-time host metrics
func (c *Collector) CollectHostState() (*HostState, error) {
	state := &HostState{
		Timestamp: time.Now(),
	}

	// CPU metrics
	cpuStats, err := c.cpu.CollectAll()
	if err != nil {
		return nil, err
	}
	state.CPUUsage = cpuStats.Usage
	state.Load1 = cpuStats.Load1
	state.Load5 = cpuStats.Load5
	state.Load15 = cpuStats.Load15

	// Memory metrics
	memStats, err := c.memory.CollectMemory()
	if err != nil {
		return nil, err
	}
	state.MemUsed = memStats.MemUsed
	state.MemUsage = memStats.MemUsage
	state.SwapUsed = memStats.SwapUsed

	// Disk metrics
	diskStats, err := c.disk.CollectDisk()
	if err != nil {
		return nil, err
	}
	state.DiskUsed = diskStats.Used
	state.DiskUsage = diskStats.Usage

	// Network metrics
	netStats, err := c.network.CollectNetwork()
	if err != nil {
		return nil, err
	}
	state.NetInTransfer = netStats.NetInTransfer
	state.NetOutTransfer = netStats.NetOutTransfer
	state.NetInSpeed = netStats.NetInSpeed
	state.NetOutSpeed = netStats.NetOutSpeed

	// Connection metrics
	connStats, err := c.network.CollectConnections()
	if err == nil {
		state.TCPConnCount = connStats.TCPCount
		state.UDPConnCount = connStats.UDPCount
	}

	// Process count
	processes, err := process.Processes()
	if err == nil {
		state.ProcessCount = uint64(len(processes))
	}

	// System uptime
	uptime, err := c.system.CollectUptime()
	if err == nil {
		state.Uptime = uptime
	}

	// Traffic statistics
	trafficStats, err := c.traffic.CollectTraffic()
	if err == nil && trafficStats != nil {
		state.TrafficSent = trafficStats.BytesSent
		state.TrafficRecv = trafficStats.BytesRecv
		state.TrafficDeltaSent = trafficStats.DeltaSent
		state.TrafficDeltaRecv = trafficStats.DeltaRecv
	}

	return state, nil
}

// HostState represents real-time host monitoring metrics
type HostState struct {
	Timestamp time.Time

	// CPU and Load
	CPUUsage float64
	Load1    float64
	Load5    float64
	Load15   float64

	// Memory
	MemUsed  uint64
	MemUsage float64
	SwapUsed uint64

	// Disk
	DiskUsed  uint64
	DiskUsage float64

	// Network
	NetInTransfer  uint64
	NetOutTransfer uint64
	NetInSpeed     uint64
	NetOutSpeed    uint64

	// Connections and Processes
	TCPConnCount uint64
	UDPConnCount uint64
	ProcessCount uint64

	// System
	Uptime uint64

	// Traffic (累计和增量)
	TrafficSent      uint64 // 总发送字节数
	TrafficRecv      uint64 // 总接收字节数
	TrafficDeltaSent uint64 // 本次上报周期发送字节数
	TrafficDeltaRecv uint64 // 本次上报周期接收字节数

	// Optional
	Temperatures string  // JSON-encoded temperature sensors
	GPUUsage     float64 // GPU usage percentage
}
