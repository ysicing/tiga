package unit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
	dbservice "github.com/ysicing/tiga/internal/services/database"
	dbrepo "github.com/ysicing/tiga/internal/repository/database"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.DatabaseAuditLog{})
	require.NoError(t, err)

	return db
}

func TestAsyncAuditLogger_BasicLogging(t *testing.T) {
	db := setupTestDB(t)
	repo := dbrepo.NewAuditLogRepository(db)

	config := &dbservice.AsyncAuditLoggerConfig{
		ChannelBuffer: 100,
		BatchSize:     10,
		FlushPeriod:   100 * time.Millisecond,
		WorkerCount:   1,
	}

	logger := dbservice.NewAsyncAuditLogger(repo, config)
	defer logger.Shutdown(5 * time.Second)

	instanceID := uuid.New()
	entry := dbservice.AuditEntry{
		InstanceID: &instanceID,
		Operator:   "test-user",
		Action:     "create_database",
		TargetType: "database",
		TargetName: "testdb",
		Details:    map[string]interface{}{"size": 100},
		Success:    true,
		ClientIP:   "127.0.0.1",
	}

	err := logger.LogAction(context.Background(), entry)
	require.NoError(t, err)

	// Wait for async processing
	time.Sleep(200 * time.Millisecond)

	// Verify log was written
	logs, total, err := repo.List(context.Background(), 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, logs, 1)
	assert.Equal(t, "test-user", logs[0].Operator)
	assert.Equal(t, "create_database", logs[0].Action)
}

func TestAsyncAuditLogger_BatchProcessing(t *testing.T) {
	db := setupTestDB(t)
	repo := dbrepo.NewAuditLogRepository(db)

	config := &dbservice.AsyncAuditLoggerConfig{
		ChannelBuffer: 100,
		BatchSize:     5, // Small batch size for testing
		FlushPeriod:   1 * time.Second,
		WorkerCount:   1, // Use 1 worker for SQLite in-memory compatibility
	}

	logger := dbservice.NewAsyncAuditLogger(repo, config)
	defer logger.Shutdown(5 * time.Second)

	// Log 12 entries (should trigger 2 full batches + 2 remaining)
	for i := 0; i < 12; i++ {
		instanceID := uuid.New()
		entry := dbservice.AuditEntry{
			InstanceID: &instanceID,
			Operator:   "test-user",
			Action:     "test_action",
			TargetType: "database",
			TargetName: "testdb",
			Success:    true,
		}
		err := logger.LogAction(context.Background(), entry)
		require.NoError(t, err)
	}

	// Shutdown to flush all remaining logs
	err := logger.Shutdown(5 * time.Second)
	require.NoError(t, err)

	logs, total, err := repo.List(context.Background(), 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(12), total)
	assert.Len(t, logs, 12)
}

func TestAsyncAuditLogger_ConcurrentLogging(t *testing.T) {
	db := setupTestDB(t)
	repo := dbrepo.NewAuditLogRepository(db)

	config := &dbservice.AsyncAuditLoggerConfig{
		ChannelBuffer: 1000,
		BatchSize:     50,
		FlushPeriod:   100 * time.Millisecond,
		WorkerCount:   1, // Use 1 worker to avoid SQLite in-memory concurrency issues
	}

	logger := dbservice.NewAsyncAuditLogger(repo, config)
	defer logger.Shutdown(5 * time.Second)

	// Concurrently log from multiple goroutines
	const goroutines = 10
	const logsPerGoroutine = 20
	var wg sync.WaitGroup

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < logsPerGoroutine; i++ {
				instanceID := uuid.New()
				entry := dbservice.AuditEntry{
					InstanceID: &instanceID,
					Operator:   "test-user",
					Action:     "concurrent_action",
					TargetType: "database",
					TargetName: "testdb",
					Success:    true,
				}
				err := logger.LogAction(context.Background(), entry)
				assert.NoError(t, err)
			}
		}(g)
	}

	wg.Wait()

	// Wait for all async processing to complete
	time.Sleep(500 * time.Millisecond)

	logs, total, err := repo.List(context.Background(), 1, 300)
	require.NoError(t, err)
	assert.Equal(t, int64(goroutines*logsPerGoroutine), total)
	assert.Len(t, logs, goroutines*logsPerGoroutine)
}

func TestAsyncAuditLogger_GracefulShutdown(t *testing.T) {
	db := setupTestDB(t)
	repo := dbrepo.NewAuditLogRepository(db)

	config := &dbservice.AsyncAuditLoggerConfig{
		ChannelBuffer: 100,
		BatchSize:     100, // Large batch to ensure logs stay in buffer
		FlushPeriod:   10 * time.Second,
		WorkerCount:   1,
	}

	logger := dbservice.NewAsyncAuditLogger(repo, config)

	// Log 10 entries
	for i := 0; i < 10; i++ {
		instanceID := uuid.New()
		entry := dbservice.AuditEntry{
			InstanceID: &instanceID,
			Operator:   "test-user",
			Action:     "shutdown_test",
			TargetType: "database",
			TargetName: "testdb",
			Success:    true,
		}
		err := logger.LogAction(context.Background(), entry)
		require.NoError(t, err)
	}

	// Shutdown immediately (before flush period)
	err := logger.Shutdown(5 * time.Second)
	require.NoError(t, err)

	// Verify all logs were flushed during shutdown
	logs, total, err := repo.List(context.Background(), 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(10), total)
	assert.Len(t, logs, 10)
}

