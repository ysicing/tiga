package collector

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/net"
)

// NetworkCollector collects network metrics
type NetworkCollector struct {
	mu              sync.Mutex
	lastIOCounters  []net.IOCountersStat
	lastCollectTime time.Time
}

// NewNetworkCollector creates a new network collector
func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{
		lastCollectTime: time.Now(),
	}
}

// CollectNetwork returns network statistics
func (c *NetworkCollector) CollectNetwork() (*NetworkStats, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stats := &NetworkStats{}
	now := time.Now()

	// Get current network I/O counters
	counters, err := net.IOCounters(false)
	if err != nil {
		return nil, err
	}

	if len(counters) == 0 {
		return stats, nil
	}

	current := counters[0] // Aggregate of all interfaces

	// Total transfer
	stats.NetInTransfer = current.BytesRecv
	stats.NetOutTransfer = current.BytesSent

	// Calculate speed if we have previous data
	if c.lastIOCounters != nil && len(c.lastIOCounters) > 0 {
		last := c.lastIOCounters[0]
		timeDiff := now.Sub(c.lastCollectTime).Seconds()

		if timeDiff > 0 {
			// Bytes per second
			stats.NetInSpeed = uint64(float64(current.BytesRecv-last.BytesRecv) / timeDiff)
			stats.NetOutSpeed = uint64(float64(current.BytesSent-last.BytesSent) / timeDiff)
		}
	}

	// Update last counters
	c.lastIOCounters = counters
	c.lastCollectTime = now

	return stats, nil
}

// CollectConnections returns TCP/UDP connection counts
func (c *NetworkCollector) CollectConnections() (*ConnectionStats, error) {
	stats := &ConnectionStats{}

	// TCP connections
	tcpConns, err := net.Connections("tcp")
	if err == nil {
		stats.TCPCount = uint64(len(tcpConns))
	}

	// UDP connections
	udpConns, err := net.Connections("udp")
	if err == nil {
		stats.UDPCount = uint64(len(udpConns))
	}

	return stats, nil
}

// NetworkStats represents network statistics
type NetworkStats struct {
	NetInTransfer  uint64 // Total bytes received
	NetOutTransfer uint64 // Total bytes sent
	NetInSpeed     uint64 // Receive speed in bytes/sec
	NetOutSpeed    uint64 // Send speed in bytes/sec
}

// ConnectionStats represents connection statistics
type ConnectionStats struct {
	TCPCount uint64 // Number of TCP connections
	UDPCount uint64 // Number of UDP connections
}
