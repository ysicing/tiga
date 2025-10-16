package audit

import (
	"context"
	"time"
)

// AuditLog represents a generic audit log entry that can be persisted.
// Both DatabaseAuditLog and MinIOAuditLog should implement this interface.
type AuditLog interface {
	// GetID returns the unique identifier of the audit log
	GetID() string
	// SetCreatedAt sets the timestamp when the log was created
	SetCreatedAt(t time.Time)
}

// AuditRepository defines the interface for persisting audit logs.
// This allows AsyncAuditLogger to work with different repository implementations.
type AuditRepository[T AuditLog] interface {
	// Create persists a single audit log entry
	Create(ctx context.Context, log T) error
	// CreateBatch persists multiple audit log entries in a single transaction (optional optimization)
	CreateBatch(ctx context.Context, logs []T) error
}

// Config holds configuration for the async audit logger.
type Config struct {
	// ChannelBuffer is the size of the channel buffer (default: 1000)
	ChannelBuffer int
	// BatchSize is the maximum number of logs to write in one transaction (default: 50)
	BatchSize int
	// FlushPeriod is how often to flush pending logs even if batch is not full (default: 5s)
	FlushPeriod time.Duration
	// WorkerCount is the number of worker goroutines (default: 2)
	WorkerCount int
}

// DefaultConfig returns sensible defaults for the audit logger.
func DefaultConfig() *Config {
	return &Config{
		ChannelBuffer: 1000,
		BatchSize:     50,
		FlushPeriod:   5 * time.Second,
		WorkerCount:   2,
	}
}

// ApplyDefaults fills in zero values with defaults.
func (c *Config) ApplyDefaults() {
	if c.ChannelBuffer <= 0 {
		c.ChannelBuffer = 1000
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 50
	}
	if c.FlushPeriod <= 0 {
		c.FlushPeriod = 5 * time.Second
	}
	if c.WorkerCount <= 0 {
		c.WorkerCount = 2
	}
}
