package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/alert"
	"github.com/ysicing/tiga/internal/services/docker"
	"github.com/ysicing/tiga/internal/services/host"
	"github.com/ysicing/tiga/internal/services/k8s"
)

// ResultProvider is an optional interface that tasks can implement
// to provide detailed execution results that will be stored in TaskExecution.Result
type ResultProvider interface {
	GetResult() string
}

// AlertTask runs alert processing
type AlertTask struct {
	processor  *alert.AlertProcessor
	lastResult string // Store last execution result for ResultProvider
}

// NewAlertTask creates a new alert task
func NewAlertTask(processor *alert.AlertProcessor) *AlertTask {
	return &AlertTask{
		processor: processor,
	}
}

// Run executes the alert task
func (t *AlertTask) Run(ctx context.Context) error {
	start := time.Now()

	err := t.processor.ProcessAlerts(ctx)

	duration := time.Since(start)
	if err != nil {
		t.lastResult = fmt.Sprintf("Alert processing failed after %s: %v", duration.Round(time.Millisecond), err)
		return err
	}

	// Store result for ResultProvider interface
	t.lastResult = fmt.Sprintf("Alert processing completed successfully in %s", duration.Round(time.Millisecond))
	return nil
}

// Name returns the task name
func (t *AlertTask) Name() string {
	return "alert_processing"
}

// GetResult implements ResultProvider interface
// Returns detailed statistics about the alert processing operation
func (t *AlertTask) GetResult() string {
	return t.lastResult
}

// DatabaseAuditCleanupTask cleans up old audit events
// T036-T037: 更新为使用统一的 AuditEventRepository
type DatabaseAuditCleanupTask struct {
	auditRepo     repository.AuditEventRepository
	retentionDays int
	lastResult    string // Store last execution result for ResultProvider
}

// NewDatabaseAuditCleanupTask creates a new audit cleanup task
// T036-T037: 使用统一的 AuditEventRepository
func NewDatabaseAuditCleanupTask(auditRepo repository.AuditEventRepository, retentionDays int) *DatabaseAuditCleanupTask {
	return &DatabaseAuditCleanupTask{
		auditRepo:     auditRepo,
		retentionDays: retentionDays,
	}
}

// Run executes the audit cleanup task
func (t *DatabaseAuditCleanupTask) Run(ctx context.Context) error {
	cutoffDate := time.Now().AddDate(0, 0, -t.retentionDays)

	logrus.Infof("Starting audit event cleanup (retention: %d days, cutoff: %s)",
		t.retentionDays, cutoffDate.Format("2006-01-02"))

	deleted, err := t.auditRepo.DeleteOlderThan(ctx, cutoffDate)
	if err != nil {
		logrus.Errorf("Failed to cleanup audit events: %v", err)
		t.lastResult = fmt.Sprintf("Failed to cleanup audit events: %v", err)
		return err
	}

	// Store result for ResultProvider interface
	t.lastResult = fmt.Sprintf("Successfully cleaned up %d audit events older than %d days (cutoff date: %s)",
		deleted, t.retentionDays, cutoffDate.Format("2006-01-02"))

	logrus.Infof("Audit event cleanup completed: deleted %d records", deleted)
	return nil
}

// Name returns the task name
func (t *DatabaseAuditCleanupTask) Name() string {
	return "database_audit_cleanup"
}

// GetResult implements ResultProvider interface
// Returns detailed statistics about the audit cleanup operation
func (t *DatabaseAuditCleanupTask) GetResult() string {
	return t.lastResult
}

// HostExpiryCheckTask checks host expiry dates and generates alerts
type HostExpiryCheckTask struct {
	expiryScheduler *host.ExpiryScheduler
	lastResult      string // Store last execution result for ResultProvider
}

// NewHostExpiryCheckTask creates a new host expiry check task
func NewHostExpiryCheckTask(expiryScheduler *host.ExpiryScheduler) *HostExpiryCheckTask {
	return &HostExpiryCheckTask{
		expiryScheduler: expiryScheduler,
	}
}

