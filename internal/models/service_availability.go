package models

import (
	"time"

	"github.com/google/uuid"
)

// ServiceAvailability represents aggregated availability statistics
type ServiceAvailability struct {
	BaseModel

	ServiceMonitorID uuid.UUID `gorm:"type:char(36);index:idx_monitor_period,priority:1;not null" json:"service_monitor_id"`
	Period           string    `gorm:"index:idx_monitor_period,priority:2;not null" json:"period"` // 1h/24h/7d/30d
	StartTime        time.Time `gorm:"index;not null" json:"start_time"`
	EndTime          time.Time `gorm:"index;not null" json:"end_time"`

	// Statistics
	TotalChecks      int     `json:"total_checks"`
	SuccessfulChecks int     `json:"successful_checks"`
	FailedChecks     int     `json:"failed_checks"`
	UptimePercentage float64 `json:"uptime_percentage"`
	AvgLatency       float64 `json:"avg_latency"` // Average latency in milliseconds
	MinLatency       int     `json:"min_latency"`
	MaxLatency       int     `json:"max_latency"`
	DowntimeSeconds  int     `json:"downtime_seconds"` // Total downtime in seconds

	// Relationship
	ServiceMonitor *ServiceMonitor `gorm:"foreignKey:ServiceMonitorID" json:"-"`
}

// TableName specifies the table name for ServiceAvailability
func (ServiceAvailability) TableName() string {
	return "service_availability"
}

// CalculateUptime calculates the uptime percentage
func (s *ServiceAvailability) CalculateUptime() {
	if s.TotalChecks > 0 {
		s.UptimePercentage = float64(s.SuccessfulChecks) / float64(s.TotalChecks) * 100
	} else {
		s.UptimePercentage = 0
	}
}
