package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/api/middleware"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/auth"
)

// UserHandler handles user management endpoints
type UserHandler struct {
	userRepo       *repository.UserRepository
	rbacService    *auth.RBACService
	passwordHasher *auth.PasswordHasher
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	userRepo *repository.UserRepository,
	rbacService *auth.RBACService,
) *UserHandler {
	return &UserHandler{
		userRepo:       userRepo,
		rbacService:    rbacService,
		passwordHasher: auth.NewPasswordHasher(),
	}
}

// ListUsersRequest represents a request to list users
type ListUsersRequest struct {
	Status   string   `form:"status"`
	Roles    []string `form:"roles"`
	Search   string   `form:"search"`
	Page     int      `form:"page"`
	PageSize int      `form:"page_size" binding:"max=100"`
}

// ListUsers lists all users with pagination
// @Summary List users
// @Description Get a paginated list of users
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status"
// @Param roles query array false "Filter by roles"
// @Param search query string false "Search in username, email, full_name"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} PaginatedResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	var req ListUsersRequest
	if !BindQuery(c, &req) {
		return
	}

	// Set defaults
	req.Page = defaultInt(req.Page, 1)
	req.PageSize = defaultInt(req.PageSize, 20)
	req.PageSize = clamp(req.PageSize, 1, 100)

	// Build filter
	filter := &repository.ListUsersFilter{
		Status:   req.Status,
		Roles:    req.Roles,
		Search:   req.Search,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	// Get users
	users, total, err := h.userRepo.ListUsers(c.Request.Context(), filter)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondPaginated(c, users, req.Page, req.PageSize, total)
}

// GetUserRequest represents a request to get a user
type GetUserRequest struct {
	UserID string `uri:"user_id" binding:"required,uuid"`
}

// GetUser gets a user by ID
// @Summary Get user
// @Description Get user details by ID
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/users/{user_id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	var req GetUserRequest
	if !BindURI(c, &req) {
		return
	}

	userID, err := ParseUUID(req.UserID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		RespondNotFound(c, err)
		return
	}

	RespondSuccess(c, user)
}

// CreateUserRequest represents a request to create a user
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
	Status   string `json:"status" binding:"oneof=active suspended"`
}

