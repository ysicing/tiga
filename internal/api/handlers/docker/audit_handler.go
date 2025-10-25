package docker

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/services/docker"

	basehandlers "github.com/ysicing/tiga/internal/api/handlers"
)

// AuditLogHandler handles Docker audit log API requests
type AuditLogHandler struct {
	auditService *docker.AuditLogService
}

// NewAuditLogHandler creates a new AuditLogHandler
func NewAuditLogHandler(auditService *docker.AuditLogService) *AuditLogHandler {
	return &AuditLogHandler{
		auditService: auditService,
	}
}

// GetDockerAuditLogs godoc
// @Summary Get Docker audit logs
// @Description Query Docker operation audit logs with pagination and filtering
// @Tags docker-audit
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20, max: 100)"
// @Param instance_id query string false "Filter by Docker instance ID (UUID)"
// @Param user query string false "Filter by username or user ID"
// @Param action query string false "Filter by operation type (e.g., container_start, image_pull)"
// @Param resource_type query string false "Filter by resource type (container, image)"
// @Param start_time query string false "Filter by start time (RFC3339 format, e.g., 2024-01-01T00:00:00Z)"
// @Param end_time query string false "Filter by end time (RFC3339 format, e.g., 2024-12-31T23:59:59Z)"
// @Param success query boolean false "Filter by operation result (true: success, false: failure)"
// @Success 200 {object} handlers.PaginatedResponse
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/audit-logs [get]
// @Security BearerAuth
func (h *AuditLogHandler) GetDockerAuditLogs(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Parse filter parameters
	instanceID := c.Query("instance_id")
	user := c.Query("user")
	action := c.Query("action")
	resourceType := c.Query("resource_type")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")
	successStr := c.Query("success")

	// Parse time range
	var startTime, endTime *time.Time
	if startTimeStr != "" {
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			logrus.WithError(err).WithField("start_time", startTimeStr).Warn("Invalid start_time format")
			basehandlers.RespondBadRequest(c, err)
			return
		}
		startTime = &t
	}
	if endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			logrus.WithError(err).WithField("end_time", endTimeStr).Warn("Invalid end_time format")
			basehandlers.RespondBadRequest(c, err)
			return
		}
		endTime = &t
	}

	// Parse success filter
	var success *bool
	if successStr != "" {
		s := successStr == "true"
		success = &s
	}

	// Build filter
	filter := &docker.DockerAuditLogFilter{
		Page:         page,
		PageSize:     pageSize,
		InstanceID:   instanceID,
		User:         user,
		Action:       action,
		ResourceType: resourceType,
		StartTime:    startTime,
		EndTime:      endTime,
		Success:      success,
	}

	// Query audit logs
	logs, total, err := h.auditService.GetAuditLogs(c.Request.Context(), filter)
	if err != nil {
		logrus.WithError(err).Error("Failed to query Docker audit logs")
		basehandlers.RespondInternalError(c, err)
		return
	}

	basehandlers.RespondPaginated(c, logs, page, pageSize, total)
}
