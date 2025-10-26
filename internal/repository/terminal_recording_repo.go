package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// RecordingRepositoryInterface defines the interface for terminal recording data access
type RecordingRepositoryInterface interface {
	// CRUD operations
	Create(ctx context.Context, recording *models.TerminalRecording) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.TerminalRecording, error)
	GetBySessionID(ctx context.Context, sessionID uuid.UUID) (*models.TerminalRecording, error)
	Update(ctx context.Context, recording *models.TerminalRecording) error
	Delete(ctx context.Context, id uuid.UUID) error
	BulkDelete(ctx context.Context, ids []uuid.UUID) error

	// Query operations
	List(ctx context.Context, filters RecordingFilters, page, limit int) ([]*models.TerminalRecording, int64, error)
	ListByUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]*models.TerminalRecording, int64, error)
	ListByType(ctx context.Context, recordingType string, page, limit int) ([]*models.TerminalRecording, int64, error)
	ListByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*models.TerminalRecording, error)
	Search(ctx context.Context, query string, page, limit int) ([]*models.TerminalRecording, int64, error)

	// Cleanup and maintenance operations
	FindExpired(ctx context.Context, retentionDays int, limit int) ([]*models.TerminalRecording, error)
	FindInvalid(ctx context.Context, limit int) ([]*models.TerminalRecording, error)

	// Statistics
	GetStatistics(ctx context.Context) (*RecordingStatistics, error)
}

// RecordingFilters defines query filters for listing recordings
type RecordingFilters struct {
	UserID        *uuid.UUID // Filter by user
	RecordingType *string    // Filter by recording type (docker, webssh, k8s_node, k8s_pod)
	StorageType   *string    // Filter by storage type (local, minio)
	StartTime     *time.Time // Filter by start time (after)
	EndTime       *time.Time // Filter by end time (before)
	SortBy        string     // Sort field (default: "started_at")
	SortOrder     string     // Sort order ("asc" or "desc", default: "desc")
}

// RecordingStatistics contains aggregated statistics about recordings
type RecordingStatistics struct {
	// Overall statistics
	TotalCount     int64  `json:"total_count"`
	TotalSize      int64  `json:"total_size"`       // Bytes
	TotalSizeHuman string `json:"total_size_human"` // "120.5 GB"

	// By type breakdown
	ByType map[string]*TypeStatistics `json:"by_type"`

	// Top users (Top 10)
	TopUsers []UserStatistics `json:"top_users"`

	// Time range
	OldestRecording *time.Time `json:"oldest_recording,omitempty"`
	NewestRecording *time.Time `json:"newest_recording,omitempty"`

	// Storage health
	InvalidCount int64   `json:"invalid_count"` // Zero size or zero duration
	ErrorRate    float64 `json:"error_rate"`    // Invalid / Total
}

// TypeStatistics contains statistics for a specific recording type
type TypeStatistics struct {
	RecordingType string `json:"recording_type"`
	Count         int64  `json:"count"`
	TotalSize     int64  `json:"total_size"`
	AvgDuration   int    `json:"avg_duration"` // Seconds
}

// UserStatistics contains statistics for a specific user
type UserStatistics struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Count    int64     `json:"count"`
}

// RecordingRepository implements RecordingRepositoryInterface
type RecordingRepository struct {
	db *gorm.DB
}

// NewRecordingRepository creates a new recording repository instance
func NewRecordingRepository(db *gorm.DB) RecordingRepositoryInterface {
	return &RecordingRepository{
		db: db,
	}
}

// Create creates a new terminal recording
func (r *RecordingRepository) Create(ctx context.Context, recording *models.TerminalRecording) error {
	return r.db.WithContext(ctx).Create(recording).Error
}

// GetByID retrieves a recording by its ID
func (r *RecordingRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.TerminalRecording, error) {
	var recording models.TerminalRecording
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&recording).Error
	if err != nil {
		return nil, err
	}
	return &recording, nil
}

// GetBySessionID retrieves a recording by its session ID
func (r *RecordingRepository) GetBySessionID(ctx context.Context, sessionID uuid.UUID) (*models.TerminalRecording, error) {
	var recording models.TerminalRecording
	err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&recording).Error
	if err != nil {
		return nil, err
	}
	return &recording, nil
}

