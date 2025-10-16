package minio

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

type PermissionRepository struct{ db *gorm.DB }

func NewPermissionRepository(db *gorm.DB) *PermissionRepository { return &PermissionRepository{db: db} }

func (r *PermissionRepository) Create(ctx context.Context, p *models.BucketPermission) error {
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return fmt.Errorf("create permission: %w", err)
	}
	return nil
}

func (r *PermissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.BucketPermission, error) {
	var p models.BucketPermission
	if err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PermissionRepository) List(ctx context.Context, instanceID uuid.UUID, userID *uuid.UUID, bucket *string) ([]*models.BucketPermission, error) {
	q := r.db.WithContext(ctx).Where("instance_id = ?", instanceID)
	if userID != nil {
		q = q.Where("user_id = ?", *userID)
	}
	if bucket != nil && *bucket != "" {
		q = q.Where("bucket_name = ?", *bucket)
	}
	var items []*models.BucketPermission
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PermissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&models.BucketPermission{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *PermissionRepository) GetByUserAndBucket(ctx context.Context, userID uuid.UUID, bucket, prefix string) (*models.BucketPermission, error) {
	var p models.BucketPermission
	if err := r.db.WithContext(ctx).Where("user_id = ? AND bucket_name = ? AND prefix = ?", userID, bucket, prefix).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}
