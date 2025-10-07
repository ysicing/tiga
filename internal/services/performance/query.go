package performance

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
)

// QueryOptimizer provides query optimization utilities
type QueryOptimizer struct {
	db             *sql.DB
	slowQueryLog   []SlowQuery
	slowThreshold  time.Duration
	maxSlowQueries int
	mu             sync.RWMutex
}

// SlowQuery represents a slow query log entry
type SlowQuery struct {
	Query        string        `json:"query"`
	Duration     time.Duration `json:"duration"`
	Timestamp    time.Time     `json:"timestamp"`
	RowsAffected int64         `json:"rows_affected"`
	Error        string        `json:"error,omitempty"`
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(db *sql.DB, slowThreshold time.Duration) *QueryOptimizer {
	return &QueryOptimizer{
		db:             db,
		slowQueryLog:   make([]SlowQuery, 0),
		slowThreshold:  slowThreshold,
		maxSlowQueries: 100,
	}
}

// ExecuteWithTiming executes a query and logs if slow
func (qo *QueryOptimizer) ExecuteWithTiming(ctx context.Context, query string, args ...interface{}) (sql.Result, time.Duration, error) {
	start := time.Now()
	result, err := qo.db.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	if duration > qo.slowThreshold {
		rowsAffected, _ := result.RowsAffected()
		qo.logSlowQuery(SlowQuery{
			Query:        query,
			Duration:     duration,
			Timestamp:    start,
			RowsAffected: rowsAffected,
			Error:        fmt.Sprintf("%v", err),
		})
	}

	return result, duration, err
}

// QueryWithTiming queries and logs if slow
func (qo *QueryOptimizer) QueryWithTiming(ctx context.Context, query string, args ...interface{}) (*sql.Rows, time.Duration, error) {
	start := time.Now()
	rows, err := qo.db.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	if duration > qo.slowThreshold {
		qo.logSlowQuery(SlowQuery{
			Query:     query,
			Duration:  duration,
			Timestamp: start,
			Error:     fmt.Sprintf("%v", err),
		})
	}

	return rows, duration, err
}

// logSlowQuery logs a slow query
func (qo *QueryOptimizer) logSlowQuery(sq SlowQuery) {
	qo.mu.Lock()
	defer qo.mu.Unlock()

	qo.slowQueryLog = append(qo.slowQueryLog, sq)

	// Keep only recent slow queries
	if len(qo.slowQueryLog) > qo.maxSlowQueries {
		qo.slowQueryLog = qo.slowQueryLog[1:]
	}
}

// GetSlowQueries retrieves slow query log
func (qo *QueryOptimizer) GetSlowQueries(limit int) []SlowQuery {
	qo.mu.RLock()
	defer qo.mu.RUnlock()

	if limit <= 0 || limit > len(qo.slowQueryLog) {
		limit = len(qo.slowQueryLog)
	}

	start := len(qo.slowQueryLog) - limit
	result := make([]SlowQuery, limit)
	copy(result, qo.slowQueryLog[start:])

	return result
}

// ClearSlowQueries clears the slow query log
func (qo *QueryOptimizer) ClearSlowQueries() {
	qo.mu.Lock()
	defer qo.mu.Unlock()

	qo.slowQueryLog = make([]SlowQuery, 0)
}

// ExplainQuery runs EXPLAIN on a query
func (qo *QueryOptimizer) ExplainQuery(ctx context.Context, query string, args ...interface{}) (string, error) {
	explainQuery := "EXPLAIN ANALYZE " + query

	rows, err := qo.db.QueryContext(ctx, explainQuery, args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var explanation strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return "", err
		}
		explanation.WriteString(line + "\n")
	}

	return explanation.String(), nil
}

// QueryStatistics holds query statistics
type QueryStatistics struct {
	TotalQueries    int64         `json:"total_queries"`
	SlowQueries     int64         `json:"slow_queries"`
	AverageDuration time.Duration `json:"average_duration"`
	MaxDuration     time.Duration `json:"max_duration"`
	MinDuration     time.Duration `json:"min_duration"`
	ErrorCount      int64         `json:"error_count"`
}

// BatchExecutor executes queries in batches for better performance
type BatchExecutor struct {
	db        *sql.DB
	batchSize int
}

// NewBatchExecutor creates a new batch executor
func NewBatchExecutor(db *sql.DB, batchSize int) *BatchExecutor {
	if batchSize <= 0 {
		batchSize = 100
	}

	return &BatchExecutor{
		db:        db,
		batchSize: batchSize,
	}
}

