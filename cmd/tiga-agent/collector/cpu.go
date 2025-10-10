package collector

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
)

// CPUCollector collects CPU metrics
type CPUCollector struct{}

// NewCPUCollector creates a new CPU collector
func NewCPUCollector() *CPUCollector {
	return &CPUCollector{}
}

// CollectCPUUsage returns CPU usage percentage (0-100)
func (c *CPUCollector) CollectCPUUsage() (float64, error) {
	// Get CPU usage over 1 second interval
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, err
	}

	if len(percentages) == 0 {
		return 0, nil
	}

	return percentages[0], nil
}

// CollectLoadAverage returns system load averages (1, 5, 15 minutes)
func (c *CPUCollector) CollectLoadAverage() (load1, load5, load15 float64, err error) {
	loadAvg, err := load.Avg()
	if err != nil {
		return 0, 0, 0, err
	}

	return loadAvg.Load1, loadAvg.Load5, loadAvg.Load15, nil
}

// CPUStats represents CPU statistics
type CPUStats struct {
	Usage  float64
	Load1  float64
	Load5  float64
	Load15 float64
}

// CollectAll collects all CPU metrics
func (c *CPUCollector) CollectAll() (*CPUStats, error) {
	stats := &CPUStats{}

	// CPU usage
	usage, err := c.CollectCPUUsage()
	if err != nil {
		return nil, err
	}
	stats.Usage = usage

	// Load average
	load1, load5, load15, err := c.CollectLoadAverage()
	if err != nil {
		return nil, err
	}
	stats.Load1 = load1
	stats.Load5 = load5
	stats.Load15 = load15

	return stats, nil
}

// getCPUModel returns the CPU model name (helper function)
func getCPUModel() string {
	info, err := cpu.Info()
	if err != nil || len(info) == 0 {
		return "Unknown"
	}
	return info[0].ModelName
}
