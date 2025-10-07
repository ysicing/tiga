package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// AuditLogRepository handles audit log data operations
type AuditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository creates a new AuditLogRepository
func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create creates a new audit log entry
func (r *AuditLogRepository) Create(ctx context.Context, log *models.AuditLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

// CreateBatch creates multiple audit log entries in batch
func (r *AuditLogRepository) CreateBatch(ctx context.Context, logs []*models.AuditLog) error {
	if len(logs) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).CreateInBatches(logs, 100).Error; err != nil {
		return fmt.Errorf("failed to create audit logs batch: %w", err)
	}
	return nil
}

// GetByID retrieves an audit log by ID
func (r *AuditLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.AuditLog, error) {
	var log models.AuditLog
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("id = ?", id).
		First(&log).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("audit log not found")
		}
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	return &log, nil
}

// ListAuditLogsFilter represents audit log list filters
type ListAuditLogsFilter struct {
	UserID       *uuid.UUID // Filter by user ID
	ResourceType string     // Filter by resource type
	ResourceID   *uuid.UUID // Filter by resource ID
	Action       string     // Filter by action
	Status       string     // Filter by status (success, failure)
	StartTime    time.Time  // Filter by start time
	EndTime      time.Time  // Filter by end time
	IPAddress    string     // Filter by IP address
	Search       string     // Search in description, changes
	Page         int        // Page number (1-based)
	PageSize     int        // Page size
}

// ListAuditLogs retrieves a paginated list of audit logs with filters
func (r *AuditLogRepository) ListAuditLogs(ctx context.Context, filter *ListAuditLogsFilter) ([]*models.AuditLog, int64, error) {
	query := r.db.WithContext(ctx).Preload("User")

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
		query = query.Where("description LIKE ? OR changes::text LIKE ?", search, search)
	}

	// Count total
	var total int64
	if err := query.Model(&models.AuditLog{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Fetch results
	var logs []*models.AuditLog
	if err := query.Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}

	return logs, total, nil
}

// ListByUser retrieves all audit logs for a specific user
func (r *AuditLogRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*models.AuditLog, error) {
	query := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	var logs []*models.AuditLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to list audit logs by user: %w", err)
	}

	return logs, nil
}

// ListByResource retrieves all audit logs for a specific resource
func (r *AuditLogRepository) ListByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*models.AuditLog, error) {
	var logs []*models.AuditLog
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("created_at DESC").
		Find(&logs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs by resource: %w", err)
	}

	return logs, nil
}

// ListByAction retrieves all audit logs for a specific action
func (r *AuditLogRepository) ListByAction(ctx context.Context, action string, limit int) ([]*models.AuditLog, error) {
	query := r.db.WithContext(ctx).
		Preload("User").
		Where("action = ?", action).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	var logs []*models.AuditLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to list audit logs by action: %w", err)
	}

	return logs, nil
}

// ListRecentLogs retrieves recent audit logs
func (r *AuditLogRepository) ListRecentLogs(ctx context.Context, limit int) ([]*models.AuditLog, error) {
	var logs []*models.AuditLog
	err := r.db.WithContext(ctx).
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list recent audit logs: %w", err)
	}

	return logs, nil
}

// ListFailedActions retrieves failed audit logs
func (r *AuditLogRepository) ListFailedActions(ctx context.Context, limit int) ([]*models.AuditLog, error) {
	query := r.db.WithContext(ctx).
		Preload("User").
		Where("status = ?", "failure").
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	var logs []*models.AuditLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to list failed actions: %w", err)
	}

	return logs, nil
}

// CountByStatus counts audit logs by status
func (r *AuditLogRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Where("status = ?", status).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs by status: %w", err)
	}

	return count, nil
}

// CountByAction counts audit logs by action
func (r *AuditLogRepository) CountByAction(ctx context.Context, action string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Where("action = ?", action).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs by action: %w", err)
	}

	return count, nil
}

// CountByUser counts audit logs by user
func (r *AuditLogRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs by user: %w", err)
	}

	return count, nil
}

// CountByResourceType counts audit logs by resource type
func (r *AuditLogRepository) CountByResourceType(ctx context.Context, resourceType string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Where("resource_type = ?", resourceType).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs by resource type: %w", err)
	}

	return count, nil
}

