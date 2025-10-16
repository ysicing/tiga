package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"

	dbrepo "github.com/ysicing/tiga/internal/repository/database"
)

// AsyncAuditLogger provides non-blocking audit logging with batching.
type AsyncAuditLogger struct {
	repo        *dbrepo.AuditLogRepository
	entryChan   chan *models.DatabaseAuditLog
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
	// ChannelBuffer is the size of the channel buffer (default: 1000)
	ChannelBuffer int
	// BatchSize is the maximum number of logs to write in one transaction (default: 50)
	BatchSize int
	// FlushPeriod is how often to flush pending logs even if batch is not full (default: 5s)
	FlushPeriod time.Duration
	// WorkerCount is the number of worker goroutines (default: 2)
	WorkerCount int
}

// NewAsyncAuditLogger creates an async audit logger with the given configuration.
func NewAsyncAuditLogger(repo *dbrepo.AuditLogRepository, config *AsyncAuditLoggerConfig) *AsyncAuditLogger {
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
		repo:        repo,
		entryChan:   make(chan *models.DatabaseAuditLog, config.ChannelBuffer),
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

// LogAction enqueues an audit entry for async processing.
// This method is non-blocking and returns immediately.
func (l *AsyncAuditLogger) LogAction(ctx context.Context, entry AuditEntry) error {
	if entry.Operator == "" {
		return fmt.Errorf("operator is required")
	}
	if entry.Action == "" {
		return fmt.Errorf("action is required")
	}

	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		return fmt.Errorf("audit logger is stopped")
	}
	l.mu.Unlock()

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

	// Non-blocking enqueue with timeout
	select {
	case l.entryChan <- log:
		return nil
	case <-time.After(100 * time.Millisecond):
		// Channel is full, log warning but don't block caller
		logrus.Warnf("Audit log channel full, dropping log entry: %s %s", entry.Action, entry.TargetName)
		return fmt.Errorf("audit log channel full")
	case <-l.ctx.Done():
		return fmt.Errorf("audit logger is shutting down")
	}
}

// worker processes audit log entries from the channel and writes them in batches.
func (l *AsyncAuditLogger) worker(workerID int) {
	defer l.wg.Done()

	batch := make([]*models.DatabaseAuditLog, 0, l.batchSize)
	flushTimer := time.NewTimer(l.flushPeriod)
	defer flushTimer.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		if err := l.writeBatch(batch); err != nil {
			logrus.Errorf("Worker %d: failed to write audit log batch: %v", workerID, err)
		} else {
			logrus.Debugf("Worker %d: wrote %d audit log entries", workerID, len(batch))
		}

		batch = batch[:0] // Reset batch
		flushTimer.Reset(l.flushPeriod)
	}

	for {
		select {
		case entry, ok := <-l.entryChan:
			if !ok {
				// Channel closed, flush remaining entries and exit
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
			// Shutdown signal, flush remaining entries and exit
			flush()
			return
		}
	}
}

// writeBatch writes a batch of audit logs to the database.
func (l *AsyncAuditLogger) writeBatch(batch []*models.DatabaseAuditLog) error {
	if len(batch) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Write all entries in the batch
	for _, entry := range batch {
		if err := l.repo.Create(ctx, entry); err != nil {
			// Log individual failures but continue processing
			logrus.Errorf("Failed to write audit log: %v", err)
		}
	}

	return nil
}

// Shutdown gracefully shuts down the async audit logger.
// It closes the entry channel and waits for all workers to finish processing.
func (l *AsyncAuditLogger) Shutdown(timeout time.Duration) error {
	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		return nil
	}
	l.stopped = true
	l.mu.Unlock()

	logrus.Info("Shutting down async audit logger...")

	// Close the channel to signal workers to stop after processing remaining entries
	close(l.entryChan)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logrus.Info("Async audit logger shutdown complete")
		return nil
	case <-time.After(timeout):
		l.cancel() // Force cancel workers
		logrus.Warn("Async audit logger shutdown timed out, some logs may be lost")
		return fmt.Errorf("shutdown timed out after %v", timeout)
	}
}

// ChannelStatus returns the current channel usage statistics.
func (l *AsyncAuditLogger) ChannelStatus() (used, capacity int) {
	return len(l.entryChan), cap(l.entryChan)
}
