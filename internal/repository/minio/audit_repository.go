package minio

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

type AuditRepository struct{ db *gorm.DB }

func NewAuditRepository(db *gorm.DB) *AuditRepository { return &AuditRepository{db: db} }

func (r *AuditRepository) Create(ctx context.Context, log *models.MinIOAuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

type AuditFilter struct {
	InstanceID uuid.UUID
	From, To   *time.Time
	OperatorID *uuid.UUID
	Resource   *string
}

func (r *AuditRepository) List(ctx context.Context, f AuditFilter, page, pageSize int) ([]*models.MinIOAuditLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.MinIOAuditLog{}).Where("instance_id = ?", f.InstanceID)
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at <= ?", *f.To)
	}
	if f.OperatorID != nil {
		q = q.Where("operator_id = ?", *f.OperatorID)
	}
	if f.Resource != nil && *f.Resource != "" {
		q = q.Where("resource_name = ?", *f.Resource)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page > 0 && pageSize > 0 {
		q = q.Offset((page - 1) * pageSize).Limit(pageSize)
	}
	var items []*models.MinIOAuditLog
	if err := q.Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *AuditRepository) DeleteBefore(ctx context.Context, t time.Time) error {
	return r.db.WithContext(ctx).Where("created_at < ?", t).Delete(&models.MinIOAuditLog{}).Error
}