// Update updates an existing recording
func (r *RecordingRepository) Update(ctx context.Context, recording *models.TerminalRecording) error {
	return r.db.WithContext(ctx).Save(recording).Error
}

// Delete deletes a recording by ID
func (r *RecordingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.TerminalRecording{}, id).Error
}

// BulkDelete deletes multiple recordings by IDs
func (r *RecordingRepository) BulkDelete(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Delete(&models.TerminalRecording{}, ids).Error
}

// List retrieves recordings with filtering, pagination, and sorting
func (r *RecordingRepository) List(ctx context.Context, filters RecordingFilters, page, limit int) ([]*models.TerminalRecording, int64, error) {
	var recordings []*models.TerminalRecording
	var total int64

	// Build query with filters
	query := r.db.WithContext(ctx).Model(&models.TerminalRecording{})

	// Apply filters
	if filters.UserID != nil {
		query = query.Where("user_id = ?", *filters.UserID)
	}
	if filters.RecordingType != nil {
		query = query.Where("recording_type = ?", *filters.RecordingType)
	}
	if filters.StorageType != nil {
		query = query.Where("storage_type = ?", *filters.StorageType)
	}
	if filters.StartTime != nil {
		query = query.Where("started_at >= ?", *filters.StartTime)
	}
	if filters.EndTime != nil {
		query = query.Where("ended_at <= ?", *filters.EndTime)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortBy := filters.SortBy
	if sortBy == "" {
		sortBy = "started_at"
	}
	sortOrder := filters.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}
	query = query.Order(sortBy + " " + sortOrder)

	// Apply pagination
	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Find(&recordings).Error; err != nil {
		return nil, 0, err
	}

	return recordings, total, nil
}

