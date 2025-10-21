package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// UserRepository handles user data operations
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Where("username = ?", username).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByUsernameOrEmail retrieves a user by username or email
func (r *UserRepository) GetByUsernameOrEmail(ctx context.Context, identifier string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Where("username = ? OR email = ?", identifier, identifier).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// UpdateFields updates specific fields of a user
func (r *UserRepository) UpdateFields(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Updates(fields)

	if result.Error != nil {
		return fmt.Errorf("failed to update user fields: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// Delete soft deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.User{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// ListUsersFilter represents user list filters
type ListUsersFilter struct {
	Status   string   // Filter by status
	Roles    []string // Filter by role names
	Search   string   // Search in username, email, full_name
	Page     int      // Page number (1-based)
	PageSize int      // Page size
}

// ListUsers retrieves a paginated list of users with filters
func (r *UserRepository) ListUsers(ctx context.Context, filter *ListUsersFilter) ([]*models.User, int64, error) {
	query := r.db.WithContext(ctx)

	// Apply filters
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if len(filter.Roles) > 0 {
		query = query.Joins("JOIN user_roles ON user_roles.user_id = users.id").
			Joins("JOIN roles ON roles.id = user_roles.role_id").
			Where("roles.name IN ?", filter.Roles).
			Distinct()
	}

	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("username LIKE ? OR email LIKE ? OR full_name LIKE ?", search, search, search)
	}

	// Count total
	var total int64
	if err := query.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Fetch results
	var users []*models.User
	if err := query.
		Preload("Roles").
		Order("created_at DESC").
		Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

// CountByStatus counts users by status
func (r *UserRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("status = ?", status).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count users by status: %w", err)
	}

	return count, nil
}

// ExistsUsername checks if a username exists
func (r *UserRepository) ExistsUsername(ctx context.Context, username string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).Model(&models.User{}).Where("username = ?", username)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}

	return count > 0, nil
}

// ExistsEmail checks if an email exists
func (r *UserRepository) ExistsEmail(ctx context.Context, email string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).Model(&models.User{}).Where("email = ?", email)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return count > 0, nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Update("last_login_at", gorm.Expr("NOW()")).Error
}

// UpdateStatus updates user status
func (r *UserRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		return fmt.Errorf("failed to update user status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// GetUserRoles retrieves all roles for a user
func (r *UserRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*models.Role, error) {
	var userRoles []models.UserRole
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("user_id = ?", userID).
		Where("expires_at IS NULL OR expires_at > NOW()").
		Find(&userRoles).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	roles := make([]*models.Role, 0, len(userRoles))
	for _, ur := range userRoles {
		if ur.Role != nil && !ur.IsExpired() {
			roles = append(roles, ur.Role)
		}
	}

	return roles, nil
}

// Legacy compatibility methods

// GetBySub retrieves a user by OAuth Sub field (legacy compatibility)
func (r *UserRepository) GetBySub(ctx context.Context, sub string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Where("sub = ?", sub).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by sub: %w", err)
	}

	return &user, nil
}

// FindWithSubOrUpsert finds user by sub or creates/updates (legacy compatibility)
func (r *UserRepository) FindWithSubOrUpsert(ctx context.Context, user *models.User) error {
	if user.Sub == "" {
		return fmt.Errorf("user sub is empty")
	}

	existingUser, err := r.GetBySub(ctx, user.Sub)
	if err != nil {
		// User not found, create new
		return r.Create(ctx, user)
	}

	// User exists, update fields
	user.ID = existingUser.ID
	user.CreatedAt = existingUser.CreatedAt
	user.Enabled = existingUser.Enabled
	return r.Update(ctx, user)
}

// SetEnabled sets the enabled flag (legacy compatibility)
func (r *UserRepository) SetEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	return r.UpdateFields(ctx, id, map[string]interface{}{"enabled": enabled})
}

// Count returns total user count (legacy compatibility)
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error
	return count, err
}
