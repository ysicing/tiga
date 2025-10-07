package db

// Query Performance Monitoring Utilities
//
// This package provides helpers for monitoring and analyzing database query performance.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// QueryStats represents query performance statistics
type QueryStats struct {
	Query         string        `json:"query"`
	ExecutionTime time.Duration `json:"execution_time"`
	PlanningTime  time.Duration `json:"planning_time"`
	RowsReturned  int64         `json:"rows_returned"`
	BuffersHit    int64         `json:"buffers_hit"`
	BuffersRead   int64         `json:"buffers_read"`
	UsedIndexes   []string      `json:"used_indexes"`
	UsedSeqScan   bool          `json:"used_seq_scan"`
}

// ExplainResult represents the EXPLAIN ANALYZE output
type ExplainResult struct {
	Plan          interface{} `json:"Plan"`
	PlanningTime  float64     `json:"Planning Time"`
	ExecutionTime float64     `json:"Execution Time"`
	Triggers      interface{} `json:"Triggers,omitempty"`
}

// AnalyzeQuery runs EXPLAIN ANALYZE on a query and returns performance stats
func AnalyzeQuery(db *gorm.DB, query string, args ...interface{}) (*QueryStats, error) {
	explainQuery := "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) " + query

	var results []map[string]interface{}
	err := db.Raw(explainQuery, args...).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no EXPLAIN results returned")
	}

	// Parse the JSON result
	jsonData, err := json.Marshal(results[0]["QUERY PLAN"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal EXPLAIN result: %w", err)
	}

	var explainResults []ExplainResult
	if err := json.Unmarshal(jsonData, &explainResults); err != nil {
		return nil, fmt.Errorf("failed to unmarshal EXPLAIN result: %w", err)
	}

	if len(explainResults) == 0 {
		return nil, fmt.Errorf("invalid EXPLAIN result structure")
	}

	result := explainResults[0]

	stats := &QueryStats{
		Query:         query,
		ExecutionTime: time.Duration(result.ExecutionTime * float64(time.Millisecond)),
		PlanningTime:  time.Duration(result.PlanningTime * float64(time.Millisecond)),
		UsedIndexes:   extractIndexes(result.Plan),
		UsedSeqScan:   containsSeqScan(result.Plan),
	}

	// Extract additional stats from the plan
	if planMap, ok := result.Plan.(map[string]interface{}); ok {
		if rows, ok := planMap["Actual Rows"].(float64); ok {
			stats.RowsReturned = int64(rows)
		}
		if hit, ok := planMap["Shared Hit Blocks"].(float64); ok {
			stats.BuffersHit = int64(hit)
		}
		if read, ok := planMap["Shared Read Blocks"].(float64); ok {
			stats.BuffersRead = int64(read)
		}
	}

	return stats, nil
}

// extractIndexes recursively extracts index names from the query plan
func extractIndexes(plan interface{}) []string {
	indexes := []string{}

	if planMap, ok := plan.(map[string]interface{}); ok {
		if nodeType, ok := planMap["Node Type"].(string); ok {
			if nodeType == "Index Scan" || nodeType == "Index Only Scan" || nodeType == "Bitmap Index Scan" {
				if indexName, ok := planMap["Index Name"].(string); ok {
					indexes = append(indexes, indexName)
				}
			}
		}

		// Check child plans
		if plans, ok := planMap["Plans"].([]interface{}); ok {
			for _, childPlan := range plans {
				indexes = append(indexes, extractIndexes(childPlan)...)
			}
		}
	}

	return indexes
}

// containsSeqScan checks if the plan contains any sequential scans
func containsSeqScan(plan interface{}) bool {
	if planMap, ok := plan.(map[string]interface{}); ok {
		if nodeType, ok := planMap["Node Type"].(string); ok {
			if nodeType == "Seq Scan" {
				return true
			}
		}

		// Check child plans
		if plans, ok := planMap["Plans"].([]interface{}); ok {
			for _, childPlan := range plans {
				if containsSeqScan(childPlan) {
					return true
				}
			}
		}
	}

	return false
}

