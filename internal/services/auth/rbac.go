package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// RBACService handles role-based access control
type RBACService struct {
	db *gorm.DB
}

// NewRBACService creates a new RBACService
func NewRBACService(db *gorm.DB) *RBACService {
	return &RBACService{
		db: db,
	}
}

// Permission represents a permission check
type Permission struct {
	Resource string   // Resource type (instance, user, role, etc.)
	Actions  []string // Actions (create, read, update, delete, *)
}

// CheckPermission checks if a user has permission to perform an action on a resource
func (s *RBACService) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	// Get user's roles
	roles, err := s.GetUserRoles(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user roles: %w", err)
	}

	// Check permissions for each role
	for _, role := range roles {
		if s.roleHasPermission(role, resource, action) {
			return true, nil
		}
	}

	return false, nil
}

// CheckPermissions checks if a user has all specified permissions
func (s *RBACService) CheckPermissions(ctx context.Context, userID uuid.UUID, permissions []Permission) (bool, error) {
	roles, err := s.GetUserRoles(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user roles: %w", err)
	}

	for _, perm := range permissions {
		hasPermission := false
		for _, action := range perm.Actions {
			for _, role := range roles {
				if s.roleHasPermission(role, perm.Resource, action) {
					hasPermission = true
					break
				}
			}
			if hasPermission {
				break
			}
		}
		if !hasPermission {
			return false, nil
		}
	}

	return true, nil
}

// CheckAnyPermission checks if a user has any of the specified permissions
func (s *RBACService) CheckAnyPermission(ctx context.Context, userID uuid.UUID, permissions []Permission) (bool, error) {
	roles, err := s.GetUserRoles(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user roles: %w", err)
	}

	for _, perm := range permissions {
		for _, action := range perm.Actions {
			for _, role := range roles {
				if s.roleHasPermission(role, perm.Resource, action) {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// GetUserRoles retrieves all active roles for a user
func (s *RBACService) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*models.Role, error) {
	var userRoles []models.UserRole
	err := s.db.WithContext(ctx).
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

// AssignRole assigns a role to a user
func (s *RBACService) AssignRole(ctx context.Context, userID, roleID, grantedBy uuid.UUID) error {
	// Check if role exists
	var role models.Role
	if err := s.db.WithContext(ctx).Where("id = ?", roleID).First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("role not found")
		}
		return fmt.Errorf("failed to find role: %w", err)
	}

	// Check if user already has the role
	var existingUserRole models.UserRole
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		First(&existingUserRole).Error

	if err == nil {
		// Role already assigned
		return fmt.Errorf("role already assigned to user")
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing role: %w", err)
	}

	// Assign role
	userRole := &models.UserRole{
		UserID:    userID,
		RoleID:    roleID,
		GrantedBy: &grantedBy,
	}

	if err := s.db.WithContext(ctx).Create(userRole).Error; err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	return nil
}

// RevokeRole revokes a role from a user
func (s *RBACService) RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error {
	result := s.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&models.UserRole{})

	if result.Error != nil {
		return fmt.Errorf("failed to revoke role: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("role assignment not found")
	}

	return nil
}

// HasRole checks if a user has a specific role
func (s *RBACService) HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Table("user_roles").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND roles.name = ?", userID, roleName).
		Where("user_roles.expires_at IS NULL OR user_roles.expires_at > NOW()").
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check role: %w", err)
	}

	return count > 0, nil
}

// IsAdmin checks if a user has the admin role
func (s *RBACService) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	return s.HasRole(ctx, userID, "admin")
}

// roleHasPermission checks if a role has permission for a resource and action
func (s *RBACService) roleHasPermission(role *models.Role, resource, action string) bool {
	// Parse permissions from JSONB
	permissions := role.Permissions
	if permissions == nil {
		return false
	}

	// Permissions format: [{"resource": "instance", "actions": ["create", "read", "update", "delete"]}]
	for _, perm := range permissions {
		permMap, ok := perm.(map[string]interface{})
		if !ok {
			continue
		}

		permResource, ok := permMap["resource"].(string)
		if !ok {
			continue
		}

		// Check if resource matches (exact match or wildcard)
		if permResource != resource && permResource != "*" {
			continue
		}

		// Check actions
		actions, ok := permMap["actions"].([]interface{})
		if !ok {
			continue
		}

		for _, a := range actions {
			actionStr, ok := a.(string)
			if !ok {
				continue
			}

			// Check if action matches (exact match or wildcard)
			if actionStr == action || actionStr == "*" {
				return true
			}
		}
	}

	return false
}