func TestAsyncAuditLogger_Validation(t *testing.T) {
	db := setupTestDB(t)
	repo := dbrepo.NewAuditLogRepository(db)

	logger := dbservice.NewAsyncAuditLogger(repo, nil)
	defer logger.Shutdown(5 * time.Second)

	t.Run("MissingOperator", func(t *testing.T) {
		entry := dbservice.AuditEntry{
			Action:  "test_action",
			Success: true,
		}
		err := logger.LogAction(context.Background(), entry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "operator is required")
	})

	t.Run("MissingAction", func(t *testing.T) {
		entry := dbservice.AuditEntry{
			Operator: "test-user",
			Success:  true,
		}
		err := logger.LogAction(context.Background(), entry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "action is required")
	})
}

func TestAsyncAuditLogger_ChannelStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := dbrepo.NewAuditLogRepository(db)

	config := &dbservice.AsyncAuditLoggerConfig{
		ChannelBuffer: 100,
		BatchSize:     1000, // Large batch to keep entries in channel
		FlushPeriod:   10 * time.Second,
		WorkerCount:   1,
	}

	logger := dbservice.NewAsyncAuditLogger(repo, config)
	defer logger.Shutdown(5 * time.Second)

	used, capacity := logger.ChannelStatus()
	assert.Equal(t, 0, used)
	assert.Equal(t, 100, capacity)

	// Add some entries
	for i := 0; i < 5; i++ {
		instanceID := uuid.New()
		entry := dbservice.AuditEntry{
			InstanceID: &instanceID,
			Operator:   "test-user",
			Action:     "test_action",
			Success:    true,
		}
		logger.LogAction(context.Background(), entry)
	}

	// Check status (should have some entries, but exact number depends on timing)
	used, capacity = logger.ChannelStatus()
	assert.Equal(t, 100, capacity)
	// used may be 0-5 depending on worker processing speed
	assert.GreaterOrEqual(t, used, 0)
	assert.LessOrEqual(t, used, 5)
}

func TestAsyncAuditLogger_DefaultConfig(t *testing.T) {
	db := setupTestDB(t)
	repo := dbrepo.NewAuditLogRepository(db)

	// Test with nil config (should use defaults)
	logger := dbservice.NewAsyncAuditLogger(repo, nil)
	defer logger.Shutdown(5 * time.Second)

	assert.NotNil(t, logger)

	// Verify channel capacity using ChannelStatus
	used, capacity := logger.ChannelStatus()
	assert.Equal(t, 1000, capacity) // Default channel buffer
	assert.Equal(t, 0, used)
}

func TestAsyncAuditLogger_FlushPeriodTrigger(t *testing.T) {
	db := setupTestDB(t)
	repo := dbrepo.NewAuditLogRepository(db)

	config := &dbservice.AsyncAuditLoggerConfig{
		ChannelBuffer: 100,
		BatchSize:     100, // Large batch so flush is triggered by period, not size
		FlushPeriod:   200 * time.Millisecond,
		WorkerCount:   1,
	}

	logger := dbservice.NewAsyncAuditLogger(repo, config)
	defer logger.Shutdown(5 * time.Second)

	// Log 5 entries (less than batch size)
	for i := 0; i < 5; i++ {
		instanceID := uuid.New()
		entry := dbservice.AuditEntry{
			InstanceID: &instanceID,
			Operator:   "test-user",
			Action:     "periodic_flush",
			Success:    true,
		}
		err := logger.LogAction(context.Background(), entry)
		require.NoError(t, err)
	}

	// Wait for flush period to trigger
	time.Sleep(400 * time.Millisecond)

	logs, total, err := repo.List(context.Background(), 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, logs, 5)
}

func TestAsyncAuditLogger_StoppedLogger(t *testing.T) {
	db := setupTestDB(t)
	repo := dbrepo.NewAuditLogRepository(db)

	logger := dbservice.NewAsyncAuditLogger(repo, nil)

	// Shutdown logger
	err := logger.Shutdown(5 * time.Second)
	require.NoError(t, err)

	// Try to log after shutdown
	instanceID := uuid.New()
	entry := dbservice.AuditEntry{
		InstanceID: &instanceID,
		Operator:   "test-user",
		Action:     "after_shutdown",
		Success:    true,
	}
	err = logger.LogAction(context.Background(), entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stopped")
}

func BenchmarkAsyncAuditLogger_LogAction(b *testing.B) {
	db := setupTestDB(&testing.T{})
	repo := dbrepo.NewAuditLogRepository(db)

	config := &dbservice.AsyncAuditLoggerConfig{
		ChannelBuffer: 10000,
		BatchSize:     100,
		FlushPeriod:   1 * time.Second,
		WorkerCount:   4,
	}

	logger := dbservice.NewAsyncAuditLogger(repo, config)
	defer logger.Shutdown(10 * time.Second)

	instanceID := uuid.New()
	entry := dbservice.AuditEntry{
		InstanceID: &instanceID,
		Operator:   "benchmark-user",
		Action:     "benchmark_action",
		TargetType: "database",
		TargetName: "testdb",
		Success:    true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogAction(context.Background(), entry)
	}
}
