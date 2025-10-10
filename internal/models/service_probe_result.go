package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ServiceProbeResult represents a single probe execution result
type ServiceProbeResult struct {
	BaseModel

	ServiceMonitorID uuid.UUID  `gorm:"type:char(36);index:idx_monitor_timestamp,priority:1;not null" json:"service_monitor_id"`
	HostNodeID       *uuid.UUID `gorm:"type:char(36);index:idx_host_service,priority:1" json:"host_node_id,omitempty"` // Nullable, nil for server-side probes
	Timestamp        time.Time  `gorm:"index:idx_monitor_timestamp,priority:2;index;not null" json:"timestamp"`

	// Probe result
	Success      bool   `gorm:"index" json:"success"`
	Latency      int    `json:"latency"` // Latency in milliseconds
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	// HTTP-specific result
	HTTPStatusCode   int    `json:"http_status_code,omitempty"`
	HTTPResponseBody string `gorm:"type:text" json:"http_response_body,omitempty"` // First 1KB

	// TCP-specific result
	TCPResponse string `gorm:"type:text" json:"tcp_response,omitempty"`

	// Relationships
	ServiceMonitor *ServiceMonitor `gorm:"foreignKey:ServiceMonitorID" json:"-"`
	HostNode       *HostNode       `gorm:"foreignKey:HostNodeID" json:"-"`
}

// TableName specifies the table name for ServiceProbeResult
func (ServiceProbeResult) TableName() string {
	return "service_probe_results"
}

// BeforeCreate sets the timestamp if not provided
func (s *ServiceProbeResult) BeforeCreate(tx *gorm.DB) error {
	// Call BaseModel's BeforeCreate to generate UUID
	if err := s.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}

	if s.Timestamp.IsZero() {
		s.Timestamp = time.Now()
	}
	return nil
}