// DeleteOldLogs deletes audit logs older than a specified time
func (r *AuditLogRepository) DeleteOldLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", olderThan).
		Delete(&models.AuditLog{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// GetDistinctActions retrieves all distinct actions
func (r *AuditLogRepository) GetDistinctActions(ctx context.Context) ([]string, error) {
	var actions []string
	err := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Distinct("action").
		Pluck("action", &actions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get distinct actions: %w", err)
	}

	return actions, nil
}

// GetDistinctResourceTypes retrieves all distinct resource types
func (r *AuditLogRepository) GetDistinctResourceTypes(ctx context.Context) ([]string, error) {
	var resourceTypes []string
	err := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Distinct("resource_type").
		Pluck("resource_type", &resourceTypes).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get distinct resource types: %w", err)
	}

	return resourceTypes, nil
}

// GetActivityTimeline retrieves activity timeline aggregated by time bucket
type ActivityTimelinePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int64     `json:"count"`
}

func (r *AuditLogRepository) GetActivityTimeline(ctx context.Context, startTime, endTime time.Time, interval string) ([]*ActivityTimelinePoint, error) {
	// Use time_bucket for time-series aggregation
	query := `
		SELECT
			time_bucket($1::interval, created_at) AS timestamp,
			COUNT(*) AS count
		FROM audit_logs
		WHERE created_at >= $2 AND created_at <= $3
		GROUP BY time_bucket($1::interval, created_at)
		ORDER BY timestamp ASC
	`

	var points []*ActivityTimelinePoint
	err := r.db.WithContext(ctx).Raw(query, interval, startTime, endTime).Scan(&points).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get activity timeline: %w", err)
	}

	return points, nil
}

// AuditLogStatistics represents audit log statistics
type AuditLogStatistics struct {
	TotalLogs         int64            `json:"total_logs"`
	SuccessfulLogs    int64            `json:"successful_logs"`
	FailedLogs        int64            `json:"failed_logs"`
	ByAction          map[string]int64 `json:"by_action"`
	ByResourceType    map[string]int64 `json:"by_resource_type"`
	UniqueUsers       int64            `json:"unique_users"`
	UniqueIPAddresses int64            `json:"unique_ip_addresses"`
}

// GetStatistics retrieves audit log statistics
func (r *AuditLogRepository) GetStatistics(ctx context.Context) (*AuditLogStatistics, error) {
	stats := &AuditLogStatistics{
		ByAction:       make(map[string]int64),
		ByResourceType: make(map[string]int64),
	}

	// Total count
	if err := r.db.WithContext(ctx).Model(&models.AuditLog{}).Count(&stats.TotalLogs).Error; err != nil {
		return nil, fmt.Errorf("failed to count total audit logs: %w", err)
	}

	// Success/Failure counts
	stats.SuccessfulLogs, _ = r.CountByStatus(ctx, "success")
	stats.FailedLogs, _ = r.CountByStatus(ctx, "failure")

	// Group by action
	type ActionCount struct {
		Action string
		Count  int64
	}
	var actionCounts []ActionCount
	if err := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Select("action, COUNT(*) as count").
		Group("action").
		Scan(&actionCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to count by action: %w", err)
	}
	for _, ac := range actionCounts {
		stats.ByAction[ac.Action] = ac.Count
	}

	// Group by resource type
	type ResourceTypeCount struct {
		ResourceType string
		Count        int64
	}
	var resourceTypeCounts []ResourceTypeCount
	if err := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Select("resource_type, COUNT(*) as count").
		Group("resource_type").
		Scan(&resourceTypeCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to count by resource type: %w", err)
	}
	for _, rtc := range resourceTypeCounts {
		stats.ByResourceType[rtc.ResourceType] = rtc.Count
	}

	// Unique users
	if err := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Distinct("user_id").
		Count(&stats.UniqueUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to count unique users: %w", err)
	}

	// Unique IP addresses
	if err := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Distinct("ip_address").
		Count(&stats.UniqueIPAddresses).Error; err != nil {
		return nil, fmt.Errorf("failed to count unique IP addresses: %w", err)
	}

	return stats, nil
}

// SearchLogs performs full-text search on audit logs
func (r *AuditLogRepository) SearchLogs(ctx context.Context, query string, limit int) ([]*models.AuditLog, error) {
	search := "%" + query + "%"

	q := r.db.WithContext(ctx).
		Preload("User").
		Where("description LIKE ? OR changes::text LIKE ?", search, search).
		Order("created_at DESC")

	if limit > 0 {
		q = q.Limit(limit)
	}

	var logs []*models.AuditLog
	if err := q.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to search audit logs: %w", err)
	}

	return logs, nil
}
