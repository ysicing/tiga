package database

import (
	"context"
	"fmt"
	"time"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/services/audit"

	dbrepo "github.com/ysicing/tiga/internal/repository/database"
)

// AsyncAuditLogger provides non-blocking audit logging with batching for database operations.
// It wraps the generic audit.AsyncLogger with database-specific interfaces.
type AsyncAuditLogger struct {
	logger *audit.AsyncLogger[*models.DatabaseAuditLog]
}

// AsyncAuditLoggerConfig holds configuration for the async audit logger.
type AsyncAuditLoggerConfig = audit.Config

// NewAsyncAuditLogger creates an async audit logger with the given configuration.
func NewAsyncAuditLogger(repo *dbrepo.AuditLogRepository, config *AsyncAuditLoggerConfig) *AsyncAuditLogger {
	logger := audit.NewAsyncLogger[*models.DatabaseAuditLog](repo, "Database", config)
	return &AsyncAuditLogger{logger: logger}
}

// LogAction enqueues an audit entry for async processing.
// This method is non-blocking and returns immediately.
func (l *AsyncAuditLogger) LogAction(ctx context.Context, entry AuditEntry) error {
	if entry.Operator == "" {
		return fmt.Errorf("operator is required")
	}
	if entry.Action == "" {
		return fmt.Errorf("action is required")
	}

	clientIP := entry.ClientIP
	if clientIP == "" {
		clientIP = ExtractClientIP(ctx)
	}

	details, err := marshalDetails(entry.Details)
	if err != nil {
		return err
	}

	log := &models.DatabaseAuditLog{
		Operator:   entry.Operator,
		Action:     entry.Action,
		TargetType: entry.TargetType,
		TargetName: entry.TargetName,
		Details:    details,
		Success:    entry.Success,
		ClientIP:   clientIP,
	}
	if entry.InstanceID != nil {
		log.InstanceID = entry.InstanceID
	}
	if entry.Error != nil {
		log.ErrorMessage = entry.Error.Error()
	}

	return l.logger.Enqueue(log)
}

// Shutdown gracefully shuts down the async audit logger.
func (l *AsyncAuditLogger) Shutdown(timeout time.Duration) error {
	return l.logger.Shutdown(timeout)
}

// ChannelStatus returns the current channel usage statistics.
func (l *AsyncAuditLogger) ChannelStatus() (used, capacity int) {
	return l.logger.ChannelStatus()
}
