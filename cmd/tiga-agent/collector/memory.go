package collector

import (
	"github.com/shirou/gopsutil/v3/mem"
)

// MemoryCollector collects memory metrics
type MemoryCollector struct{}

// NewMemoryCollector creates a new memory collector
func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{}
}

// CollectMemory returns memory statistics
func (c *MemoryCollector) CollectMemory() (*MemoryStats, error) {
	stats := &MemoryStats{}

	// Virtual memory
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	stats.MemTotal = vmStat.Total
	stats.MemUsed = vmStat.Used
	stats.MemUsage = vmStat.UsedPercent

	// Swap memory
	swapStat, err := mem.SwapMemory()
	if err != nil {
		return nil, err
	}

	stats.SwapTotal = swapStat.Total
	stats.SwapUsed = swapStat.Used
	stats.SwapUsage = swapStat.UsedPercent

	return stats, nil
}

// MemoryStats represents memory statistics
type MemoryStats struct {
	MemTotal  uint64  // Total memory in bytes
	MemUsed   uint64  // Used memory in bytes
	MemUsage  float64 // Memory usage percentage (0-100)
	SwapTotal uint64  // Total swap in bytes
	SwapUsed  uint64  // Used swap in bytes
	SwapUsage float64 // Swap usage percentage (0-100)
}