// CreateUser creates a new user
// @Summary Create user
// @Description Create a new user (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateUserRequest true "User creation request"
// @Success 201 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if !BindJSON(c, &req) {
		return
	}

	// Check if username exists
	exists, err := h.userRepo.ExistsUsername(c.Request.Context(), req.Username, nil)
	if err != nil {
		RespondInternalError(c, err)
		return
	}
	if exists {
		RespondConflict(c, fmt.Errorf("username already exists"))
		return
	}

	// Check if email exists
	exists, err = h.userRepo.ExistsEmail(c.Request.Context(), req.Email, nil)
	if err != nil {
		RespondInternalError(c, err)
		return
	}
	if exists {
		RespondConflict(c, fmt.Errorf("email already exists"))
		return
	}

	// Hash password
	passwordHash, err := h.passwordHasher.Hash(req.Password)
	if err != nil {
		RespondInternalError(c, fmt.Errorf("failed to hash password: %w", err))
		return
	}

	// Create user
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: passwordHash,
		FullName: req.FullName,
		Status:   defaultIfEmpty(req.Status, "active"),
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondCreated(c, user)
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	UserID    string  `uri:"user_id" binding:"required,uuid"`
	Email     *string `json:"email,omitempty" binding:"omitempty,email"`
	FullName  *string `json:"full_name,omitempty"`
	Status    *string `json:"status,omitempty" binding:"omitempty,oneof=active suspended"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// UpdateUser updates a user
// @Summary Update user
// @Description Update user details
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Param request body UpdateUserRequest true "User update request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/users/{user_id} [patch]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	var req UpdateUserRequest
	if !BindURI(c, &req) {
		return
	}
	if !BindJSON(c, &req) {
		return
	}

	userID, err := ParseUUID(req.UserID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	// Get existing user
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		RespondNotFound(c, err)
		return
	}

	// Build updates
	updates := make(map[string]interface{})
	if req.Email != nil {
		// Check email uniqueness
		exists, err := h.userRepo.ExistsEmail(c.Request.Context(), *req.Email, &userID)
		if err != nil {
			RespondInternalError(c, err)
			return
		}
		if exists {
			RespondConflict(c, fmt.Errorf("email already exists"))
			return
		}
		updates["email"] = *req.Email
	}
	if req.FullName != nil {
		updates["full_name"] = *req.FullName
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.AvatarURL != nil {
		updates["avatar_url"] = *req.AvatarURL
	}

	// Update user
	if len(updates) > 0 {
		if err := h.userRepo.UpdateFields(c.Request.Context(), userID, updates); err != nil {
			RespondInternalError(c, err)
			return
		}
	}

	// Get updated user
	user, err = h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, user)
}

// DeleteUser deletes a user
// @Summary Delete user
// @Description Soft delete a user (admin only)
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/users/{user_id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	var req GetUserRequest
	if !BindURI(c, &req) {
		return
	}

	userID, err := ParseUUID(req.UserID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	// Cannot delete self
	currentUserID, _ := middleware.GetUserID(c)
	if currentUserID == userID {
		RespondBadRequest(c, fmt.Errorf("cannot delete your own account"))
		return
	}

	if err := h.userRepo.Delete(c.Request.Context(), userID); err != nil {
		RespondNotFound(c, err)
		return
	}

	RespondNoContent(c)
}

// GetUserRoles gets roles for a user
// @Summary Get user roles
// @Description Get all roles assigned to a user
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/users/{user_id}/roles [get]
func (h *UserHandler) GetUserRoles(c *gin.Context) {
	var req GetUserRequest
	if !BindURI(c, &req) {
		return
	}

	userID, err := ParseUUID(req.UserID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	roles, err := h.userRepo.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, roles)
}

// AssignRoleRequest represents a request to assign a role
type AssignRoleRequest struct {
	UserID string `uri:"user_id" binding:"required,uuid"`
	RoleID string `json:"role_id" binding:"required,uuid"`
}

// AssignRole assigns a role to a user
// @Summary Assign role
// @Description Assign a role to a user (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Param request body AssignRoleRequest true "Role assignment request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/users/{user_id}/roles [post]
func (h *UserHandler) AssignRole(c *gin.Context) {
	var req AssignRoleRequest
	if !BindURI(c, &req) {
		return
	}
	if !BindJSON(c, &req) {
		return
	}

	userID, err := ParseUUID(req.UserID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	roleID, err := ParseUUID(req.RoleID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	// Get granter ID
	grantedBy, err := middleware.GetUserID(c)
	if err != nil {
		RespondUnauthorized(c, err)
		return
	}

	// Assign role
	if err := h.rbacService.AssignRole(c.Request.Context(), userID, roleID, grantedBy); err != nil {
		RespondBadRequest(c, err)
		return
	}

	RespondSuccessWithMessage(c, nil, "role assigned successfully")
}

// RevokeRoleRequest represents a request to revoke a role
type RevokeRoleRequest struct {
	UserID string `uri:"user_id" binding:"required,uuid"`
	RoleID string `uri:"role_id" binding:"required,uuid"`
}

// RevokeRole revokes a role from a user
// @Summary Revoke role
// @Description Revoke a role from a user (admin only)
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Param role_id path string true "Role ID (UUID)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/users/{user_id}/roles/{role_id} [delete]
func (h *UserHandler) RevokeRole(c *gin.Context) {
	var req RevokeRoleRequest
	if !BindURI(c, &req) {
		return
	}

	userID, err := ParseUUID(req.UserID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	roleID, err := ParseUUID(req.RoleID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	// Revoke role
	if err := h.rbacService.RevokeRole(c.Request.Context(), userID, roleID); err != nil {
		RespondBadRequest(c, err)
		return
	}

	RespondSuccessWithMessage(c, nil, "role revoked successfully")
}