// Run executes the host expiry check
func (t *HostExpiryCheckTask) Run(ctx context.Context) error {
	logrus.Debug("Running host expiry check task")
	start := time.Now()

	t.expiryScheduler.CheckAllHosts()

	duration := time.Since(start)

	// Store result for ResultProvider interface
	t.lastResult = fmt.Sprintf("Host expiry check completed successfully in %s", duration.Round(time.Millisecond))
	return nil
}

// Name returns the task name
func (t *HostExpiryCheckTask) Name() string {
	return "host_expiry_check"
}

// GetResult implements ResultProvider interface
// Returns detailed statistics about the host expiry check operation
func (t *HostExpiryCheckTask) GetResult() string {
	return t.lastResult
}

// ClusterHealthCheckTask checks Kubernetes cluster health status
type ClusterHealthCheckTask struct {
	healthService *k8s.ClusterHealthService
	lastResult    string // Store last execution result for ResultProvider
}

// NewClusterHealthCheckTask creates a new cluster health check task
func NewClusterHealthCheckTask(healthService *k8s.ClusterHealthService) *ClusterHealthCheckTask {
	return &ClusterHealthCheckTask{
		healthService: healthService,
	}
}

// Run executes the cluster health check
func (t *ClusterHealthCheckTask) Run(ctx context.Context) error {
	logrus.Debug("Running cluster health check task")
	start := time.Now()

	err := t.healthService.CheckAll(ctx)

	duration := time.Since(start)
	if err != nil {
		t.lastResult = fmt.Sprintf("Cluster health check failed after %s: %v", duration.Round(time.Millisecond), err)
		return err
	}

	// Store result for ResultProvider interface
	t.lastResult = fmt.Sprintf("Cluster health check completed successfully in %s", duration.Round(time.Millisecond))
	return nil
}

// Name returns the task name
func (t *ClusterHealthCheckTask) Name() string {
	return "cluster_health_check"
}

// GetResult implements ResultProvider interface
// Returns detailed statistics about the cluster health check operation
func (t *ClusterHealthCheckTask) GetResult() string {
	return t.lastResult
}

// DockerHealthCheckTask checks Docker instance health status (T030)
type DockerHealthCheckTask struct {
	healthService *docker.DockerHealthService
	lastResult    string // Store last execution result for ResultProvider
}

// NewDockerHealthCheckTask creates a new Docker health check task
func NewDockerHealthCheckTask(healthService *docker.DockerHealthService) *DockerHealthCheckTask {
	return &DockerHealthCheckTask{
		healthService: healthService,
	}
}

// Run executes the Docker health check
func (t *DockerHealthCheckTask) Run(ctx context.Context) error {
	logrus.Debug("Running Docker instance health check task")
	start := time.Now()

	err := t.healthService.CheckAllInstances(ctx)

	duration := time.Since(start)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"duration": duration.String(),
			"error":    err.Error(),
		}).Error("Docker health check task failed")
		t.lastResult = fmt.Sprintf("Docker health check failed after %s: %v", duration.Round(time.Millisecond), err)
		return err
	}

	// Store result for ResultProvider interface
	t.lastResult = fmt.Sprintf("Docker health check completed successfully in %s", duration.Round(time.Millisecond))

	logrus.WithField("duration", duration.String()).Debug("Docker health check task completed")
	return nil
}

// Name returns the task name
func (t *DockerHealthCheckTask) Name() string {
	return "docker_health_check"
}

// GetResult implements ResultProvider interface
// Returns detailed statistics about the Docker health check operation
func (t *DockerHealthCheckTask) GetResult() string {
	return t.lastResult
}

// DockerAuditCleanupTask cleans up old Docker audit logs (T031)
type DockerAuditCleanupTask struct {
	auditRepo     repository.AuditLogRepositoryInterface
	retentionDays int
	lastResult    string // Store last execution result for ResultProvider
}

