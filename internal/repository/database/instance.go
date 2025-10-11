package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// InstanceRepository manages DatabaseInstance persistence.
type InstanceRepository struct {
	db *gorm.DB
}

// NewInstanceRepository creates a new repository instance.
func NewInstanceRepository(db *gorm.DB) *InstanceRepository {
	return &InstanceRepository{db: db}
}

// Create persists a new database instance.
func (r *InstanceRepository) Create(ctx context.Context, instance *models.DatabaseInstance) error {
	if err := r.db.WithContext(ctx).Create(instance).Error; err != nil {
		return fmt.Errorf("failed to create database instance: %w", err)
	}
	return nil
}

// GetByID retrieves an instance by ID and preloads related aggregates.
func (r *InstanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DatabaseInstance, error) {
	var instance models.DatabaseInstance
	err := r.db.WithContext(ctx).
		Preload("Databases").
		Preload("Users").
		First(&instance, "id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("database instance not found")
		}
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	return &instance, nil
}

// GetByName retrieves an instance by its unique name.
func (r *InstanceRepository) GetByName(ctx context.Context, name string) (*models.DatabaseInstance, error) {
	var instance models.DatabaseInstance
	err := r.db.WithContext(ctx).
		Preload("Databases").
		Preload("Users").
		First(&instance, "name = ?", name).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("database instance not found")
		}
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	return &instance, nil
}

// List returns all database instances with eager loaded relations.
func (r *InstanceRepository) List(ctx context.Context) ([]*models.DatabaseInstance, error) {
	var instances []*models.DatabaseInstance
	if err := r.db.WithContext(ctx).
		Preload("Databases").
		Preload("Users").
		Order("created_at DESC").
		Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("failed to list database instances: %w", err)
	}
	return instances, nil
}

// ListByType returns instances filtered by type (mysql|postgresql|redis).
func (r *InstanceRepository) ListByType(ctx context.Context, instanceType string) ([]*models.DatabaseInstance, error) {
	var instances []*models.DatabaseInstance
	if err := r.db.WithContext(ctx).
		Preload("Databases").
		Preload("Users").
		Where("type = ?", instanceType).
		Order("created_at DESC").
		Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("failed to list database instances by type: %w", err)
	}
	return instances, nil
}

// Update persists modifications to an existing instance.
func (r *InstanceRepository) Update(ctx context.Context, instance *models.DatabaseInstance) error {
	if err := r.db.WithContext(ctx).Save(instance).Error; err != nil {
		return fmt.Errorf("failed to update database instance: %w", err)
	}
	return nil
}

// Delete performs a soft delete on the instance.
func (r *InstanceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.DatabaseInstance{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete database instance: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("database instance not found")
	}
	return nil
}
