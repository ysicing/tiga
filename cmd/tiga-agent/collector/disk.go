package collector

import (
	"github.com/shirou/gopsutil/v3/disk"
)

// DiskCollector collects disk metrics
type DiskCollector struct {
	mountPoint string
}

// NewDiskCollector creates a new disk collector
// mountPoint: "/" for Linux/macOS, "C:" for Windows
func NewDiskCollector(mountPoint string) *DiskCollector {
	if mountPoint == "" {
		mountPoint = "/"
	}
	return &DiskCollector{
		mountPoint: mountPoint,
	}
}

// CollectDisk returns disk statistics for the specified mount point
func (c *DiskCollector) CollectDisk() (*DiskStats, error) {
	stats := &DiskStats{
		MountPoint: c.mountPoint,
	}

	// Disk usage
	usage, err := disk.Usage(c.mountPoint)
	if err != nil {
		return nil, err
	}

	stats.Total = usage.Total
	stats.Used = usage.Used
	stats.Free = usage.Free
	stats.Usage = usage.UsedPercent

	return stats, nil
}

// CollectAllPartitions returns disk statistics for all partitions
func (c *DiskCollector) CollectAllPartitions() ([]*DiskStats, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	var allStats []*DiskStats
	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue // Skip partitions we can't read
		}

		stats := &DiskStats{
			MountPoint: partition.Mountpoint,
			Device:     partition.Device,
			Fstype:     partition.Fstype,
			Total:      usage.Total,
			Used:       usage.Used,
			Free:       usage.Free,
			Usage:      usage.UsedPercent,
		}
		allStats = append(allStats, stats)
	}

	return allStats, nil
}

// DiskStats represents disk statistics
type DiskStats struct {
	MountPoint string
	Device     string
	Fstype     string
	Total      uint64  // Total disk space in bytes
	Used       uint64  // Used disk space in bytes
	Free       uint64  // Free disk space in bytes
	Usage      float64 // Disk usage percentage (0-100)
}

// getTotalDiskSpace returns total disk space for root partition (helper function)
func getTotalDiskSpace() uint64 {
	collector := NewDiskCollector("/")
	stats, err := collector.CollectDisk()
	if err != nil {
		return 0
	}
	return stats.Total
}