// NewDockerAuditCleanupTask creates a new Docker audit cleanup task
func NewDockerAuditCleanupTask(auditRepo repository.AuditLogRepositoryInterface, retentionDays int) *DockerAuditCleanupTask {
	if retentionDays <= 0 {
		retentionDays = 90 // Default 90 days retention
	}
	return &DockerAuditCleanupTask{
		auditRepo:     auditRepo,
		retentionDays: retentionDays,
	}
}

// Run executes the Docker audit log cleanup
func (t *DockerAuditCleanupTask) Run(ctx context.Context) error {
	cutoffDate := time.Now().AddDate(0, 0, -t.retentionDays)

	logrus.WithFields(logrus.Fields{
		"retention_days": t.retentionDays,
		"cutoff_date":    cutoffDate.Format("2006-01-02"),
	}).Info("Starting Docker audit log cleanup")

	// Delete all audit logs older than retention period
	// This will include Docker operations (container, image) and other operations
	deleted, err := t.auditRepo.DeleteOldLogs(ctx, cutoffDate)
	if err != nil {
		logrus.WithError(err).Error("Failed to cleanup Docker audit logs")
		t.lastResult = fmt.Sprintf("Failed to cleanup Docker audit logs: %v", err)
		return err
	}

	// Store result for ResultProvider interface
	t.lastResult = fmt.Sprintf("Successfully cleaned up %d Docker audit log entries older than %d days (cutoff date: %s)",
		deleted, t.retentionDays, cutoffDate.Format("2006-01-02"))

	logrus.WithField("deleted_count", deleted).Info("Docker audit log cleanup completed")
	return nil
}

// Name returns the task name
func (t *DockerAuditCleanupTask) Name() string {
	return "docker_audit_cleanup"
}

// GetResult implements ResultProvider interface
// Returns detailed statistics about the Docker audit cleanup operation
func (t *DockerAuditCleanupTask) GetResult() string {
	return t.lastResult
}

// TerminalRecordingCleanupTask cleans up old terminal recordings
type TerminalRecordingCleanupTask struct {
	recordingRepo repository.TerminalRecordingRepositoryInterface
	retentionDays int
	lastResult    string // Store last execution result for ResultProvider
}

// NewTerminalRecordingCleanupTask creates a new terminal recording cleanup task
func NewTerminalRecordingCleanupTask(recordingRepo repository.TerminalRecordingRepositoryInterface, retentionDays int) *TerminalRecordingCleanupTask {
	if retentionDays <= 0 {
		retentionDays = 90 // Default 90 days retention
	}
	return &TerminalRecordingCleanupTask{
		recordingRepo: recordingRepo,
		retentionDays: retentionDays,
	}
}

// Run executes the terminal recording cleanup
func (t *TerminalRecordingCleanupTask) Run(ctx context.Context) error {
	cutoffDate := time.Now().AddDate(0, 0, -t.retentionDays)

	logrus.WithFields(logrus.Fields{
		"retention_days": t.retentionDays,
		"cutoff_date":    cutoffDate.Format("2006-01-02"),
	}).Info("Starting terminal recording cleanup")

	// Delete recordings older than retention period
	deleted, err := t.recordingRepo.DeleteOlderThan(ctx, cutoffDate)
	if err != nil {
		logrus.WithError(err).Error("Failed to cleanup terminal recordings")
		t.lastResult = fmt.Sprintf("Failed to cleanup recordings: %v", err)
		return err
	}

	// Store result for ResultProvider interface
	t.lastResult = fmt.Sprintf("Successfully cleaned up %d terminal recordings older than %d days (cutoff date: %s)",
		deleted, t.retentionDays, cutoffDate.Format("2006-01-02"))

	logrus.WithField("deleted_count", deleted).Info("Terminal recording cleanup completed")
	return nil
}

// Name returns the task name
func (t *TerminalRecordingCleanupTask) Name() string {
	return "terminal_recording_cleanup"
}

// GetResult implements ResultProvider interface
// Returns detailed statistics about the cleanup operation
func (t *TerminalRecordingCleanupTask) GetResult() string {
	return t.lastResult
}
