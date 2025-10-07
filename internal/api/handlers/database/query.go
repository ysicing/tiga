package database

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/managers"
)

// QueryHandler handles database query operations
type QueryHandler struct {
	instanceRepo repository.InstanceRepository
}

// NewQueryHandler creates a new query handler
func NewQueryHandler(instanceRepo repository.InstanceRepository) *QueryHandler {
	return &QueryHandler{
		instanceRepo: instanceRepo,
	}
}

// ExecuteQuery handles POST /api/v1/database/instances/{id}/query
func (h *QueryHandler) ExecuteQuery(c *gin.Context) {
	instanceIDStr := c.Param("id")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	var request struct {
		Query    string `json:"query" binding:"required"`
		Database string `json:"database"`
		Limit    int    `json:"limit"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Set default limit to prevent large result sets
	if request.Limit <= 0 || request.Limit > 1000 {
		request.Limit = 100
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

	// Validate query for safety
	if err := h.validateQuery(request.Query); err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Execute query based on type
	var result *managers.QueryResult
	var queryErr error

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

		result, queryErr = manager.ExecuteQuery(c.Request.Context(), request.Database, request.Query, request.Limit)

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

		result, queryErr = manager.ExecuteQuery(c.Request.Context(), request.Database, request.Query, request.Limit)
	}

	if queryErr != nil {
		handlers.RespondInternalError(c, queryErr)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"columns":        result.Columns,
		"rows":           result.Rows,
		"affected_rows":  result.AffectedRows,
		"row_count":      result.RowCount,
		"execution_time": result.ExecutionTime,
	})
}

// validateQuery performs basic query validation to prevent dangerous operations
func (h *QueryHandler) validateQuery(query string) error {
	query = strings.TrimSpace(strings.ToUpper(query))

	// Block dangerous keywords
	dangerousKeywords := []string{
		"DROP DATABASE",
		"DROP SCHEMA",
		"TRUNCATE",
		"DELETE FROM mysql.",
		"DELETE FROM pg_",
		"UPDATE mysql.",
		"UPDATE pg_",
		"GRANT",
		"REVOKE",
		"CREATE USER",
		"ALTER USER",
		"DROP USER",
	}

	for _, keyword := range dangerousKeywords {
		if strings.Contains(query, keyword) {
			return fmt.Errorf("query contains dangerous keyword: %s", keyword)
		}
	}

	return nil
}