// LogSlowQuery logs slow queries with performance stats
type SlowQueryLogger struct {
	db        *gorm.DB
	threshold time.Duration
}

// NewSlowQueryLogger creates a new slow query logger
func NewSlowQueryLogger(db *gorm.DB, threshold time.Duration) *SlowQueryLogger {
	return &SlowQueryLogger{
		db:        db,
		threshold: threshold,
	}
}

// LogIfSlow analyzes and logs a query if it exceeds the threshold
func (l *SlowQueryLogger) LogIfSlow(query string, args ...interface{}) {
	stats, err := AnalyzeQuery(l.db, query, args...)
	if err != nil {
		logrus.Errorf("Failed to analyze query: %v", err)
		return
	}

	if stats.ExecutionTime >= l.threshold {
		logrus.Warnf("Slow query detected: %s", formatSlowQuery(stats))
	}
}

// formatSlowQuery formats a slow query for logging
func formatSlowQuery(stats *QueryStats) string {
	msg := fmt.Sprintf("\n  Query: %s\n  Execution Time: %v\n  Planning Time: %v\n  Rows: %d\n",
		stats.Query, stats.ExecutionTime, stats.PlanningTime, stats.RowsReturned)

	if len(stats.UsedIndexes) > 0 {
		msg += fmt.Sprintf("  Indexes Used: %v\n", stats.UsedIndexes)
	} else {
		msg += "  Indexes Used: NONE (WARNING!)\n"
	}

	if stats.UsedSeqScan {
		msg += "  Sequential Scan: YES (WARNING: Consider adding index)\n"
	}

	cacheHitRatio := float64(0)
	if total := stats.BuffersHit + stats.BuffersRead; total > 0 {
		cacheHitRatio = float64(stats.BuffersHit) / float64(total) * 100
	}
	msg += fmt.Sprintf("  Cache Hit Ratio: %.1f%%\n", cacheHitRatio)

	return msg
}

// IndexUsageStats represents index usage statistics
type IndexUsageStats struct {
	SchemaName  string  `json:"schema_name"`
	TableName   string  `json:"table_name"`
	IndexName   string  `json:"index_name"`
	IndexScans  int64   `json:"index_scans"`
	TuplesRead  int64   `json:"tuples_read"`
	TuplesFetch int64   `json:"tuples_fetched"`
	IndexSizeMB float64 `json:"index_size_mb"`
}

// GetIndexUsageStats retrieves index usage statistics
func GetIndexUsageStats(ctx context.Context, db *gorm.DB) ([]IndexUsageStats, error) {
	query := `
		SELECT
			schemaname as schema_name,
			tablename as table_name,
			indexname as index_name,
			idx_scan as index_scans,
			idx_tup_read as tuples_read,
			idx_tup_fetch as tuples_fetch,
			pg_size_pretty(pg_relation_size(indexrelid))::text as index_size,
			pg_relation_size(indexrelid) / (1024.0 * 1024.0) as index_size_mb
		FROM pg_stat_user_indexes
		WHERE schemaname = 'public'
		ORDER BY idx_scan DESC, index_size_mb DESC
	`

	var stats []IndexUsageStats
	if err := db.WithContext(ctx).Raw(query).Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("failed to get index usage stats: %w", err)
	}

	return stats, nil
}

// UnusedIndexes finds indexes that have never been used
func UnusedIndexes(ctx context.Context, db *gorm.DB) ([]IndexUsageStats, error) {
	query := `
		SELECT
			schemaname as schema_name,
			tablename as table_name,
			indexname as index_name,
			idx_scan as index_scans,
			pg_size_pretty(pg_relation_size(indexrelid))::text as index_size,
			pg_relation_size(indexrelid) / (1024.0 * 1024.0) as index_size_mb
		FROM pg_stat_user_indexes
		WHERE schemaname = 'public'
			AND idx_scan = 0
			AND indexrelid::regclass::text NOT LIKE '%_pkey'
		ORDER BY pg_relation_size(indexrelid) DESC
	`

	var stats []IndexUsageStats
	if err := db.WithContext(ctx).Raw(query).Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("failed to get unused indexes: %w", err)
	}

	return stats, nil
}

