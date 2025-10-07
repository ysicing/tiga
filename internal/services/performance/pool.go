package performance

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// PoolConfig holds connection pool configuration
type PoolConfig struct {
	MaxIdleConns    int           // Maximum idle connections
	MaxOpenConns    int           // Maximum open connections
	ConnMaxLifetime time.Duration // Maximum connection lifetime
	ConnMaxIdleTime time.Duration // Maximum connection idle time
}

// DefaultPoolConfig returns optimal pool configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: 1 * time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
	}
}

// PoolOptimizer optimizes connection pool settings
type PoolOptimizer struct {
	db     *sql.DB
	config *PoolConfig
	stats  *PoolStats
	mu     sync.RWMutex
}

// PoolStats tracks pool statistics
type PoolStats struct {
	OpenConnections   int           `json:"open_connections"`
	InUse             int           `json:"in_use"`
	Idle              int           `json:"idle"`
	WaitCount         int64         `json:"wait_count"`
	WaitDuration      time.Duration `json:"wait_duration"`
	MaxIdleClosed     int64         `json:"max_idle_closed"`
	MaxLifetimeClosed int64         `json:"max_lifetime_closed"`
	Timestamp         time.Time     `json:"timestamp"`
}

// NewPoolOptimizer creates a new pool optimizer
func NewPoolOptimizer(db *sql.DB, config *PoolConfig) *PoolOptimizer {
	if config == nil {
		config = DefaultPoolConfig()
	}

	optimizer := &PoolOptimizer{
		db:     db,
		config: config,
		stats:  &PoolStats{},
	}

	// Apply pool settings
	optimizer.ApplySettings()

	return optimizer
}

// ApplySettings applies pool configuration to database
func (po *PoolOptimizer) ApplySettings() {
	po.db.SetMaxIdleConns(po.config.MaxIdleConns)
	po.db.SetMaxOpenConns(po.config.MaxOpenConns)
	po.db.SetConnMaxLifetime(po.config.ConnMaxLifetime)
	po.db.SetConnMaxIdleTime(po.config.ConnMaxIdleTime)
}

// CollectStats collects current pool statistics
func (po *PoolOptimizer) CollectStats() *PoolStats {
	po.mu.Lock()
	defer po.mu.Unlock()

	stats := po.db.Stats()

	po.stats = &PoolStats{
		OpenConnections:   stats.OpenConnections,
		InUse:             stats.InUse,
		Idle:              stats.Idle,
		WaitCount:         stats.WaitCount,
		WaitDuration:      stats.WaitDuration,
		MaxIdleClosed:     stats.MaxIdleClosed,
		MaxLifetimeClosed: stats.MaxLifetimeClosed,
		Timestamp:         time.Now(),
	}

	return po.stats
}

// GetStats retrieves pool statistics
func (po *PoolOptimizer) GetStats() *PoolStats {
	po.mu.RLock()
	defer po.mu.RUnlock()

	statsCopy := *po.stats
	return &statsCopy
}

// CheckHealth checks pool health
func (po *PoolOptimizer) CheckHealth(ctx context.Context) error {
	return po.db.PingContext(ctx)
}

// AutoTune automatically adjusts pool settings based on usage
func (po *PoolOptimizer) AutoTune() {
	po.mu.Lock()
	defer po.mu.Unlock()

	stats := po.db.Stats()

	// If wait count is high, increase max open connections
	if stats.WaitCount > 100 && po.config.MaxOpenConns < 200 {
		po.config.MaxOpenConns += 10
		po.db.SetMaxOpenConns(po.config.MaxOpenConns)
	}

	// If idle connections are consistently low, reduce max idle
	if stats.Idle < po.config.MaxIdleConns/2 && po.config.MaxIdleConns > 5 {
		po.config.MaxIdleConns -= 2
		po.db.SetMaxIdleConns(po.config.MaxIdleConns)
	}

	// If idle connections are consistently high, increase max idle
	if stats.Idle >= po.config.MaxIdleConns && po.config.MaxIdleConns < 50 {
		po.config.MaxIdleConns += 2
		po.db.SetMaxIdleConns(po.config.MaxIdleConns)
	}
}

// StartAutoTune starts automatic pool tuning
func (po *PoolOptimizer) StartAutoTune(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			po.CollectStats()
			po.AutoTune()
		}
	}
}

