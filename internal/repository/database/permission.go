package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// PermissionRepository manages permission policies between users and databases.
type PermissionRepository struct {
	db *gorm.DB
}

// NewPermissionRepository creates a new permission repository.
func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

// Grant stores a new permission policy. Duplicate checks should be performed before calling.
func (r *PermissionRepository) Grant(ctx context.Context, policy *models.PermissionPolicy) error {
	if policy.GrantedAt.IsZero() {
		policy.GrantedAt = time.Now().UTC()
	}

	if err := r.db.WithContext(ctx).Create(policy).Error; err != nil {
		return fmt.Errorf("failed to grant permission: %w", err)
	}
	return nil
}

// GetByID returns a permission policy by identifier.
func (r *PermissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.PermissionPolicy, error) {
	var policy models.PermissionPolicy
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Database").
		First(&policy, "id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("permission not found")
		}
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return &policy, nil
}

// Revoke marks a permission as revoked by setting RevokedAt timestamp.
func (r *PermissionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&models.PermissionPolicy{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now)

	if result.Error != nil {
		return fmt.Errorf("failed to revoke permission: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("permission not found or already revoked")
	}

	return nil
}

// ListByUser returns active permissions for a user.
func (r *PermissionRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.PermissionPolicy, error) {
	var permissions []*models.PermissionPolicy
	if err := r.db.WithContext(ctx).
		Preload("Database").
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Order("created_at DESC").
		Find(&permissions).Error; err != nil {
		return nil, fmt.Errorf("failed to list permissions by user: %w", err)
	}
	return permissions, nil
}

// ListByDatabase returns active permissions for a database.
func (r *PermissionRepository) ListByDatabase(ctx context.Context, databaseID uuid.UUID) ([]*models.PermissionPolicy, error) {
	var permissions []*models.PermissionPolicy
	if err := r.db.WithContext(ctx).
		Preload("User").
		Where("database_id = ? AND revoked_at IS NULL", databaseID).
		Order("created_at DESC").
		Find(&permissions).Error; err != nil {
		return nil, fmt.Errorf("failed to list permissions by database: %w", err)
	}
	return permissions, nil
}

// CheckExisting verifies active permission existence to avoid duplicates.
func (r *PermissionRepository) CheckExisting(ctx context.Context, userID, databaseID uuid.UUID, role string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.PermissionPolicy{}).
		Where("user_id = ? AND database_id = ? AND role = ? AND revoked_at IS NULL", userID, databaseID, role).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}
	return count > 0, nil
}
