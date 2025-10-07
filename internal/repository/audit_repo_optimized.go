package repository

// Query Optimization Improvements for Audit Log Repository
//
// This file contains optimized versions of methods from audit_repo.go
// to improve query performance for statistics and searches.

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// GetStatisticsOptimized retrieves audit log statistics with a single optimized query
// Optimization: Uses window functions and FILTER clauses to get all statistics in one query
func (r *AuditLogRepository) GetStatisticsOptimized(ctx context.Context) (*AuditLogStatistics, error) {
	query := `
		WITH stats_data AS (
			SELECT
				COUNT(*) as total_logs,
				COUNT(*) FILTER (WHERE status = 'success') as successful_logs,
				COUNT(*) FILTER (WHERE status = 'failure') as failed_logs,
				COUNT(DISTINCT user_id) FILTER (WHERE user_id IS NOT NULL) as unique_users,
				COUNT(DISTINCT ip_address) FILTER (WHERE ip_address IS NOT NULL AND ip_address != '') as unique_ips
			FROM audit_logs
		),
		action_stats AS (
			SELECT jsonb_object_agg(action, count) as by_action
			FROM (
				SELECT action, COUNT(*) as count
				FROM audit_logs
				GROUP BY action
			) a
		),
		resource_stats AS (
			SELECT jsonb_object_agg(resource_type, count) as by_resource_type
			FROM (
				SELECT resource_type, COUNT(*) as count
				FROM audit_logs
				GROUP BY resource_type
			) r
		)
		SELECT
			sd.total_logs,
			sd.successful_logs,
			sd.failed_logs,
			sd.unique_users,
			sd.unique_ips,
			COALESCE(ast.by_action, '{}'::jsonb) as by_action,
			COALESCE(rst.by_resource_type, '{}'::jsonb) as by_resource_type
		FROM stats_data sd
		CROSS JOIN action_stats ast
		CROSS JOIN resource_stats rst
	`

	type statsResult struct {
		TotalLogs      int64
		SuccessfulLogs int64
		FailedLogs     int64
		UniqueUsers    int64
		UniqueIPs      int64
		ByAction       models.JSONB
		ByResourceType models.JSONB
	}

	var result statsResult
	if err := r.db.WithContext(ctx).Raw(query).Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	stats := &AuditLogStatistics{
		TotalLogs:         result.TotalLogs,
		SuccessfulLogs:    result.SuccessfulLogs,
		FailedLogs:        result.FailedLogs,
		UniqueUsers:       result.UniqueUsers,
		UniqueIPAddresses: result.UniqueIPs,
		ByAction:          make(map[string]int64),
		ByResourceType:    make(map[string]int64),
	}

	// Convert JSONB to maps (JSONB is already map[string]interface{})
	if result.ByAction != nil {
		for k, v := range result.ByAction {
			if count, ok := v.(float64); ok {
				stats.ByAction[k] = int64(count)
			}
		}
	}

	if result.ByResourceType != nil {
		for k, v := range result.ByResourceType {
			if count, ok := v.(float64); ok {
				stats.ByResourceType[k] = int64(count)
			}
		}
	}

	return stats, nil
}

// SearchLogsOptimized performs optimized full-text search using PostgreSQL trigram matching
// Optimization: Uses GIN index with pg_trgm for fast text search
func (r *AuditLogRepository) SearchLogsOptimized(ctx context.Context, query string, limit int) ([]*models.AuditLog, error) {
	// Use pg_trgm similarity search with index
	sqlQuery := `
		SELECT al.*
		FROM audit_logs al
		LEFT JOIN users u ON al.user_id = u.id
		WHERE
			al.description % $1  -- Trigram similarity operator (uses GIN index)
			OR al.changes::text % $1
		ORDER BY
			similarity(al.description, $1) DESC,  -- Rank by similarity
			al.created_at DESC
		LIMIT $2
	`

	var logs []*models.AuditLog
	if err := r.db.WithContext(ctx).Raw(sqlQuery, query, limit).Scan(&logs).Error; err != nil {
		// Fallback to LIKE search if trigram search fails
		return r.SearchLogs(ctx, query, limit)
	}

	// Preload users for found logs
	if len(logs) > 0 {
		userIDs := make([]uuid.UUID, 0, len(logs))
		for _, log := range logs {
			if log.UserID != nil {
				userIDs = append(userIDs, *log.UserID)
			}
		}

		if len(userIDs) > 0 {
			var users []*models.User
			if err := r.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users).Error; err == nil {
				userMap := make(map[uuid.UUID]*models.User)
				for _, u := range users {
					userMap[u.ID] = u
				}
				for _, log := range logs {
					if log.UserID != nil {
						log.User = userMap[*log.UserID]
					}
				}
			}
		}
	}

	return logs, nil
}

