package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// DatabaseRepository manages logical database records.
type DatabaseRepository struct {
	db *gorm.DB
}

// NewDatabaseRepository creates a repository for database entities.
func NewDatabaseRepository(db *gorm.DB) *DatabaseRepository {
	return &DatabaseRepository{db: db}
}

// Create persists a new database metadata record.
func (r *DatabaseRepository) Create(ctx context.Context, database *models.Database) error {
	if err := r.db.WithContext(ctx).Create(database).Error; err != nil {
		return fmt.Errorf("failed to create database record: %w", err)
	}
	return nil
}

// GetByID fetches a database record by ID.
func (r *DatabaseRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Database, error) {
	var database models.Database
	err := r.db.WithContext(ctx).
		Preload("Instance").
		First(&database, "id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("database not found")
		}
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	return &database, nil
}

// ListByInstance returns all databases for a given instance.
func (r *DatabaseRepository) ListByInstance(ctx context.Context, instanceID uuid.UUID) ([]*models.Database, error) {
	var databases []*models.Database
	if err := r.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("name ASC").
		Find(&databases).Error; err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	return databases, nil
}

// CheckUniqueName verifies uniqueness of a database name within an instance.
func (r *DatabaseRepository) CheckUniqueName(ctx context.Context, instanceID uuid.UUID, name string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Database{}).
		Where("instance_id = ? AND name = ?", instanceID, name).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check database uniqueness: %w", err)
	}
	return count == 0, nil
}

// Delete removes a database record.
func (r *DatabaseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Database{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete database record: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("database not found")
	}
	return nil
}
