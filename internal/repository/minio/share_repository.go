package minio

import (
    "context"
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"

    "github.com/ysicing/tiga/internal/models"
)

type ShareRepository struct { db *gorm.DB }
func NewShareRepository(db *gorm.DB) *ShareRepository { return &ShareRepository{db: db} }

func (r *ShareRepository) Create(ctx context.Context, s *models.MinIOShareLink) error { return r.db.WithContext(ctx).Create(s).Error }
func (r *ShareRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.MinIOShareLink, error) {
    var s models.MinIOShareLink; if err := r.db.WithContext(ctx).First(&s, "id = ?", id).Error; err != nil { return nil, err }; return &s, nil
}
func (r *ShareRepository) List(ctx context.Context, instanceID uuid.UUID, createdBy *uuid.UUID) ([]*models.MinIOShareLink, error) {
    q := r.db.WithContext(ctx).Where("instance_id = ?", instanceID)
    if createdBy != nil { q = q.Where("created_by = ?", *createdBy) }
    var items []*models.MinIOShareLink; if err := q.Order("created_at DESC").Find(&items).Error; err != nil { return nil, err }
    return items, nil
}
func (r *ShareRepository) ListAll(ctx context.Context) ([]*models.MinIOShareLink, error) {
    var items []*models.MinIOShareLink
    if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&items).Error; err != nil { return nil, err }
    return items, nil
}
func (r *ShareRepository) Revoke(ctx context.Context, id uuid.UUID) error { return r.db.WithContext(ctx).Model(&models.MinIOShareLink{}).Where("id = ?", id).Update("status", "revoked").Error }
func (r *ShareRepository) DeleteExpired(ctx context.Context) error { return r.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&models.MinIOShareLink{}).Error }
