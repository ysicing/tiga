package scheduler

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/alert"
	"github.com/ysicing/tiga/internal/services/docker"
	"github.com/ysicing/tiga/internal/services/host"
	"github.com/ysicing/tiga/internal/services/k8s"
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

// DatabaseAuditCleanupTask cleans up old audit events
// T036-T037: 更新为使用统一的 AuditEventRepository
type DatabaseAuditCleanupTask struct {
	auditRepo     repository.AuditEventRepository
	retentionDays int
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
		return err
	}

	logrus.Infof("Audit event cleanup completed: deleted %d records", deleted)
	return nil
}

// Name returns the task name
func (t *DatabaseAuditCleanupTask) Name() string {
	return "database_audit_cleanup"
}

// HostExpiryCheckTask checks host expiry dates and generates alerts
type HostExpiryCheckTask struct {
	expiryScheduler *host.ExpiryScheduler
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
	t.expiryScheduler.CheckAllHosts()
	return nil
}

// Name returns the task name
func (t *HostExpiryCheckTask) Name() string {
	return "host_expiry_check"
}

// ClusterHealthCheckTask checks Kubernetes cluster health status
type ClusterHealthCheckTask struct {
	healthService *k8s.ClusterHealthService
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
	return t.healthService.CheckAll(ctx)
}

// Name returns the task name
func (t *ClusterHealthCheckTask) Name() string {
	return "cluster_health_check"
}

// DockerHealthCheckTask checks Docker instance health status (T030)
type DockerHealthCheckTask struct {
	healthService *docker.DockerHealthService
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
		return err
	}

	logrus.WithField("duration", duration.String()).Debug("Docker health check task completed")
	return nil
}

// Name returns the task name
func (t *DockerHealthCheckTask) Name() string {
	return "docker_health_check"
}

// DockerAuditCleanupTask cleans up old Docker audit logs (T031)
type DockerAuditCleanupTask struct {
	auditRepo     repository.AuditLogRepositoryInterface
	retentionDays int
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
		return err
	}

	logrus.WithField("deleted_count", deleted).Info("Docker audit log cleanup completed")
	return nil
}

// Name returns the task name
func (t *DockerAuditCleanupTask) Name() string {
	return "docker_audit_cleanup"
}
