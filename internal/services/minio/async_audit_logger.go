package minio

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/services/audit"

	repo "github.com/ysicing/tiga/internal/repository/minio"
)

// AsyncAuditLogger provides non-blocking audit logging for MinIO operations.
// It wraps the generic audit.AsyncLogger with MinIO-specific interfaces.
type AsyncAuditLogger struct {
	logger *audit.AsyncLogger[*models.MinIOAuditLog]
}

// AsyncAuditLoggerConfig holds configuration for the async audit logger.
type AsyncAuditLoggerConfig = audit.Config

// NewAsyncAuditLogger creates an async audit logger for MinIO.
func NewAsyncAuditLogger(r *repo.AuditRepository, config *AsyncAuditLoggerConfig) *AsyncAuditLogger {
	logger := audit.NewAsyncLogger[*models.MinIOAuditLog](r, "MinIO", config)
	return &AsyncAuditLogger{logger: logger}
}

// LogOperation enqueues a MinIO operation for async audit logging.
func (l *AsyncAuditLogger) LogOperation(
	ctx context.Context,
	instanceID uuid.UUID,
	opType, resType, resName, action, status, message string,
	operatorID *uuid.UUID,
	operatorName, clientIP string,
	details models.JSONB,
) error {
	entry := &models.MinIOAuditLog{
		InstanceID:    instanceID,
		OperationType: opType,
		ResourceType:  resType,
		ResourceName:  resName,
		Action:        action,
		OperatorID:    operatorID,
		OperatorName:  operatorName,
		ClientIP:      clientIP,
		Status:        status,
		ErrorMessage:  message,
		Details:       details,
	}

	return l.logger.Enqueue(entry)
}

// Shutdown gracefully shuts down the async audit logger.
func (l *AsyncAuditLogger) Shutdown(timeout time.Duration) error {
	return l.logger.Shutdown(timeout)
}

// ChannelStatus returns the current channel usage statistics.
func (l *AsyncAuditLogger) ChannelStatus() (used, capacity int) {
	return l.logger.ChannelStatus()
}