// TableStats represents table statistics
type TableStats struct {
	TableName       string  `json:"table_name"`
	RowCount        int64   `json:"row_count"`
	TableSizeMB     float64 `json:"table_size_mb"`
	IndexSizeMB     float64 `json:"index_size_mb"`
	TotalSizeMB     float64 `json:"total_size_mb"`
	SeqScans        int64   `json:"seq_scans"`
	SeqTupRead      int64   `json:"seq_tup_read"`
	IndexScans      int64   `json:"index_scans"`
	IndexTupFetch   int64   `json:"index_tup_fetch"`
	IndexUsageRatio float64 `json:"index_usage_ratio"`
}

// GetTableStats retrieves table statistics
func GetTableStats(ctx context.Context, db *gorm.DB) ([]TableStats, error) {
	query := `
		SELECT
			schemaname || '.' || tablename as table_name,
			n_live_tup as row_count,
			pg_total_relation_size(relid) / (1024.0 * 1024.0) as total_size_mb,
			pg_relation_size(relid) / (1024.0 * 1024.0) as table_size_mb,
			(pg_total_relation_size(relid) - pg_relation_size(relid)) / (1024.0 * 1024.0) as index_size_mb,
			seq_scan as seq_scans,
			seq_tup_read as seq_tup_read,
			idx_scan as index_scans,
			idx_tup_fetch as index_tup_fetch,
			CASE
				WHEN seq_scan + idx_scan > 0
				THEN 100.0 * idx_scan / (seq_scan + idx_scan)
				ELSE 0
			END as index_usage_ratio
		FROM pg_stat_user_tables
		WHERE schemaname = 'public'
		ORDER BY total_size_mb DESC
	`

	var stats []TableStats
	if err := db.WithContext(ctx).Raw(query).Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("failed to get table stats: %w", err)
	}

	return stats, nil
}

// CacheHitRatio calculates the database cache hit ratio
func CacheHitRatio(ctx context.Context, db *gorm.DB) (float64, error) {
	query := `
		SELECT
			CASE
				WHEN sum(heap_blks_hit) + sum(heap_blks_read) > 0
				THEN 100.0 * sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read))
				ELSE 0
			END as cache_hit_ratio
		FROM pg_statio_user_tables
	`

	var ratio float64
	if err := db.WithContext(ctx).Raw(query).Scan(&ratio).Error; err != nil {
		return 0, fmt.Errorf("failed to get cache hit ratio: %w", err)
	}

	return ratio, nil
}

// VacuumStats checks if tables need vacuuming
type VacuumStats struct {
	TableName      string     `json:"table_name"`
	LastVacuum     *time.Time `json:"last_vacuum"`
	LastAutoVacuum *time.Time `json:"last_autovacuum"`
	DeadTuples     int64      `json:"dead_tuples"`
	LiveTuples     int64      `json:"live_tuples"`
	DeadRatio      float64    `json:"dead_ratio"`
}

// GetVacuumStats retrieves vacuum statistics
func GetVacuumStats(ctx context.Context, db *gorm.DB) ([]VacuumStats, error) {
	query := `
		SELECT
			schemaname || '.' || relname as table_name,
			last_vacuum,
			last_autovacuum,
			n_dead_tup as dead_tuples,
			n_live_tup as live_tuples,
			CASE
				WHEN n_live_tup > 0
				THEN 100.0 * n_dead_tup / n_live_tup
				ELSE 0
			END as dead_ratio
		FROM pg_stat_user_tables
		WHERE schemaname = 'public'
		ORDER BY dead_ratio DESC
	`

	var stats []VacuumStats
	if err := db.WithContext(ctx).Raw(query).Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("failed to get vacuum stats: %w", err)
	}

	return stats, nil
}
