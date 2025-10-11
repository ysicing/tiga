package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// UserRepository manages database user metadata.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new database user repository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create persists a new database user.
func (r *UserRepository) Create(ctx context.Context, user *models.DatabaseUser) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("failed to create database user: %w", err)
	}
	return nil
}

// GetByID retrieves a database user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DatabaseUser, error) {
	var user models.DatabaseUser
	err := r.db.WithContext(ctx).
		Preload("Instance").
		First(&user, "id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("database user not found")
		}
		return nil, fmt.Errorf("failed to get database user: %w", err)
	}

	return &user, nil
}

// GetByUsername finds a user by instance and username.
func (r *UserRepository) GetByUsername(ctx context.Context, instanceID uuid.UUID, username string) (*models.DatabaseUser, error) {
	var user models.DatabaseUser
	err := r.db.WithContext(ctx).
		Preload("Instance").
		First(&user, "instance_id = ? AND username = ?", instanceID, username).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("database user not found")
		}
		return nil, fmt.Errorf("failed to get database user: %w", err)
	}

	return &user, nil
}

// ListByInstance lists users for a given instance.
func (r *UserRepository) ListByInstance(ctx context.Context, instanceID uuid.UUID) ([]*models.DatabaseUser, error) {
	var users []*models.DatabaseUser
	if err := r.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("username ASC").
		Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to list database users: %w", err)
	}
	return users, nil
}

// Update persists updates to a database user.
func (r *UserRepository) Update(ctx context.Context, user *models.DatabaseUser) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("failed to update database user: %w", err)
	}
	return nil
}

// Delete removes a database user record.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.DatabaseUser{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete database user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("database user not found")
	}
	return nil
}
