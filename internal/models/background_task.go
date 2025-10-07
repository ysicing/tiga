package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BackgroundTask represents a background task
type BackgroundTask struct {
	ID         uuid.UUID  `gorm:"type:char(36);primary_key" json:"id"`
	TaskType   string     `gorm:"type:varchar(64);not null;index" json:"task_type"` // backup, health_check, metric_collect
	TaskName   string     `gorm:"type:varchar(128)" json:"task_name,omitempty"`
	InstanceID *uuid.UUID `gorm:"type:char(36);index" json:"instance_id,omitempty"`

	// Task data
	Payload JSONB `gorm:"type:text;default:'{}'" json:"payload"`

	// Status
	Status   string `gorm:"type:varchar(32);not null;default:'pending';index" json:"status"` // pending, running, completed, failed, cancelled
	Progress int    `gorm:"default:0" json:"progress"`                                       // 0-100

	// Result
	Result       JSONB  `gorm:"type:text" json:"result,omitempty"`
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	// Retry
	RetryCount int `gorm:"default:0" json:"retry_count"`
	MaxRetries int `gorm:"default:3" json:"max_retries"`

	// Schedule
	ScheduledAt *time.Time `gorm:"index" json:"scheduled_at,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Timestamps
	CreatedAt time.Time `gorm:"index" json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Associations
	Instance *Instance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
}

// TableName overrides the table name
func (BackgroundTask) TableName() string {
	return "background_tasks"
}

// BeforeCreate hook
func (bt *BackgroundTask) BeforeCreate(tx *gorm.DB) error {
	if bt.ID == uuid.Nil {
		bt.ID = uuid.New()
	}
	return nil
}

// IsCompleted checks if the task is completed
func (bt *BackgroundTask) IsCompleted() bool {
	return bt.Status == "completed"
}

// IsFailed checks if the task has failed
func (bt *BackgroundTask) IsFailed() bool {
	return bt.Status == "failed"
}

// CanRetry checks if the task can be retried
func (bt *BackgroundTask) CanRetry() bool {
	return bt.IsFailed() && bt.RetryCount < bt.MaxRetries
}

// MarkRunning marks the task as running
func (bt *BackgroundTask) MarkRunning(tx *gorm.DB) error {
	now := time.Now()
	bt.Status = "running"
	bt.StartedAt = &now
	return tx.Model(bt).Updates(map[string]interface{}{
		"status":     "running",
		"started_at": now,
	}).Error
}

// MarkCompleted marks the task as completed
func (bt *BackgroundTask) MarkCompleted(tx *gorm.DB, result JSONB) error {
	now := time.Now()
	bt.Status = "completed"
	bt.Progress = 100
	bt.CompletedAt = &now
	bt.Result = result
	return tx.Model(bt).Updates(map[string]interface{}{
		"status":       "completed",
		"progress":     100,
		"completed_at": now,
		"result":       result,
	}).Error
}

// MarkFailed marks the task as failed
func (bt *BackgroundTask) MarkFailed(tx *gorm.DB, errorMsg string) error {
	now := time.Now()
	bt.Status = "failed"
	bt.CompletedAt = &now
	bt.ErrorMessage = errorMsg
	bt.RetryCount++
	return tx.Model(bt).Updates(map[string]interface{}{
		"status":        "failed",
		"completed_at":  now,
		"error_message": errorMsg,
		"retry_count":   bt.RetryCount,
	}).Error
}
