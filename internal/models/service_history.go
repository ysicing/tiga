package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ServiceHistory stores aggregated service probe results by day
// This table accumulates daily statistics for each (ServiceMonitor, HostNode) pair
// and maintains 30-day historical data for trend analysis
type ServiceHistory struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"index:idx_service_host_time" json:"created_at"`

	// ServiceMonitorID is the ID of the service being monitored
	ServiceMonitorID uuid.UUID `gorm:"type:char(36);not null;index:idx_service_host_time" json:"service_monitor_id"`

	// HostNodeID is the ID of the host that executed this probe
	// A value of 00000000-0000-0000-0000-000000000000 (uuid.Nil) represents aggregated summary data from all nodes
	HostNodeID uuid.UUID `gorm:"type:char(36);not null;index:idx_service_host_time" json:"host_node_id"`

	// AvgDelay is the average response time in milliseconds for this time period
	AvgDelay float32 `gorm:"index:idx_service_host_time" json:"avg_delay"`

	// Up is the count of successful probes during this time period
	Up uint64 `gorm:"not null;default:0" json:"up"`

	// Down is the count of failed probes during this time period
	Down uint64 `gorm:"not null;default:0" json:"down"`

	// Data stores additional probe-specific information:
	// - For HTTPS probes: TLS certificate issuer and expiration date (format: "Issuer|ExpiryDate")
	// - For failed probes: Error messages or diagnostic information
	Data string `gorm:"type:text" json:"data,omitempty"`

	// Relationships
	ServiceMonitor *ServiceMonitor `gorm:"foreignKey:ServiceMonitorID" json:"-"`
	HostNode       *HostNode       `gorm:"foreignKey:HostNodeID" json:"-"`
}

// TableName specifies the table name for ServiceHistory
func (ServiceHistory) TableName() string {
	return "service_histories"
}

// BeforeCreate generates UUID before creating the record
func (s *ServiceHistory) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// UptimePercent calculates the uptime percentage for this history record
func (s *ServiceHistory) UptimePercent() float64 {
	total := s.Up + s.Down
	if total == 0 {
		return 0
	}
	return float64(s.Up) * 100.0 / float64(total)
}

// IsHealthy determines if the service is considered healthy based on uptime
// Uses the same thresholds as Nezha:
// - >= 95%: Good
// - >= 80%: Low Availability
// - < 80%: Down
func (s *ServiceHistory) IsHealthy() bool {
	return s.UptimePercent() >= 95.0
}

// HealthStatus returns a human-readable health status
func (s *ServiceHistory) HealthStatus() string {
	uptime := s.UptimePercent()
	if uptime >= 95.0 {
		return "Good"
	} else if uptime >= 80.0 {
		return "LowAvailability"
	}
	return "Down"
}

// AfterMigrate creates additional indexes after migration
func (ServiceHistory) AfterMigrate(db *gorm.DB) error {
	// Check database type to use appropriate syntax
	var dbType string
	db.Raw("SELECT sqlite_version()").Scan(&dbType)

	// For SQLite, create indexes manually
	if dbType != "" {
		// Create covering index for common queries
		// This index covers the most frequent query pattern: filter by host, time range, order by created_at
		if err := db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_host_created_desc
			ON service_histories(host_node_id, created_at DESC)
			WHERE host_node_id != '00000000-0000-0000-0000-000000000000'
		`).Error; err != nil {
			return err
		}

		// Create index for summary data queries (where host_node_id = uuid.Nil)
		if err := db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_summary_created
			ON service_histories(created_at DESC, service_monitor_id)
			WHERE host_node_id = '00000000-0000-0000-0000-000000000000'
		`).Error; err != nil {
			return err
		}
	} else {
		// For PostgreSQL/MySQL, use standard syntax
		// PostgreSQL partial index for non-summary data
		db.Exec(`
			CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_host_created_desc
			ON service_histories(host_node_id, created_at DESC)
			WHERE host_node_id != '00000000-0000-0000-0000-000000000000'
		`)

		// PostgreSQL partial index for summary data
		db.Exec(`
			CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_summary_created
			ON service_histories(created_at DESC, service_monitor_id)
			WHERE host_node_id = '00000000-0000-0000-0000-000000000000'
		`)
	}

	return nil
}
