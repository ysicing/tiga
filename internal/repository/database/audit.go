package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// AuditLogRepository handles persistence for database audit logs.
type AuditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository constructs a new audit log repository.
func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create stores a new audit log entry.
func (r *AuditLogRepository) Create(ctx context.Context, log *models.DatabaseAuditLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("failed to create database audit log: %w", err)
	}
	return nil
}

// List returns paginated audit logs without additional filters.
func (r *AuditLogRepository) List(ctx context.Context, page, pageSize int) ([]*models.DatabaseAuditLog, int64, error) {
	filter := &AuditLogFilter{
		Page:     page,
		PageSize: pageSize,
	}
	return r.Filter(ctx, filter)
}

// AuditLogFilter defines filter options for audit log queries.
type AuditLogFilter struct {
	InstanceID *uuid.UUID
	Operator   string
	Action     string
	StartDate  *time.Time
	EndDate    *time.Time
	Page       int
	PageSize   int
}

// Filter returns filtered and paginated audit logs plus total count.
func (r *AuditLogRepository) Filter(ctx context.Context, filter *AuditLogFilter) ([]*models.DatabaseAuditLog, int64, error) {
	if filter == nil {
		filter = &AuditLogFilter{}
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	query := r.db.WithContext(ctx).Model(&models.DatabaseAuditLog{})

	if filter.InstanceID != nil {
		query = query.Where("instance_id = ?", *filter.InstanceID)
	}

	if filter.Operator != "" {
		query = query.Where("operator = ?", filter.Operator)
	}

	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}

	if filter.StartDate != nil {
		query = query.Where("created_at >= ?", *filter.StartDate)
	}

	if filter.EndDate != nil {
		query = query.Where("created_at <= ?", *filter.EndDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count database audit logs: %w", err)
	}

	offset := (page - 1) * pageSize
	var logs []*models.DatabaseAuditLog
	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list database audit logs: %w", err)
	}

	return logs, total, nil
}

// DeleteOldLogs removes audit logs created before the specified cutoff time.
func (r *AuditLogRepository) DeleteOldLogs(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Delete(&models.DatabaseAuditLog{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old database audit logs: %w", result.Error)
	}

	return result.RowsAffected, nil
}
