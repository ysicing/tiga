package database

import (
	"context"

	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/audit"
)

// AuditLogger provides synchronous audit logging by wrapping AsyncAuditLogger
// Deprecated: Use AsyncAuditLogger directly for better performance
type AuditLogger struct {
	async *AsyncAuditLogger
}

// NewAuditLogger constructs a synchronous AuditLogger
// Deprecated: Use NewAsyncAuditLogger instead
func NewAuditLogger(repo repository.AuditEventRepository) *AuditLogger {
	config := audit.DefaultConfig()
	return &AuditLogger{
		async: NewAsyncAuditLogger(repo, config),
	}
}

// LogAction logs an audit entry synchronously
func (l *AuditLogger) LogAction(ctx context.Context, entry AuditEntry) error {
	return l.async.LogAction(ctx, entry)
}