// GetRecommendations provides pool configuration recommendations
func (po *PoolOptimizer) GetRecommendations() []string {
	po.mu.RLock()
	defer po.mu.RUnlock()

	stats := po.db.Stats()
	recommendations := make([]string, 0)

	// Check wait time
	if stats.WaitCount > 50 {
		recommendations = append(recommendations,
			fmt.Sprintf("High wait count (%d). Consider increasing MaxOpenConns from %d to %d",
				stats.WaitCount, po.config.MaxOpenConns, po.config.MaxOpenConns+20))
	}

	// Check idle connections
	utilizationRate := float64(stats.InUse) / float64(stats.OpenConnections)
	if utilizationRate < 0.3 && po.config.MaxOpenConns > 50 {
		recommendations = append(recommendations,
			fmt.Sprintf("Low utilization rate (%.2f%%). Consider reducing MaxOpenConns from %d to %d",
				utilizationRate*100, po.config.MaxOpenConns, po.config.MaxOpenConns-20))
	}

	// Check connection lifetime
	if stats.MaxLifetimeClosed > 100 {
		recommendations = append(recommendations,
			fmt.Sprintf("High lifetime closures (%d). Consider increasing ConnMaxLifetime from %s to %s",
				stats.MaxLifetimeClosed, po.config.ConnMaxLifetime, po.config.ConnMaxLifetime*2))
	}

	// Check idle timeout
	if stats.MaxIdleClosed > 100 {
		recommendations = append(recommendations,
			fmt.Sprintf("High idle closures (%d). Consider adjusting MaxIdleConns from %d or ConnMaxIdleTime from %s",
				stats.MaxIdleClosed, po.config.MaxIdleConns, po.config.ConnMaxIdleTime))
	}

	return recommendations
}

// PoolMonitorMiddleware creates middleware for pool monitoring
func PoolMonitorMiddleware(optimizer *PoolOptimizer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Collect stats before request
		beforeStats := optimizer.CollectStats()

		c.Next()

		// Collect stats after request
		afterStats := optimizer.CollectStats()

		// Add pool stats to response headers
		c.Header("X-DB-Connections-Open", fmt.Sprintf("%d", afterStats.OpenConnections))
		c.Header("X-DB-Connections-InUse", fmt.Sprintf("%d", afterStats.InUse))
		c.Header("X-DB-Connections-Idle", fmt.Sprintf("%d", afterStats.Idle))

		// Log if wait count increased
		if afterStats.WaitCount > beforeStats.WaitCount {
			c.Header("X-DB-Connection-Waited", "true")
		}
	}
}

// ConnectionGuard prevents connection exhaustion
type ConnectionGuard struct {
	maxConcurrent int
	current       int
	mu            sync.Mutex
	cond          *sync.Cond
}

// NewConnectionGuard creates a new connection guard
func NewConnectionGuard(maxConcurrent int) *ConnectionGuard {
	guard := &ConnectionGuard{
		maxConcurrent: maxConcurrent,
		current:       0,
	}
	guard.cond = sync.NewCond(&guard.mu)
	return guard
}

// Acquire acquires a connection slot
func (cg *ConnectionGuard) Acquire(ctx context.Context) error {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	for cg.current >= cg.maxConcurrent {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Wait for a slot
		cg.cond.Wait()
	}

	cg.current++
	return nil
}

// Release releases a connection slot
func (cg *ConnectionGuard) Release() {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	if cg.current > 0 {
		cg.current--
		cg.cond.Signal()
	}
}

// WithGuard executes a function with connection guard
func (cg *ConnectionGuard) WithGuard(ctx context.Context, fn func() error) error {
	if err := cg.Acquire(ctx); err != nil {
		return err
	}
	defer cg.Release()

	return fn()
}

// GetCurrent returns current connection count
func (cg *ConnectionGuard) GetCurrent() int {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	return cg.current
}

// ConnectionPoolHealth represents pool health status
type ConnectionPoolHealth struct {
	Healthy         bool       `json:"healthy"`
	Timestamp       time.Time  `json:"timestamp"`
	Issues          []string   `json:"issues,omitempty"`
	Recommendations []string   `json:"recommendations,omitempty"`
	Stats           *PoolStats `json:"stats"`
}

// CheckPoolHealth performs comprehensive pool health check
func CheckPoolHealth(optimizer *PoolOptimizer, ctx context.Context) *ConnectionPoolHealth {
	health := &ConnectionPoolHealth{
		Healthy:         true,
		Timestamp:       time.Now(),
		Issues:          make([]string, 0),
		Recommendations: make([]string, 0),
	}

	// Check connectivity
	if err := optimizer.CheckHealth(ctx); err != nil {
		health.Healthy = false
		health.Issues = append(health.Issues, fmt.Sprintf("Database connectivity failed: %v", err))
	}

	// Collect stats
	health.Stats = optimizer.CollectStats()

	// Check for issues
	if health.Stats.WaitCount > 100 {
		health.Healthy = false
		health.Issues = append(health.Issues, fmt.Sprintf("High wait count: %d", health.Stats.WaitCount))
	}

	utilizationRate := float64(health.Stats.InUse) / float64(health.Stats.OpenConnections)
	if utilizationRate > 0.9 {
		health.Healthy = false
		health.Issues = append(health.Issues, fmt.Sprintf("High utilization rate: %.2f%%", utilizationRate*100))
	}

	// Get recommendations
	health.Recommendations = optimizer.GetRecommendations()

	return health
}
