package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// OAuthProviderRepository handles OAuth provider data operations
type OAuthProviderRepository struct {
	db *gorm.DB
}

// NewOAuthProviderRepository creates a new OAuthProviderRepository
func NewOAuthProviderRepository(db *gorm.DB) *OAuthProviderRepository {
	return &OAuthProviderRepository{db: db}
}

// Create creates a new OAuth provider
func (r *OAuthProviderRepository) Create(ctx context.Context, provider *models.OAuthProvider) error {
	if err := r.db.WithContext(ctx).Create(provider).Error; err != nil {
		return fmt.Errorf("failed to create OAuth provider: %w", err)
	}
	return nil
}

// GetByID retrieves an OAuth provider by ID
func (r *OAuthProviderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.OAuthProvider, error) {
	var provider models.OAuthProvider
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&provider).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("OAuth provider not found")
		}
		return nil, fmt.Errorf("failed to get OAuth provider: %w", err)
	}

	return &provider, nil
}

// GetByName retrieves an OAuth provider by name
func (r *OAuthProviderRepository) GetByName(ctx context.Context, name string) (*models.OAuthProvider, error) {
	var provider models.OAuthProvider
	err := r.db.WithContext(ctx).
		Where("LOWER(name) = LOWER(?) AND enabled = ?", name, true).
		First(&provider).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("OAuth provider not found")
		}
		return nil, fmt.Errorf("failed to get OAuth provider: %w", err)
	}

	return &provider, nil
}

// List retrieves all OAuth providers
func (r *OAuthProviderRepository) List(ctx context.Context) ([]*models.OAuthProvider, error) {
	var providers []*models.OAuthProvider
	if err := r.db.WithContext(ctx).Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("failed to list OAuth providers: %w", err)
	}
	return providers, nil
}

// ListEnabled retrieves only enabled OAuth providers
func (r *OAuthProviderRepository) ListEnabled(ctx context.Context) ([]*models.OAuthProvider, error) {
	var providers []*models.OAuthProvider
	if err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("failed to list enabled OAuth providers: %w", err)
	}
	return providers, nil
}

// Update updates an OAuth provider
func (r *OAuthProviderRepository) Update(ctx context.Context, provider *models.OAuthProvider) error {
	if err := r.db.WithContext(ctx).Save(provider).Error; err != nil {
		return fmt.Errorf("failed to update OAuth provider: %w", err)
	}
	return nil
}

// UpdateFields updates specific fields of an OAuth provider
func (r *OAuthProviderRepository) UpdateFields(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&models.OAuthProvider{}).
		Where("id = ?", id).
		Updates(fields)

	if result.Error != nil {
		return fmt.Errorf("failed to update OAuth provider fields: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("OAuth provider not found")
	}

	return nil
}

// Delete soft deletes an OAuth provider
func (r *OAuthProviderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.OAuthProvider{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete OAuth provider: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("OAuth provider not found")
	}

	return nil
}

// SetEnabled sets the enabled flag for an OAuth provider
func (r *OAuthProviderRepository) SetEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	return r.UpdateFields(ctx, id, map[string]interface{}{"enabled": enabled})
}
