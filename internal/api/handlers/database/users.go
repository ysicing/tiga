package database

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/managers"
)

// UserHandler handles database user operations
type UserHandler struct {
	instanceRepo repository.InstanceRepository
}

// NewUserHandler creates a new user handler
func NewUserHandler(instanceRepo repository.InstanceRepository) *UserHandler {
	return &UserHandler{
		instanceRepo: instanceRepo,
	}
}

// ListUsers handles GET /api/v1/database/instances/{id}/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	instanceIDStr := c.Param("id")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Get instance
	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}

	// Check if instance is database type
	if instance.Type != "mysql" && instance.Type != "postgresql" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not database type"))
		return
	}

	// Get users based on type
	var users []map[string]interface{}
	var listErr error

	switch instance.Type {
	case "mysql":
		manager := managers.NewMySQLManager()
		if err := manager.Initialize(c.Request.Context(), instance); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		if err := manager.Connect(c.Request.Context()); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		defer manager.Disconnect(c.Request.Context())

		users, listErr = manager.ListUsers(c.Request.Context())

	case "postgresql":
		manager := managers.NewPostgreSQLManager()
		if err := manager.Initialize(c.Request.Context(), instance); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		if err := manager.Connect(c.Request.Context()); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		defer manager.Disconnect(c.Request.Context())

		users, listErr = manager.ListUsers(c.Request.Context())
	}

	if listErr != nil {
		handlers.RespondInternalError(c, listErr)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"users": users,
		"count": len(users),
	})
}

// CreateUser handles POST /api/v1/database/instances/{id}/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	instanceIDStr := c.Param("id")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	var request struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Host     string `json:"host"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Get instance
	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}

	// Check if instance is database type
	if instance.Type != "mysql" && instance.Type != "postgresql" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not database type"))
		return
	}

	// Set default host for MySQL if not provided
	if request.Host == "" {
		if instance.Type == "mysql" {
			request.Host = "%"
		}
	}

	// Create user based on type
	var createErr error

	switch instance.Type {
	case "mysql":
		manager := managers.NewMySQLManager()
		if err := manager.Initialize(c.Request.Context(), instance); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		if err := manager.Connect(c.Request.Context()); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		defer manager.Disconnect(c.Request.Context())

		createErr = manager.CreateUser(c.Request.Context(), request.Username, request.Password, request.Host)

	case "postgresql":
		manager := managers.NewPostgreSQLManager()
		if err := manager.Initialize(c.Request.Context(), instance); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		if err := manager.Connect(c.Request.Context()); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		defer manager.Disconnect(c.Request.Context())

		createErr = manager.CreateUser(c.Request.Context(), request.Username, request.Password)
	}

	if createErr != nil {
		handlers.RespondInternalError(c, createErr)
		return
	}

	handlers.RespondCreated(c, gin.H{
		"username": request.Username,
		"host":     request.Host,
	})
}

// DeleteUser handles DELETE /api/v1/database/instances/{id}/users/{user}
func (h *UserHandler) DeleteUser(c *gin.Context) {
	instanceIDStr := c.Param("id")
	username := c.Param("user")
	host := c.DefaultQuery("host", "%")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Get instance
	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}

	// Check if instance is database type
	if instance.Type != "mysql" && instance.Type != "postgresql" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not database type"))
		return
	}

	// Delete user based on type
	var deleteErr error

	switch instance.Type {
	case "mysql":
		manager := managers.NewMySQLManager()
		if err := manager.Initialize(c.Request.Context(), instance); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		if err := manager.Connect(c.Request.Context()); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		defer manager.Disconnect(c.Request.Context())

		deleteErr = manager.DropUser(c.Request.Context(), username, host)

	case "postgresql":
		manager := managers.NewPostgreSQLManager()
		if err := manager.Initialize(c.Request.Context(), instance); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		if err := manager.Connect(c.Request.Context()); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		defer manager.Disconnect(c.Request.Context())

		deleteErr = manager.DropUser(c.Request.Context(), username)
	}

	if deleteErr != nil {
		handlers.RespondInternalError(c, deleteErr)
		return
	}

	handlers.RespondSuccessWithMessage(c, nil, "user deleted successfully")
}

// GrantPrivileges handles POST /api/v1/database/instances/{id}/users/{user}/privileges
func (h *UserHandler) GrantPrivileges(c *gin.Context) {
	instanceIDStr := c.Param("id")
	username := c.Param("user")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	var request struct {
		Database   string   `json:"database" binding:"required"`
		Privileges []string `json:"privileges" binding:"required"`
		Host       string   `json:"host"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Get instance
	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}

	// Check if instance is database type
	if instance.Type != "mysql" && instance.Type != "postgresql" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not database type"))
		return
	}

	// Set default host for MySQL if not provided
	if request.Host == "" {
		if instance.Type == "mysql" {
			request.Host = "%"
		}
	}

	// Grant privileges based on type
	var grantErr error

	switch instance.Type {
	case "mysql":
		manager := managers.NewMySQLManager()
		if err := manager.Initialize(c.Request.Context(), instance); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		if err := manager.Connect(c.Request.Context()); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		defer manager.Disconnect(c.Request.Context())

		grantErr = manager.GrantPrivileges(c.Request.Context(), username, request.Host, request.Database, request.Privileges)

	case "postgresql":
		manager := managers.NewPostgreSQLManager()
		if err := manager.Initialize(c.Request.Context(), instance); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		if err := manager.Connect(c.Request.Context()); err != nil {
			handlers.RespondInternalError(c, err)
			return
		}
		defer manager.Disconnect(c.Request.Context())

		grantErr = manager.GrantPrivileges(c.Request.Context(), username, request.Database, request.Privileges)
	}

	if grantErr != nil {
		handlers.RespondInternalError(c, grantErr)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"username":   username,
		"database":   request.Database,
		"privileges": request.Privileges,
	})
}
