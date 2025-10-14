package scheduler

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	dbrepo "github.com/ysicing/tiga/internal/repository/database"
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

// DatabaseAuditCleanupTask cleans up old database audit logs
type DatabaseAuditCleanupTask struct {
	auditRepo     *dbrepo.AuditLogRepository
	retentionDays int
}

// NewDatabaseAuditCleanupTask creates a new audit cleanup task
func NewDatabaseAuditCleanupTask(auditRepo *dbrepo.AuditLogRepository, retentionDays int) *DatabaseAuditCleanupTask {
	return &DatabaseAuditCleanupTask{
		auditRepo:     auditRepo,
		retentionDays: retentionDays,
	}
}

// Run executes the audit cleanup task
func (t *DatabaseAuditCleanupTask) Run(ctx context.Context) error {
	cutoffDate := time.Now().AddDate(0, 0, -t.retentionDays)

	logrus.Infof("Starting database audit log cleanup (retention: %d days, cutoff: %s)",
		t.retentionDays, cutoffDate.Format("2006-01-02"))

	deleted, err := t.auditRepo.DeleteOldLogs(ctx, cutoffDate)
	if err != nil {
		logrus.Errorf("Failed to cleanup database audit logs: %v", err)
		return err
	}

	logrus.Infof("Database audit log cleanup completed: deleted %d records", deleted)
	return nil
}

// Name returns the task name
func (t *DatabaseAuditCleanupTask) Name() string {
	return "database_audit_cleanup"
}