// GetUserPermissions retrieves all permissions for a user
func (s *RBACService) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]Permission, error) {
	roles, err := s.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	permissionsMap := make(map[string]map[string]bool) // resource -> action -> exists

	for _, role := range roles {
		permissions := role.Permissions
		if permissions == nil {
			continue
		}

		for _, perm := range permissions {
			permMap, ok := perm.(map[string]interface{})
			if !ok {
				continue
			}

			resource, ok := permMap["resource"].(string)
			if !ok {
				continue
			}

			if permissionsMap[resource] == nil {
				permissionsMap[resource] = make(map[string]bool)
			}

			actions, ok := permMap["actions"].([]interface{})
			if !ok {
				continue
			}

			for _, a := range actions {
				actionStr, ok := a.(string)
				if ok {
					permissionsMap[resource][actionStr] = true
				}
			}
		}
	}

	// Convert map to slice
	result := make([]Permission, 0, len(permissionsMap))
	for resource, actionsMap := range permissionsMap {
		actions := make([]string, 0, len(actionsMap))
		for action := range actionsMap {
			actions = append(actions, action)
		}
		result = append(result, Permission{
			Resource: resource,
			Actions:  actions,
		})
	}

	return result, nil
}

// RequirePermission returns an error if user doesn't have permission
func (s *RBACService) RequirePermission(ctx context.Context, userID uuid.UUID, resource, action string) error {
	hasPermission, err := s.CheckPermission(ctx, userID, resource, action)
	if err != nil {
		return err
	}

	if !hasPermission {
		return fmt.Errorf("permission denied: %s:%s", resource, action)
	}

	return nil
}

// RequireRole returns an error if user doesn't have the specified role
func (s *RBACService) RequireRole(ctx context.Context, userID uuid.UUID, roleName string) error {
	hasRole, err := s.HasRole(ctx, userID, roleName)
	if err != nil {
		return err
	}

	if !hasRole {
		return fmt.Errorf("role required: %s", roleName)
	}

	return nil
}

// RequireAdmin returns an error if user is not an admin
func (s *RBACService) RequireAdmin(ctx context.Context, userID uuid.UUID) error {
	return s.RequireRole(ctx, userID, "admin")
}

// GetRolePermissions retrieves permissions for a specific role
func (s *RBACService) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]Permission, error) {
	var role models.Role
	if err := s.db.WithContext(ctx).Where("id = ?", roleID).First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("role not found")
		}
		return nil, fmt.Errorf("failed to find role: %w", err)
	}

	permissions := role.Permissions
	if permissions == nil {
		return []Permission{}, nil
	}

	result := make([]Permission, 0, len(permissions))
	for _, perm := range permissions {
		permMap, ok := perm.(map[string]interface{})
		if !ok {
			continue
		}

		resource, ok := permMap["resource"].(string)
		if !ok {
			continue
		}

		actionsInterface, ok := permMap["actions"].([]interface{})
		if !ok {
			continue
		}

		actions := make([]string, 0, len(actionsInterface))
		for _, a := range actionsInterface {
			if actionStr, ok := a.(string); ok {
				actions = append(actions, actionStr)
			}
		}

		result = append(result, Permission{
			Resource: resource,
			Actions:  actions,
		})
	}

	return result, nil
}

// ValidatePermissionString validates a permission string format (resource:action)
func ValidatePermissionString(permStr string) (string, string, error) {
	parts := strings.Split(permStr, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid permission format: %s (expected resource:action)", permStr)
	}

	resource := strings.TrimSpace(parts[0])
	action := strings.TrimSpace(parts[1])

	if resource == "" || action == "" {
		return "", "", fmt.Errorf("resource and action cannot be empty")
	}

	return resource, action, nil
}
