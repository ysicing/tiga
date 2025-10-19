package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// Service handles audit log business logic
type Service struct {
	repo *repository.AuditLogRepository
}

// NewService creates a new audit log service
func NewService(repo *repository.AuditLogRepository) *Service {
	return &Service{
		repo: repo,
	}
}

// LogEntry represents an audit log entry for creation
type LogEntry struct {
	UserID       *uuid.UUID
	Username     string
	ClusterID    *uint  // Phase 4: Cluster context
	ClusterName  string // Phase 4: Cluster snapshot
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	ResourceName string
	Description  string
	Changes      map[string]interface{}
	IPAddress    string
	UserAgent    string
	RequestID    string
	Status       string
	ErrorMessage string
}

// Log creates a new audit log entry
func (s *Service) Log(ctx context.Context, entry *LogEntry) error {
	if entry.Status == "" {
		entry.Status = "success"
	}

	auditLog := &models.AuditLog{
		UserID:       entry.UserID,
		Username:     entry.Username,
		ClusterID:    entry.ClusterID,   // Phase 4
		ClusterName:  entry.ClusterName, // Phase 4
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceID:   entry.ResourceID,
		ResourceName: entry.ResourceName,
		Description:  entry.Description,
		Changes:      entry.Changes,
		IPAddress:    entry.IPAddress,
		UserAgent:    entry.UserAgent,
		RequestID:    entry.RequestID,
		Status:       entry.Status,
		ErrorMessage: entry.ErrorMessage,
		CreatedAt:    time.Now(),
	}

	return s.repo.Create(ctx, auditLog)
}

// LogSuccess creates a successful audit log entry
func (s *Service) LogSuccess(ctx context.Context, userID *uuid.UUID, username, action, resourceType string, resourceID *uuid.UUID, resourceName, description string, changes map[string]interface{}, ipAddress, userAgent, requestID string) error {
	entry := &LogEntry{
		UserID:       userID,
		Username:     username,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Description:  description,
		Changes:      changes,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		RequestID:    requestID,
		Status:       "success",
	}

	return s.Log(ctx, entry)
}

// LogFailure creates a failed audit log entry
func (s *Service) LogFailure(ctx context.Context, userID *uuid.UUID, username, action, resourceType string, resourceID *uuid.UUID, resourceName, description, errorMessage string, ipAddress, userAgent, requestID string) error {
	entry := &LogEntry{
		UserID:       userID,
		Username:     username,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Description:  description,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		RequestID:    requestID,
		Status:       "failure",
		ErrorMessage: errorMessage,
	}

	return s.Log(ctx, entry)
}

// LogBatch creates multiple audit log entries in batch
func (s *Service) LogBatch(ctx context.Context, entries []*LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	logs := make([]*models.AuditLog, 0, len(entries))
	for _, entry := range entries {
		if entry.Status == "" {
			entry.Status = "success"
		}

		logs = append(logs, &models.AuditLog{
			UserID:       entry.UserID,
			Username:     entry.Username,
			Action:       entry.Action,
			ResourceType: entry.ResourceType,
			ResourceID:   entry.ResourceID,
			ResourceName: entry.ResourceName,
			Description:  entry.Description,
			Changes:      entry.Changes,
			IPAddress:    entry.IPAddress,
			UserAgent:    entry.UserAgent,
			RequestID:    entry.RequestID,
			Status:       entry.Status,
			ErrorMessage: entry.ErrorMessage,
			CreatedAt:    time.Now(),
		})
	}

	return s.repo.CreateBatch(ctx, logs)
}

// GetByID retrieves an audit log by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.AuditLog, error) {
	return s.repo.GetByID(ctx, id)
}

// QueryFilter represents query parameters for audit logs
type QueryFilter struct {
	UserID       *uuid.UUID
	ResourceType string
	ResourceID   *uuid.UUID
	Action       string
	Status       string
	StartTime    *time.Time
	EndTime      *time.Time
	IPAddress    string
	Search       string
	Page         int
	PageSize     int
}

