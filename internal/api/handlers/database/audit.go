package database

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/api/handlers"
	dbrepo "github.com/ysicing/tiga/internal/repository/database"
)

// AuditHandler exposes audit log queries for database operations.
type AuditHandler struct {
	repo *dbrepo.AuditLogRepository
}

// NewAuditHandler constructs an AuditHandler.
func NewAuditHandler(repo *dbrepo.AuditLogRepository) *AuditHandler {
	return &AuditHandler{repo: repo}
}

// ListAuditLogs handles GET /api/v1/database/audit-logs
func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	filter, page, pageSize, err := h.parseFilter(c)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	logs, total, err := h.repo.Filter(c.Request.Context(), filter)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondPaginated(c, logs, page, pageSize, total)
}

func (h *AuditHandler) parseFilter(c *gin.Context) (*dbrepo.AuditLogFilter, int, int, error) {
	filter := &dbrepo.AuditLogFilter{}

	var instanceIDPtr *uuid.UUID
	if instanceIDStr := c.Query("instance_id"); instanceIDStr != "" {
		instanceID, err := handlers.ParseUUID(instanceIDStr)
		if err != nil {
			return nil, 0, 0, err
		}
		instanceIDPtr = &instanceID
	}

	filter.InstanceID = instanceIDPtr
	filter.Operator = c.Query("operator")
	filter.Action = c.Query("action")

	if start := c.Query("start"); start != "" {
		t, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return nil, 0, 0, err
		}
		filter.StartDate = &t
	}

	if end := c.Query("end"); end != "" {
		t, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return nil, 0, 0, err
		}
		filter.EndDate = &t
	}

	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 50)

	filter.Page = page
	filter.PageSize = pageSize

	return filter, page, pageSize, nil
}

func parsePositiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
