package scheduler

import (
	"context"

	"github.com/ysicing/tiga/internal/services/alert"
)

// AlertTask runs alert processing
type AlertTask struct {
	processor *alert.AlertProcessor
}

// NewAlertTask creates a new alert task
func NewAlertTask(processor *alert.AlertProcessor) *AlertTask {
	return &AlertTask{
		processor: processor,
	}
}

// Run executes the alert task
func (t *AlertTask) Run(ctx context.Context) error {
	return t.processor.ProcessAlerts(ctx)
}

// Name returns the task name
func (t *AlertTask) Name() string {
	return "alert_processing"
}
