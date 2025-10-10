package monitor

import (
	"github.com/google/uuid"
)

// ServiceResponseItem represents aggregated 30-day statistics for a service monitor
type ServiceResponseItem struct {
	ServiceMonitorID uuid.UUID `json:"service_monitor_id"`
	ServiceName      string    `json:"service_name"`

	// 30-day data arrays (index 0 = today, 29 = 30 days ago)
	Delay [30]float32 `json:"delay"` // Average delay per day (ms)
	Up    [30]uint64  `json:"up"`    // Successful checks per day
	Down  [30]uint64  `json:"down"`  // Failed checks per day

	// Aggregated statistics
	TotalUp          uint64  `json:"total_up"`          // Total successful checks (30 days)
	TotalDown        uint64  `json:"total_down"`        // Total failed checks (30 days)
	UptimePercentage float64 `json:"uptime_percentage"` // Overall uptime % (30 days)

	// Current status (from statusToday)
	CurrentUp   uint64 `json:"current_up"`   // Today's successful checks
	CurrentDown uint64 `json:"current_down"` // Today's failed checks

	// Status code: Good(>95%) / LowAvailability(80-95%) / Down(<80%)
	StatusCode string `json:"status_code"`
}

// ServiceOverviewResponse represents the complete overview response
type ServiceOverviewResponse struct {
	Services map[string]*ServiceResponseItem `json:"services"` // Key: service_monitor_id
}