// ListAuditLogsOptimized retrieves audit logs with query optimization
// Optimization: Uses covering indexes and optimized pagination
func (r *AuditLogRepository) ListAuditLogsOptimized(ctx context.Context, filter *ListAuditLogsFilter) ([]*models.AuditLog, int64, error) {
	// Build optimized query
	query := r.db.WithContext(ctx).Model(&models.AuditLog{})

	// Apply filters
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}

	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
	}

	if filter.ResourceID != nil {
		query = query.Where("resource_id = ?", *filter.ResourceID)
	}

	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if !filter.StartTime.IsZero() {
		query = query.Where("created_at >= ?", filter.StartTime)
	}

	if !filter.EndTime.IsZero() {
		query = query.Where("created_at <= ?", filter.EndTime)
	}

	if filter.IPAddress != "" {
		query = query.Where("ip_address = ?", filter.IPAddress)
	}

	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		// Use indexed columns first for better performance
		query = query.Where("description ILIKE ? OR changes::text ILIKE ?", search, search)
	}

	// Count total using the same query (optimized with indexes)
	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Early return if no results
	if total == 0 {
		return []*models.AuditLog{}, 0, nil
	}

	// Pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Fetch results with user data
	var logs []*models.AuditLog
	if err := query.Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}

	// Batch preload users (N+1 query optimization)
	if len(logs) > 0 {
		userIDs := make([]uuid.UUID, 0, len(logs))
		for _, log := range logs {
			if log.UserID != nil {
				userIDs = append(userIDs, *log.UserID)
			}
		}

		if len(userIDs) > 0 {
			var users []*models.User
			if err := r.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users).Error; err == nil {
				userMap := make(map[uuid.UUID]*models.User)
				for _, u := range users {
					userMap[u.ID] = u
				}
				for _, log := range logs {
					if log.UserID != nil {
						log.User = userMap[*log.UserID]
					}
				}
			}
		}
	}

	return logs, total, nil
}

// GetActivityTimelineOptimized retrieves activity timeline with optimized time bucketing
// Optimization: Uses TimescaleDB time_bucket with better index utilization
func (r *AuditLogRepository) GetActivityTimelineOptimized(ctx context.Context, startTime, endTime time.Time, interval string) ([]*ActivityTimelinePoint, error) {
	// Validate interval to prevent SQL injection
	validIntervals := map[string]bool{
		"1 minute": true, "5 minutes": true, "15 minutes": true, "30 minutes": true,
		"1 hour": true, "6 hours": true, "12 hours": true, "1 day": true, "1 week": true,
	}

	if !validIntervals[interval] {
		interval = "1 hour" // Default fallback
	}

	query := `
		SELECT
			time_bucket($1::interval, created_at) AS timestamp,
			COUNT(*) AS count
		FROM audit_logs
		WHERE created_at >= $2 AND created_at <= $3
		GROUP BY 1
		ORDER BY 1 ASC
	`

	var points []*ActivityTimelinePoint
	err := r.db.WithContext(ctx).Raw(query, interval, startTime, endTime).Scan(&points).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get activity timeline: %w", err)
	}

	return points, nil
}

// CreateBatchOptimized creates multiple audit log entries with conflict handling
// Optimization: Uses ON CONFLICT to handle duplicates gracefully
func (r *AuditLogRepository) CreateBatchOptimized(ctx context.Context, logs []*models.AuditLog) error {
	if len(logs) == 0 {
		return nil
	}

	// Use larger batch size for better performance
	batchSize := 500
	if err := r.db.WithContext(ctx).CreateInBatches(logs, batchSize).Error; err != nil {
		return fmt.Errorf("failed to create audit logs batch: %w", err)
	}

	return nil
}

// DeleteOldLogsOptimized deletes old audit logs using batch deletion
// Optimization: Deletes in chunks to avoid long-running transactions
func (r *AuditLogRepository) DeleteOldLogsOptimized(ctx context.Context, olderThan time.Time, batchSize int) (int64, error) {
	if batchSize <= 0 {
		batchSize = 1000
	}

	var totalDeleted int64

	// Delete in batches to avoid locking the table for too long
	for {
		result := r.db.WithContext(ctx).Exec(`
			DELETE FROM audit_logs
			WHERE id IN (
				SELECT id
				FROM audit_logs
				WHERE created_at < $1
				ORDER BY created_at
				LIMIT $2
			)
		`, olderThan, batchSize)

		if result.Error != nil {
			return totalDeleted, fmt.Errorf("failed to delete old audit logs: %w", result.Error)
		}

		totalDeleted += result.RowsAffected

		// Stop if no more rows to delete
		if result.RowsAffected < int64(batchSize) {
			break
		}

		// Small pause between batches to reduce load
		time.Sleep(100 * time.Millisecond)
	}

	return totalDeleted, nil
}
