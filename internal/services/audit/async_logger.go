package audit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// AsyncLogger provides non-blocking audit logging with batching using Go generics.
// T must implement the AuditLog interface.
type AsyncLogger[T AuditLog] struct {
	repo        AuditRepository[T]
	entryChan   chan T
	batchSize   int
	flushPeriod time.Duration
	workerCount int
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	stopped     bool
	mu          sync.Mutex
	subsystem   string // For logging identification (e.g., "Database", "MinIO")
}

// NewAsyncLogger creates a generic async audit logger.
func NewAsyncLogger[T AuditLog](
	repo AuditRepository[T],
	subsystem string,
	config *Config,
) *AsyncLogger[T] {
	if config == nil {
		config = DefaultConfig()
	} else {
		config.ApplyDefaults()
	}

	ctx, cancel := context.WithCancel(context.Background())

	logger := &AsyncLogger[T]{
		repo:        repo,
		entryChan:   make(chan T, config.ChannelBuffer),
		batchSize:   config.BatchSize,
		flushPeriod: config.FlushPeriod,
		workerCount: config.WorkerCount,
		ctx:         ctx,
		cancel:      cancel,
		subsystem:   subsystem,
	}

	// Start worker goroutines
	for i := 0; i < config.WorkerCount; i++ {
		logger.wg.Add(1)
		go logger.worker(i)
	}

	return logger
}

// Enqueue adds an audit log entry to the processing queue.
// This method is non-blocking and returns immediately (with timeout).
func (l *AsyncLogger[T]) Enqueue(entry T) error {
	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		return fmt.Errorf("audit logger is stopped")
	}
	l.mu.Unlock()

	// Set creation timestamp
	entry.SetCreatedAt(time.Now().UTC())

	// Non-blocking enqueue with timeout
	select {
	case l.entryChan <- entry:
		return nil
	case <-time.After(100 * time.Millisecond):
		logrus.Warnf("%s audit log channel full, dropping entry: %s", l.subsystem, entry.GetID())
		return fmt.Errorf("audit log channel full")
	case <-l.ctx.Done():
		return fmt.Errorf("audit logger is shutting down")
	}
}

// worker processes audit log entries from the channel and writes them in batches.
func (l *AsyncLogger[T]) worker(workerID int) {
	defer l.wg.Done()

	batch := make([]T, 0, l.batchSize)
	flushTimer := time.NewTimer(l.flushPeriod)
	defer flushTimer.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		if err := l.writeBatch(batch); err != nil {
			logrus.Errorf("%s Worker %d: failed to write audit log batch: %v", l.subsystem, workerID, err)
		} else {
			logrus.Debugf("%s Worker %d: wrote %d audit log entries", l.subsystem, workerID, len(batch))
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
func (l *AsyncLogger[T]) writeBatch(batch []T) error {
	if len(batch) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try batch write first if supported
	if err := l.repo.CreateBatch(ctx, batch); err == nil {
		return nil
	}

	// Fall back to individual writes
	for _, entry := range batch {
		if err := l.repo.Create(ctx, entry); err != nil {
			logrus.Errorf("%s: failed to write audit log: %v", l.subsystem, err)
		}
	}

	return nil
}

// Shutdown gracefully shuts down the async audit logger.
func (l *AsyncLogger[T]) Shutdown(timeout time.Duration) error {
	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		return nil
	}
	l.stopped = true
	l.mu.Unlock()

	logrus.Infof("Shutting down %s async audit logger...", l.subsystem)

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
		logrus.Infof("%s async audit logger shutdown complete", l.subsystem)
		return nil
	case <-time.After(timeout):
		l.cancel() // Force cancel workers
		logrus.Warnf("%s async audit logger shutdown timed out, some logs may be lost", l.subsystem)
		return fmt.Errorf("shutdown timed out after %v", timeout)
	}
}

// ChannelStatus returns the current channel usage statistics.
func (l *AsyncLogger[T]) ChannelStatus() (used, capacity int) {
	return len(l.entryChan), cap(l.entryChan)
}
