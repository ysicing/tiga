package models

import (
	"time"

	"github.com/google/uuid"
)

// Metric represents a time-series metric data point
type Metric struct {
	Time       time.Time `gorm:"not null;index:idx_metrics_time" json:"time"`
	InstanceID uuid.UUID `gorm:"type:char(36);not null;index:idx_metrics_instance_id" json:"instance_id"`

	// Metric information
	MetricName string `gorm:"type:varchar(128);not null;index:idx_metrics_metric_name" json:"metric_name"`
	MetricType string `gorm:"type:varchar(32);not null" json:"metric_type"` // gauge, counter, histogram

	// Value
	Value float64 `json:"value"`

	// Labels
	Labels JSONB `gorm:"type:text" json:"labels"`

	// Aggregated values (optional)
	MinValue *float64 `json:"min_value,omitempty"`
	MaxValue *float64 `json:"max_value,omitempty"`
	AvgValue *float64 `json:"avg_value,omitempty"`
	SumValue *float64 `json:"sum_value,omitempty"`
	Count    *int     `json:"count,omitempty"`

	// Associations
	Instance *Instance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
}

// TableName overrides the table name
func (Metric) TableName() string {
	return "metrics"
}