// ExecuteBatch executes multiple statements in batches
func (be *BatchExecutor) ExecuteBatch(ctx context.Context, query string, params [][]interface{}) error {
	if len(params) == 0 {
		return nil
	}

	tx, err := be.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, param := range params {
		if _, err := stmt.ExecContext(ctx, param...); err != nil {
			return fmt.Errorf("failed to execute batch item %d: %w", i, err)
		}

		// Commit in batches
		if (i+1)%be.batchSize == 0 {
			if err := tx.Commit(); err != nil {
				return err
			}

			// Start new transaction
			tx, err = be.db.BeginTx(ctx, nil)
			if err != nil {
				return err
			}
			defer tx.Rollback()

			stmt, err = tx.PrepareContext(ctx, query)
			if err != nil {
				return err
			}
			defer stmt.Close()
		}
	}

	// Commit remaining items
	return tx.Commit()
}

// PreparedStatementCache caches prepared statements
type PreparedStatementCache struct {
	db         *sql.DB
	statements map[string]*sql.Stmt
	mu         sync.RWMutex
}

// NewPreparedStatementCache creates a new prepared statement cache
func NewPreparedStatementCache(db *sql.DB) *PreparedStatementCache {
	return &PreparedStatementCache{
		db:         db,
		statements: make(map[string]*sql.Stmt),
	}
}

// Get retrieves or creates a prepared statement
func (psc *PreparedStatementCache) Get(ctx context.Context, query string) (*sql.Stmt, error) {
	psc.mu.RLock()
	stmt, exists := psc.statements[query]
	psc.mu.RUnlock()

	if exists {
		return stmt, nil
	}

	// Prepare statement
	psc.mu.Lock()
	defer psc.mu.Unlock()

	// Double-check after acquiring write lock
	if stmt, exists := psc.statements[query]; exists {
		return stmt, nil
	}

	stmt, err := psc.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}

	psc.statements[query] = stmt
	return stmt, nil
}

// Close closes all prepared statements
func (psc *PreparedStatementCache) Close() error {
	psc.mu.Lock()
	defer psc.mu.Unlock()

	var lastErr error
	for _, stmt := range psc.statements {
		if err := stmt.Close(); err != nil {
			lastErr = err
		}
	}

	psc.statements = make(map[string]*sql.Stmt)
	return lastErr
}

// IndexRecommendation represents an index suggestion
type IndexRecommendation struct {
	Table           string   `json:"table"`
	Columns         []string `json:"columns"`
	Reason          string   `json:"reason"`
	EstimatedImpact string   `json:"estimated_impact"`
	CreateSQL       string   `json:"create_sql"`
}

// AnalyzeQueryPlan analyzes a query plan and suggests optimizations
func AnalyzeQueryPlan(ctx context.Context, db *sql.DB, query string, args ...interface{}) ([]string, error) {
	suggestions := make([]string, 0)

	// Run EXPLAIN ANALYZE
	explainQuery := "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) " + query
	rows, err := db.QueryContext(ctx, explainQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var planLines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return nil, err
		}
		planLines = append(planLines, line)
	}

	plan := strings.Join(planLines, "\n")

	// Analyze plan for common issues
	if strings.Contains(plan, "Seq Scan") {
		suggestions = append(suggestions, "Sequential scan detected - consider adding index")
	}

	if strings.Contains(plan, "rows=") {
		suggestions = append(suggestions, "Check if row estimates are accurate - update statistics if needed")
	}

	if strings.Contains(plan, "Buffers:") && strings.Contains(plan, "read=") {
		suggestions = append(suggestions, "High disk I/O detected - consider increasing shared_buffers or adding indexes")
	}

	if strings.Contains(plan, "Sort") && strings.Contains(plan, "external") {
		suggestions = append(suggestions, "External sort detected - consider increasing work_mem")
	}

	if strings.Contains(plan, "Hash Join") {
		suggestions = append(suggestions, "Hash join detected - ensure join columns are indexed")
	}

	return suggestions, nil
}

// QueryCache provides simple in-memory query result caching
type QueryCache struct {
	cache map[string]queryCacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

type queryCacheEntry struct {
	result    interface{}
	expiresAt time.Time
}

// NewQueryCache creates a new query cache
func NewQueryCache(ttl time.Duration) *QueryCache {
	return &QueryCache{
		cache: make(map[string]queryCacheEntry),
		ttl:   ttl,
	}
}

// Get retrieves from cache
func (qc *QueryCache) Get(key string) (interface{}, bool) {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	entry, exists := qc.cache[key]
	if !exists || time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.result, true
}

// Set stores in cache
func (qc *QueryCache) Set(key string, value interface{}) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	qc.cache[key] = queryCacheEntry{
		result:    value,
		expiresAt: time.Now().Add(qc.ttl),
	}
}

// Clear removes expired entries
func (qc *QueryCache) Clear() {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	now := time.Now()
	for key, entry := range qc.cache {
		if now.After(entry.expiresAt) {
			delete(qc.cache, key)
		}
	}
}

// Size returns cache size
func (qc *QueryCache) Size() int {
	qc.mu.RLock()
	defer qc.mu.RUnlock()
	return len(qc.cache)
}
