package repository

import (
	"context"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// TerminalRecordingRepository implements TerminalRecordingRepositoryInterface
type TerminalRecordingRepository struct {
	db *gorm.DB
}

// NewTerminalRecordingRepository creates a new terminal recording repository
func NewTerminalRecordingRepository(db *gorm.DB) *TerminalRecordingRepository {
	return &TerminalRecordingRepository{db: db}
}

// Create creates a new terminal recording
func (r *TerminalRecordingRepository) Create(ctx context.Context, recording *models.TerminalRecording) error {
	return r.db.WithContext(ctx).Create(recording).Error
}

// GetByID retrieves a terminal recording by ID
func (r *TerminalRecordingRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.TerminalRecording, error) {
	var recording models.TerminalRecording
	err := r.db.WithContext(ctx).
		Preload("Instance").
		Where("id = ?", id).
		First(&recording).Error
	if err != nil {
		return nil, err
	}
	return &recording, nil
}

// GetBySessionID retrieves a terminal recording by session ID
func (r *TerminalRecordingRepository) GetBySessionID(ctx context.Context, sessionID uuid.UUID) (*models.TerminalRecording, error) {
	var recording models.TerminalRecording
	err := r.db.WithContext(ctx).
		Preload("Instance").
		Where("session_id = ?", sessionID).
		First(&recording).Error
	if err != nil {
		return nil, err
	}
	return &recording, nil
}

// Update updates a terminal recording
func (r *TerminalRecordingRepository) Update(ctx context.Context, recording *models.TerminalRecording) error {
	return r.db.WithContext(ctx).Save(recording).Error
}

// Delete deletes a terminal recording by ID
func (r *TerminalRecordingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.TerminalRecording{}, "id = ?", id).Error
}

// List retrieves a list of terminal recordings with filters
func (r *TerminalRecordingRepository) List(ctx context.Context, filter *TerminalRecordingFilter) ([]*models.TerminalRecording, int64, error) {
	var recordings []*models.TerminalRecording
	var total int64

	query := r.db.WithContext(ctx).Model(&models.TerminalRecording{})

	// Apply filters
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.InstanceID != nil {
		query = query.Where("instance_id = ?", *filter.InstanceID)
	}
	if filter.ContainerID != nil {
		query = query.Where("container_id = ?", *filter.ContainerID)
	}
	if filter.StartDate != nil {
		query = query.Where("started_at >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("started_at <= ?", *filter.EndDate)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortBy := "started_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	sortOrder := "DESC"
	if filter.SortOrder != "" {
		sortOrder = filter.SortOrder
	}
	query = query.Order(sortBy + " " + sortOrder)

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	// Fetch recordings with preloaded relations
	err := query.Preload("Instance").Find(&recordings).Error
	if err != nil {
		return nil, 0, err
	}

	return recordings, total, nil
}

// ListByUser retrieves terminal recordings for a specific user
func (r *TerminalRecordingRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.TerminalRecording, int64, error) {
	var recordings []*models.TerminalRecording
	var total int64

	query := r.db.WithContext(ctx).
		Model(&models.TerminalRecording{}).
		Where("user_id = ?", userID)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch with pagination
	err := query.
		Order("started_at DESC").
		Limit(limit).
		Offset(offset).
		Preload("Instance").
		Find(&recordings).Error
	if err != nil {
		return nil, 0, err
	}

	return recordings, total, nil
}

// ListByInstance retrieves terminal recordings for a specific Docker instance
func (r *TerminalRecordingRepository) ListByInstance(ctx context.Context, instanceID uuid.UUID, limit, offset int) ([]*models.TerminalRecording, int64, error) {
	var recordings []*models.TerminalRecording
	var total int64

	query := r.db.WithContext(ctx).
		Model(&models.TerminalRecording{}).
		Where("instance_id = ?", instanceID)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch with pagination
	err := query.
		Order("started_at DESC").
		Limit(limit).
		Offset(offset).
		Preload("Instance").
		Find(&recordings).Error
	if err != nil {
		return nil, 0, err
	}

	return recordings, total, nil
}

// DeleteOlderThan deletes recordings older than the specified time
// This method also deletes the associated recording files from disk
func (r *TerminalRecordingRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	// First, query all recordings to be deleted to get their file paths
	var recordings []*models.TerminalRecording
	if err := r.db.WithContext(ctx).
		Where("started_at < ?", before).
		Find(&recordings).Error; err != nil {
		return 0, err
	}

	// Delete recording files from disk
	deletedFiles := 0
	for _, recording := range recordings {
		if recording.StoragePath != "" {
			if err := os.Remove(recording.StoragePath); err != nil && !os.IsNotExist(err) {
				// Log error but continue - don't fail entire cleanup if one file is missing
				logrus.WithFields(logrus.Fields{
					"recording_id": recording.ID,
					"file_path":    recording.StoragePath,
					"error":        err,
				}).Warn("Failed to delete recording file")
			} else if err == nil {
				deletedFiles++
			}
		}
	}

	// Delete database records
	result := r.db.WithContext(ctx).
		Where("started_at < ?", before).
		Delete(&models.TerminalRecording{})

	if result.Error != nil {
		return 0, result.Error
	}

	logrus.WithFields(logrus.Fields{
		"db_records_deleted": result.RowsAffected,
		"files_deleted":      deletedFiles,
		"before_date":        before.Format("2006-01-02"),
	}).Debug("Terminal recording cleanup details")

	return result.RowsAffected, nil
}

// GetStatistics retrieves statistics about terminal recordings
func (r *TerminalRecordingRepository) GetStatistics(ctx context.Context) (*TerminalRecordingStatistics, error) {
	var stats TerminalRecordingStatistics

	// Get total count, duration, and size
	err := r.db.WithContext(ctx).
		Model(&models.TerminalRecording{}).
		Select("COUNT(*) as total_recordings, "+
			"COALESCE(SUM(duration), 0) as total_duration, "+
			"COALESCE(SUM(file_size), 0) as total_size, "+
			"COALESCE(AVG(duration), 0) as avg_duration, "+
			"COALESCE(AVG(file_size), 0) as avg_size").
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	// Get recordings created today
	today := time.Now().Truncate(24 * time.Hour)
	err = r.db.WithContext(ctx).
		Model(&models.TerminalRecording{}).
		Where("started_at >= ?", today).
		Count(&stats.RecordingsToday).Error
	if err != nil {
		return nil, err
	}

	return &stats, nil
}
