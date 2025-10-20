package models

import (
	"time"
)

// ScheduledTask represents a scheduled task configuration
// T022: Data model for scheduled tasks in scheduler system
//
// Reference: .claude/specs/006-gitness-tiga/contracts/scheduler_api.yaml ScheduledTask schema
type ScheduledTask struct {
	UID  string `gorm:"type:varchar(255);primaryKey" json:"uid"`
	Name string `gorm:"type:varchar(255);not null;uniqueIndex:idx_scheduled_tasks_name" json:"name"`
	Type string `gorm:"type:varchar(255);not null;index:idx_scheduled_tasks_type" json:"type"`

	Description string `gorm:"type:text" json:"description,omitempty"`

	// Scheduling configuration
	IsRecurring bool       `gorm:"not null;default:true" json:"is_recurring"`
	CronExpr    string     `gorm:"type:varchar(255)" json:"cron_expr,omitempty"` // For recurring tasks
	Interval    int64      `gorm:"default:0" json:"interval,omitempty"`          // Interval in seconds (for non-cron tasks)
	NextRun     *time.Time `gorm:"index:idx_scheduled_tasks_next_run" json:"next_run,omitempty"`

	// Control flags
	Enabled bool `gorm:"not null;default:true;index:idx_scheduled_tasks_enabled" json:"enabled"`

	// Execution constraints
	MaxDurationSeconds int `gorm:"not null;default:3600" json:"max_duration_seconds"`                      // Maximum execution time
	MaxRetries         int `gorm:"default:0" json:"max_retries,omitempty"`                                 // Maximum retry count
	TimeoutGracePeriod int `gorm:"default:30" json:"timeout_grace_period,omitempty"`                       // Grace period in seconds
	MaxConcurrent      int `gorm:"default:1" json:"max_concurrent,omitempty"`                              // Maximum concurrent executions
	Priority           int `gorm:"default:0;index:idx_scheduled_tasks_priority" json:"priority,omitempty"` // Task priority

	// Metadata
	Labels map[string]string `gorm:"type:text;serializer:json" json:"labels,omitempty"` // Resource labels
	Data   string            `gorm:"type:text" json:"data,omitempty"`                   // Task input data (JSON string)

	// Statistics (computed fields)
	TotalExecutions     int64      `gorm:"default:0" json:"total_executions"`
	SuccessExecutions   int64      `gorm:"default:0" json:"success_executions"`
	FailureExecutions   int64      `gorm:"default:0" json:"failure_executions"`
	ConsecutiveFailures int        `gorm:"default:0" json:"consecutive_failures"`
	LastExecutedAt      *time.Time `json:"last_executed_at,omitempty"`
	LastFailureError    string     `gorm:"type:text" json:"last_failure_error,omitempty"`

	// Timestamps
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

// TableName specifies the table name for ScheduledTask
func (ScheduledTask) TableName() string {
	return "scheduled_tasks"
}

// IncrementExecutions increments execution counters
func (st *ScheduledTask) IncrementExecutions(success bool) {
	st.TotalExecutions++
	if success {
		st.SuccessExecutions++
		st.ConsecutiveFailures = 0
		st.LastFailureError = ""
	} else {
		st.FailureExecutions++
		st.ConsecutiveFailures++
	}
	now := time.Now()
	st.LastExecutedAt = &now
}

// SetLastFailure records the last failure error
func (st *ScheduledTask) SetLastFailure(errorMsg string) {
	st.LastFailureError = errorMsg
}

// IsHealthy checks if the task is in a healthy state
// Considers task unhealthy if consecutive failures exceed threshold
func (st *ScheduledTask) IsHealthy() bool {
	return st.ConsecutiveFailures < 3
}

// CalculateNextRun calculates the next run time based on cron expression or interval
// This should be called after successful execution or when task is enabled
func (st *ScheduledTask) CalculateNextRun(fromTime time.Time) {
	// Implementation will be in scheduler service
	// This is a placeholder for model method
}
