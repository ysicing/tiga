package database

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/managers"
)

// DatabaseHandler handles database operations
type DatabaseHandler struct {
	instanceRepo repository.InstanceRepository
}

// NewDatabaseHandler creates a new database handler
func NewDatabaseHandler(instanceRepo repository.InstanceRepository) *DatabaseHandler {
	return &DatabaseHandler{
		instanceRepo: instanceRepo,
	}
}

// ListDatabases handles GET /api/v1/database/instances/{id}/databases
func (h *DatabaseHandler) ListDatabases(c *gin.Context) {
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

	// Get manager based on type
	var databases []string
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

		databases, listErr = manager.ListDatabases(c.Request.Context())

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

		databases, listErr = manager.ListDatabases(c.Request.Context())
	}

	if listErr != nil {
		handlers.RespondInternalError(c, listErr)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"databases": databases,
		"count":     len(databases),
	})
}

// CreateDatabase handles POST /api/v1/database/instances/{id}/databases
func (h *DatabaseHandler) CreateDatabase(c *gin.Context) {
	instanceIDStr := c.Param("id")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	var request struct {
		Name      string `json:"name" binding:"required"`
		Charset   string `json:"charset"`
		Collation string `json:"collation"`
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

	// Create database based on type
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

		createErr = manager.CreateDatabase(c.Request.Context(), request.Name, request.Charset, request.Collation)

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

		createErr = manager.CreateDatabase(c.Request.Context(), request.Name)
	}

	if createErr != nil {
		handlers.RespondInternalError(c, createErr)
		return
	}

	handlers.RespondCreated(c, gin.H{
		"name": request.Name,
	})
}

// DeleteDatabase handles DELETE /api/v1/database/instances/{id}/databases/{database}
func (h *DatabaseHandler) DeleteDatabase(c *gin.Context) {
	instanceIDStr := c.Param("id")
	databaseName := c.Param("database")

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

	// Delete database based on type
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

		deleteErr = manager.DropDatabase(c.Request.Context(), databaseName)

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

		deleteErr = manager.DropDatabase(c.Request.Context(), databaseName)
	}

	if deleteErr != nil {
		handlers.RespondInternalError(c, deleteErr)
		return
	}

	handlers.RespondSuccessWithMessage(c, nil, "database deleted successfully")
}
