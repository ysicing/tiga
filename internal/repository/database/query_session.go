package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// QuerySessionRepository persists query execution sessions.
type QuerySessionRepository struct {
	db *gorm.DB
}

// NewQuerySessionRepository creates a query session repository.
func NewQuerySessionRepository(db *gorm.DB) *QuerySessionRepository {
	return &QuerySessionRepository{db: db}
}

// Create inserts a new query session record.
func (r *QuerySessionRepository) Create(ctx context.Context, session *models.QuerySession) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("failed to create query session: %w", err)
	}
	return nil
}
