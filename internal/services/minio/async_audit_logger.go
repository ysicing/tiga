package minio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"

	repo "github.com/ysicing/tiga/internal/repository/minio"
)

// AsyncAuditLogger provides non-blocking audit logging for MinIO operations.
type AsyncAuditLogger struct {
	repo        *repo.AuditRepository
	entryChan   chan *models.MinIOAuditLog
	batchSize   int
	flushPeriod time.Duration
	workerCount int
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	stopped     bool
	mu          sync.Mutex
}

// AsyncAuditLoggerConfig holds configuration for the async audit logger.
type AsyncAuditLoggerConfig struct {
	ChannelBuffer int
	BatchSize     int
	FlushPeriod   time.Duration
	WorkerCount   int
}

// NewAsyncAuditLogger creates an async audit logger for MinIO.
func NewAsyncAuditLogger(r *repo.AuditRepository, config *AsyncAuditLoggerConfig) *AsyncAuditLogger {
	if config == nil {
		config = &AsyncAuditLoggerConfig{}
	}
	if config.ChannelBuffer <= 0 {
		config.ChannelBuffer = 1000
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 50
	}
	if config.FlushPeriod <= 0 {
		config.FlushPeriod = 5 * time.Second
	}
	if config.WorkerCount <= 0 {
		config.WorkerCount = 2
	}

	ctx, cancel := context.WithCancel(context.Background())

	logger := &AsyncAuditLogger{
		repo:        r,
		entryChan:   make(chan *models.MinIOAuditLog, config.ChannelBuffer),
		batchSize:   config.BatchSize,
		flushPeriod: config.FlushPeriod,
		workerCount: config.WorkerCount,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start worker goroutines
	for i := 0; i < config.WorkerCount; i++ {
		logger.wg.Add(1)
		go logger.worker(i)
	}

	return logger
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
	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		return fmt.Errorf("audit logger is stopped")
	}
	l.mu.Unlock()

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

	// Non-blocking enqueue with timeout
	select {
	case l.entryChan <- entry:
		return nil
	case <-time.After(100 * time.Millisecond):
		logrus.Warnf("MinIO audit log channel full, dropping entry: %s %s", action, resName)
		return fmt.Errorf("audit log channel full")
	case <-l.ctx.Done():
		return fmt.Errorf("audit logger is shutting down")
	}
}

// worker processes audit log entries from the channel.
func (l *AsyncAuditLogger) worker(workerID int) {
	defer l.wg.Done()

	batch := make([]*models.MinIOAuditLog, 0, l.batchSize)
	flushTimer := time.NewTimer(l.flushPeriod)
	defer flushTimer.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		if err := l.writeBatch(batch); err != nil {
			logrus.Errorf("MinIO Worker %d: failed to write audit log batch: %v", workerID, err)
		} else {
			logrus.Debugf("MinIO Worker %d: wrote %d audit log entries", workerID, len(batch))
		}

		batch = batch[:0]
		flushTimer.Reset(l.flushPeriod)
	}

	for {
		select {
		case entry, ok := <-l.entryChan:
			if !ok {
				flush()
				return
			}

			batch = append(batch, entry)
			if len(batch) >= l.batchSize {
				flush()
			}

		case <-flushTimer.C:
			flush()

		case <-l.ctx.Done():
			flush()
			return
		}
	}
}

// writeBatch writes a batch of MinIO audit logs to the database.
func (l *AsyncAuditLogger) writeBatch(batch []*models.MinIOAuditLog) error {
	if len(batch) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, entry := range batch {
		if err := l.repo.Create(ctx, entry); err != nil {
			logrus.Errorf("Failed to write MinIO audit log: %v", err)
		}
	}

	return nil
}

// Shutdown gracefully shuts down the async audit logger.
func (l *AsyncAuditLogger) Shutdown(timeout time.Duration) error {
	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		return nil
	}
	l.stopped = true
	l.mu.Unlock()

	logrus.Info("Shutting down MinIO async audit logger...")

	close(l.entryChan)

	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logrus.Info("MinIO async audit logger shutdown complete")
		return nil
	case <-time.After(timeout):
		l.cancel()
		logrus.Warn("MinIO async audit logger shutdown timed out")
		return fmt.Errorf("shutdown timed out after %v", timeout)
	}
}

// ChannelStatus returns the current channel usage statistics.
func (l *AsyncAuditLogger) ChannelStatus() (used, capacity int) {
	return len(l.entryChan), cap(l.entryChan)
}
