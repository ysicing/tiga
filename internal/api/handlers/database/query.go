package database

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/api/middleware"
	dbservices "github.com/ysicing/tiga/internal/services/database"
)

// QueryHandler executes SQL/Redis queries.
type QueryHandler struct {
	executor *dbservices.QueryExecutor
	audit    *dbservices.AuditLogger
}

// NewQueryHandler constructs a QueryHandler.
func NewQueryHandler(executor *dbservices.QueryExecutor, audit *dbservices.AuditLogger) *QueryHandler {
	return &QueryHandler{
		executor: executor,
		audit:    audit,
	}
}

type executeQueryRequest struct {
	Query    string `json:"query" binding:"required"`
	Database string `json:"database"`
	Limit    int    `json:"limit"`
}

// ExecuteQuery handles POST /api/v1/database/instances/{id}/query
func (h *QueryHandler) ExecuteQuery(c *gin.Context) {
	instanceID, err := handlers.ParseUUID(c.Param("id"))
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	var req executeQueryRequest
	if !handlers.BindJSON(c, &req) {
		return
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		handlers.RespondUnauthorized(c, err)
		return
	}

	result, execErr := h.executor.ExecuteQuery(c.Request.Context(), dbservices.QueryExecutionRequest{
		InstanceID:   instanceID,
		ExecutedBy:   userID.String(),
		DatabaseName: req.Database,
		Query:        req.Query,
		Limit:        req.Limit,
		ClientIP:     c.ClientIP(),
	})

	entry := dbservices.AuditEntry{
		InstanceID: &instanceID,
		Action:     "query.execute",
		TargetType: "query",
		TargetName: instanceID.String(),
		Details: map[string]interface{}{
			"database": req.Database,
		},
		Success: execErr == nil,
		Error:   execErr,
	}
	entry.Operator = userID.String()

	if execErr != nil {
		status := http.StatusInternalServerError
		if errors.Is(execErr, dbservices.ErrSQLDangerousOperation) ||
			errors.Is(execErr, dbservices.ErrSQLDangerousFunction) ||
			errors.Is(execErr, dbservices.ErrSQLMissingWhere) ||
			errors.Is(execErr, dbservices.ErrRedisDangerousCommand) {
			status = http.StatusBadRequest
			entry.Action = "query.blocked"
		} else if errors.Is(execErr, context.DeadlineExceeded) {
			status = http.StatusGatewayTimeout
		} else if strings.Contains(execErr.Error(), "row limit exceeded") {
			status = http.StatusBadRequest
			entry.Action = "query.row_limit_exceeded"
		}

		h.logAudit(c, entry)
		handlers.RespondError(c, status, execErr)
		return
	}

	h.logAudit(c, entry)

	payload := gin.H{
		"columns":        result.Columns,
		"rows":           result.Rows,
		"affected_rows":  result.AffectedRows,
		"row_count":      result.RowCount,
		"execution_time": result.ExecutionTime.Milliseconds(),
		"truncated":      result.Truncated,
	}
	if result.Message != "" {
		payload["message"] = result.Message
	}

	handlers.RespondSuccess(c, payload)
}

func (h *QueryHandler) logAudit(c *gin.Context, entry dbservices.AuditEntry) {
	if h.audit == nil {
		return
	}
	entry.ClientIP = c.ClientIP()
	if err := h.audit.LogAction(c.Request.Context(), entry); err != nil {
		logrus.WithError(err).Warn("failed to write database audit log")
	}
}
