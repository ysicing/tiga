package collector

import (
	"sync"

	"github.com/shirou/gopsutil/v3/net"
)

// TrafficCollector tracks network traffic statistics
type TrafficCollector struct {
	lastStats map[string]net.IOCountersStat
	mu        sync.RWMutex
}

// NewTrafficCollector creates a new traffic collector
func NewTrafficCollector() *TrafficCollector {
	return &TrafficCollector{
		lastStats: make(map[string]net.IOCountersStat),
	}
}

// TrafficStats represents traffic statistics
type TrafficStats struct {
	// Total bytes since system boot
	BytesSent uint64
	BytesRecv uint64

	// Delta since last collection
	DeltaSent uint64
	DeltaRecv uint64
}

// CollectTraffic collects network traffic statistics
func (c *TrafficCollector) CollectTraffic() (*TrafficStats, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get current network IO stats
	ioCounters, err := net.IOCounters(false)
	if err != nil {
		return nil, err
	}

	if len(ioCounters) == 0 {
		return nil, nil
	}

	// Use the first (aggregated) counter
	current := ioCounters[0]

	stats := &TrafficStats{
		BytesSent: current.BytesSent,
		BytesRecv: current.BytesRecv,
	}

	// Calculate delta if we have previous stats
	if last, exists := c.lastStats[current.Name]; exists {
		// Handle counter overflow (wraparound)
		if current.BytesSent >= last.BytesSent {
			stats.DeltaSent = current.BytesSent - last.BytesSent
		}
		if current.BytesRecv >= last.BytesRecv {
			stats.DeltaRecv = current.BytesRecv - last.BytesRecv
		}
	}

	// Store current stats for next delta calculation
	c.lastStats[current.Name] = current

	return stats, nil
}

// Reset resets the traffic collector state
func (c *TrafficCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastStats = make(map[string]net.IOCountersStat)
}