// Query retrieves audit logs with filters
func (s *Service) Query(ctx context.Context, filter *QueryFilter) ([]*models.AuditLog, int64, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	// Build repository filter
	repoFilter := &repository.ListAuditLogsFilter{
		UserID:       filter.UserID,
		ResourceType: filter.ResourceType,
		ResourceID:   filter.ResourceID,
		Action:       filter.Action,
		Status:       filter.Status,
		IPAddress:    filter.IPAddress,
		Search:       filter.Search,
		Page:         filter.Page,
		PageSize:     filter.PageSize,
	}

	if filter.StartTime != nil {
		repoFilter.StartTime = *filter.StartTime
	}
	if filter.EndTime != nil {
		repoFilter.EndTime = *filter.EndTime
	}

	return s.repo.ListAuditLogs(ctx, repoFilter)
}

// QueryByUser retrieves audit logs for a specific user
func (s *Service) QueryByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*models.AuditLog, int64, error) {
	return s.Query(ctx, &QueryFilter{
		UserID:   &userID,
		Page:     page,
		PageSize: pageSize,
	})
}

// QueryByResource retrieves audit logs for a specific resource
func (s *Service) QueryByResource(ctx context.Context, resourceType string, resourceID uuid.UUID, page, pageSize int) ([]*models.AuditLog, int64, error) {
	return s.Query(ctx, &QueryFilter{
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		Page:         page,
		PageSize:     pageSize,
	})
}

// QueryByAction retrieves audit logs for a specific action
func (s *Service) QueryByAction(ctx context.Context, action string, page, pageSize int) ([]*models.AuditLog, int64, error) {
	return s.Query(ctx, &QueryFilter{
		Action:   action,
		Page:     page,
		PageSize: pageSize,
	})
}

// QueryByTimeRange retrieves audit logs within a time range
func (s *Service) QueryByTimeRange(ctx context.Context, startTime, endTime time.Time, page, pageSize int) ([]*models.AuditLog, int64, error) {
	return s.Query(ctx, &QueryFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
		Page:      page,
		PageSize:  pageSize,
	})
}

// GetUserActivity retrieves recent activity for a user
func (s *Service) GetUserActivity(ctx context.Context, userID uuid.UUID, limit int) ([]*models.AuditLog, error) {
	logs, _, err := s.Query(ctx, &QueryFilter{
		UserID:   &userID,
		Page:     1,
		PageSize: limit,
	})
	return logs, err
}

// GetResourceHistory retrieves change history for a resource
func (s *Service) GetResourceHistory(ctx context.Context, resourceType string, resourceID uuid.UUID, limit int) ([]*models.AuditLog, error) {
	logs, _, err := s.Query(ctx, &QueryFilter{
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		Page:         1,
		PageSize:     limit,
	})
	return logs, err
}

// GetActions retrieves distinct actions from audit logs
func (s *Service) GetActions(ctx context.Context) ([]string, error) {
	return s.repo.GetDistinctActions(ctx)
}

// GetResourceTypes retrieves distinct resource types from audit logs
func (s *Service) GetResourceTypes(ctx context.Context) ([]string, error) {
	return s.repo.GetDistinctResourceTypes(ctx)
}

// GetStatistics retrieves audit log statistics
func (s *Service) GetStatistics(ctx context.Context) (*repository.AuditLogStatistics, error) {
	stats, err := s.repo.GetStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	return stats, nil
}

// GetActivityTimeline retrieves activity timeline (grouped by time)
func (s *Service) GetActivityTimeline(ctx context.Context, startTime, endTime time.Time, interval string) ([]*repository.ActivityTimelinePoint, error) {
	timeline, err := s.repo.GetActivityTimeline(ctx, startTime, endTime, interval)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity timeline: %w", err)
	}

	return timeline, nil
}

// CleanupOldLogs deletes audit logs older than the specified retention period
func (s *Service) CleanupOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, fmt.Errorf("retention days must be positive")
	}

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	return s.repo.DeleteOldLogs(ctx, cutoffDate)
}

// ExportLogs exports audit logs to a specific format (for compliance)
func (s *Service) ExportLogs(ctx context.Context, filter *QueryFilter, format string) ([]byte, error) {
	logs, _, err := s.Query(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}

	// TODO: Implement different export formats (CSV, JSON, etc.)
	// For now, just return JSON
	return s.exportToJSON(logs)
}

// exportToJSON exports logs to JSON format
func (s *Service) exportToJSON(logs []*models.AuditLog) ([]byte, error) {
	// This is a placeholder - real implementation would use encoding/json
	return []byte("{}"), nil
}