// ListByUser retrieves recordings for a specific user with pagination
func (r *RecordingRepository) ListByUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]*models.TerminalRecording, int64, error) {
	var recordings []*models.TerminalRecording
	var total int64

	query := r.db.WithContext(ctx).Where("user_id = ?", userID)

	// Count total
	if err := query.Model(&models.TerminalRecording{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (page - 1) * limit
	if err := query.Order("started_at DESC").Offset(offset).Limit(limit).Find(&recordings).Error; err != nil {
		return nil, 0, err
	}

	return recordings, total, nil
}

// ListByType retrieves recordings of a specific type with pagination
func (r *RecordingRepository) ListByType(ctx context.Context, recordingType string, page, limit int) ([]*models.TerminalRecording, int64, error) {
	var recordings []*models.TerminalRecording
	var total int64

	query := r.db.WithContext(ctx).Where("recording_type = ?", recordingType)

	// Count total
	if err := query.Model(&models.TerminalRecording{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (page - 1) * limit
	if err := query.Order("started_at DESC").Offset(offset).Limit(limit).Find(&recordings).Error; err != nil {
		return nil, 0, err
	}

	return recordings, total, nil
}

// ListByTimeRange retrieves recordings within a time range
func (r *RecordingRepository) ListByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*models.TerminalRecording, error) {
	var recordings []*models.TerminalRecording
	err := r.db.WithContext(ctx).
		Where("started_at >= ? AND started_at <= ?", startTime, endTime).
		Order("started_at DESC").
		Find(&recordings).Error
	return recordings, err
}

// Search performs full-text search on recordings (username, description, tags)
func (r *RecordingRepository) Search(ctx context.Context, query string, page, limit int) ([]*models.TerminalRecording, int64, error) {
	var recordings []*models.TerminalRecording
	var total int64

	// Build search query
	searchPattern := "%" + query + "%"
	dbQuery := r.db.WithContext(ctx).Where(
		"username LIKE ? OR description LIKE ? OR tags LIKE ?",
		searchPattern, searchPattern, searchPattern,
	)

	// Count total
	if err := dbQuery.Model(&models.TerminalRecording{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (page - 1) * limit
	if err := dbQuery.Order("started_at DESC").Offset(offset).Limit(limit).Find(&recordings).Error; err != nil {
		return nil, 0, err
	}

	return recordings, total, nil
}

// FindExpired finds recordings that have exceeded the retention period
func (r *RecordingRepository) FindExpired(ctx context.Context, retentionDays int, limit int) ([]*models.TerminalRecording, error) {
	var recordings []*models.TerminalRecording
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	err := r.db.WithContext(ctx).
		Where("ended_at IS NOT NULL AND ended_at < ?", cutoffTime).
		Limit(limit).
		Find(&recordings).Error

	return recordings, err
}

// FindInvalid finds recordings with zero file size or zero duration
func (r *RecordingRepository) FindInvalid(ctx context.Context, limit int) ([]*models.TerminalRecording, error) {
	var recordings []*models.TerminalRecording

	err := r.db.WithContext(ctx).
		Where("ended_at IS NOT NULL AND (file_size = 0 OR duration = 0)").
		Limit(limit).
		Find(&recordings).Error

	return recordings, err
}

// GetStatistics retrieves aggregated statistics about recordings
func (r *RecordingRepository) GetStatistics(ctx context.Context) (*RecordingStatistics, error) {
	stats := &RecordingStatistics{
		ByType: make(map[string]*TypeStatistics),
	}

	// Overall statistics
	var totalResult struct {
		TotalCount int64
		TotalSize  int64
	}
	if err := r.db.WithContext(ctx).
		Model(&models.TerminalRecording{}).
		Select("COUNT(*) as total_count, COALESCE(SUM(file_size), 0) as total_size").
		Scan(&totalResult).Error; err != nil {
		return nil, err
	}
	stats.TotalCount = totalResult.TotalCount
	stats.TotalSize = totalResult.TotalSize
	stats.TotalSizeHuman = formatFileSize(totalResult.TotalSize)

	// By type statistics
	var typeResults []struct {
		RecordingType string
		Count         int64
		TotalSize     int64
		AvgDuration   int
	}
	if err := r.db.WithContext(ctx).
		Model(&models.TerminalRecording{}).
		Select("recording_type, COUNT(*) as count, COALESCE(SUM(file_size), 0) as total_size, COALESCE(AVG(duration), 0) as avg_duration").
		Group("recording_type").
		Scan(&typeResults).Error; err != nil {
		return nil, err
	}
	for _, result := range typeResults {
		stats.ByType[result.RecordingType] = &TypeStatistics{
			RecordingType: result.RecordingType,
			Count:         result.Count,
			TotalSize:     result.TotalSize,
			AvgDuration:   result.AvgDuration,
		}
	}

	// Top users (Top 10)
	var userResults []struct {
		UserID   uuid.UUID
		Username string
		Count    int64
	}
	if err := r.db.WithContext(ctx).
		Model(&models.TerminalRecording{}).
		Select("user_id, username, COUNT(*) as count").
		Group("user_id, username").
		Order("count DESC").
		Limit(10).
		Scan(&userResults).Error; err != nil {
		return nil, err
	}
	stats.TopUsers = make([]UserStatistics, 0, len(userResults))
	for _, result := range userResults {
		stats.TopUsers = append(stats.TopUsers, UserStatistics{
			UserID:   result.UserID,
			Username: result.Username,
			Count:    result.Count,
		})
	}

	// Time range
	var timeResult struct {
		OldestRecording *time.Time
		NewestRecording *time.Time
	}
	if err := r.db.WithContext(ctx).
		Model(&models.TerminalRecording{}).
		Select("MIN(started_at) as oldest_recording, MAX(started_at) as newest_recording").
		Scan(&timeResult).Error; err != nil {
		return nil, err
	}
	stats.OldestRecording = timeResult.OldestRecording
	stats.NewestRecording = timeResult.NewestRecording

	// Invalid count
	var invalidCount int64
	if err := r.db.WithContext(ctx).
		Model(&models.TerminalRecording{}).
		Where("ended_at IS NOT NULL AND (file_size = 0 OR duration = 0)").
		Count(&invalidCount).Error; err != nil {
		return nil, err
	}
	stats.InvalidCount = invalidCount
	if stats.TotalCount > 0 {
		stats.ErrorRate = float64(invalidCount) / float64(stats.TotalCount)
	}

	return stats, nil
}

// formatFileSize formats bytes into human-readable string
func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	size := float64(bytes)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", size/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", size/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", size/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
