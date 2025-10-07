package handlers

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/repository"
)

// AuditLogHandler handles audit log endpoints
type AuditLogHandler struct {
	auditRepo *repository.AuditLogRepository
}

// NewAuditLogHandler creates a new audit log handler
func NewAuditLogHandler(auditRepo *repository.AuditLogRepository) *AuditLogHandler {
	return &AuditLogHandler{
		auditRepo: auditRepo,
	}
}

// ListAuditLogsRequest represents a request to list audit logs
type ListAuditLogsRequest struct {
	UserID       *string `form:"user_id" binding:"omitempty,uuid"`
	ResourceType string  `form:"resource_type"`
	ResourceID   *string `form:"resource_id" binding:"omitempty,uuid"`
	Action       string  `form:"action"`
	Status       string  `form:"status" binding:"omitempty,oneof=success failure"`
	StartTime    string  `form:"start_time"` // RFC3339
	EndTime      string  `form:"end_time"`   // RFC3339
	IPAddress    string  `form:"ip_address"`
	Search       string  `form:"search"`
	Page         int     `form:"page" binding:"min=1"`
	PageSize     int     `form:"page_size" binding:"min=1,max=100"`
}

// ListAuditLogs lists all audit logs with pagination
// @Summary List audit logs
// @Description Get a paginated list of audit logs
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Param user_id query string false "Filter by user ID (UUID)"
// @Param resource_type query string false "Filter by resource type"
// @Param resource_id query string false "Filter by resource ID (UUID)"
// @Param action query string false "Filter by action (create, read, update, delete)"
// @Param status query string false "Filter by status (success, failure)"
// @Param start_time query string false "Start time (RFC3339)"
// @Param end_time query string false "End time (RFC3339)"
// @Param ip_address query string false "Filter by IP address"
// @Param search query string false "Search in description, changes"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} PaginatedResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/audit [get]
func (h *AuditLogHandler) ListAuditLogs(c *gin.Context) {
	var req ListAuditLogsRequest
	if !BindQuery(c, &req) {
		return
	}

	// Set defaults
	req.Page = defaultInt(req.Page, 1)
	req.PageSize = defaultInt(req.PageSize, 20)
	req.PageSize = clamp(req.PageSize, 1, 100)

	// Build filter
	filter := &repository.ListAuditLogsFilter{
		ResourceType: req.ResourceType,
		Action:       req.Action,
		Status:       req.Status,
		IPAddress:    req.IPAddress,
		Search:       req.Search,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}

	// Parse user ID if provided
	if req.UserID != nil {
		userID, err := ParseUUID(*req.UserID)
		if err != nil {
			RespondBadRequest(c, err)
			return
		}
		filter.UserID = &userID
	}

	// Parse resource ID if provided
	if req.ResourceID != nil {
		resourceID, err := ParseUUID(*req.ResourceID)
		if err != nil {
			RespondBadRequest(c, err)
			return
		}
		filter.ResourceID = &resourceID
	}

	// Parse time range
	if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			RespondBadRequest(c, err)
			return
		}
		filter.StartTime = startTime
	}

	if req.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			RespondBadRequest(c, err)
			return
		}
		filter.EndTime = endTime
	}

	// Get audit logs
	logs, total, err := h.auditRepo.ListAuditLogs(c.Request.Context(), filter)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondPaginated(c, logs, req.Page, req.PageSize, total)
}

// GetAuditLogRequest represents a request to get an audit log
type GetAuditLogRequest struct {
	LogID string `uri:"log_id" binding:"required,uuid"`
}

// GetAuditLog gets an audit log by ID
// @Summary Get audit log
// @Description Get audit log details by ID
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Param log_id path string true "Log ID (UUID)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/audit/{log_id} [get]
func (h *AuditLogHandler) GetAuditLog(c *gin.Context) {
	var req GetAuditLogRequest
	if !BindURI(c, &req) {
		return
	}

	logID, err := ParseUUID(req.LogID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	log, err := h.auditRepo.GetByID(c.Request.Context(), logID)
	if err != nil {
		RespondNotFound(c, err)
		return
	}

	RespondSuccess(c, log)
}

// ListUserAuditLogsRequest represents a request to list user audit logs
type ListUserAuditLogsRequest struct {
	UserID string `uri:"user_id" binding:"required,uuid"`
	Limit  int    `form:"limit" binding:"min=1,max=1000"`
}

// ListUserAuditLogs lists audit logs for a specific user
// @Summary List user audit logs
// @Description Get audit logs for a specific user
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Param limit query int false "Limit results" default(100)
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/users/{user_id}/audit [get]
func (h *AuditLogHandler) ListUserAuditLogs(c *gin.Context) {
	var req ListUserAuditLogsRequest
	if !BindURI(c, &req) {
		return
	}
	if !BindQuery(c, &req) {
		return
	}

	userID, err := ParseUUID(req.UserID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	limit := defaultInt(req.Limit, 100)

	logs, err := h.auditRepo.ListByUser(c.Request.Context(), userID, limit)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, logs)
}

// ListResourceAuditLogsRequest represents a request to list resource audit logs
type ListResourceAuditLogsRequest struct {
	ResourceType string `uri:"resource_type" binding:"required"`
	ResourceID   string `uri:"resource_id" binding:"required,uuid"`
}

// ListResourceAuditLogs lists audit logs for a specific resource
// @Summary List resource audit logs
// @Description Get audit logs for a specific resource
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Param resource_type path string true "Resource type"
// @Param resource_id path string true "Resource ID (UUID)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/audit/resources/{resource_type}/{resource_id} [get]
func (h *AuditLogHandler) ListResourceAuditLogs(c *gin.Context) {
	var req ListResourceAuditLogsRequest
	if !BindURI(c, &req) {
		return
	}

	resourceID, err := ParseUUID(req.ResourceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	logs, err := h.auditRepo.ListByResource(c.Request.Context(), req.ResourceType, resourceID)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, logs)
}

// ListRecentLogsRequest represents a request to list recent logs
type ListRecentLogsRequest struct {
	Limit int `form:"limit" binding:"min=1,max=1000"`
}

// ListRecentLogs lists recent audit logs
// @Summary List recent audit logs
// @Description Get the most recent audit logs
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit results" default(100)
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/audit/recent [get]
func (h *AuditLogHandler) ListRecentLogs(c *gin.Context) {
	var req ListRecentLogsRequest
	if !BindQuery(c, &req) {
		return
	}

	limit := defaultInt(req.Limit, 100)

	logs, err := h.auditRepo.ListRecentLogs(c.Request.Context(), limit)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, logs)
}

// ListFailedActionsRequest represents a request to list failed actions
type ListFailedActionsRequest struct {
	Limit int `form:"limit" binding:"min=1,max=1000"`
}

// ListFailedActions lists failed audit logs
// @Summary List failed actions
// @Description Get failed audit logs
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit results" default(100)
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/audit/failed [get]
func (h *AuditLogHandler) ListFailedActions(c *gin.Context) {
	var req ListFailedActionsRequest
	if !BindQuery(c, &req) {
		return
	}

	limit := defaultInt(req.Limit, 100)

	logs, err := h.auditRepo.ListFailedActions(c.Request.Context(), limit)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, logs)
}

// GetActivityTimelineRequest represents a request to get activity timeline
type GetActivityTimelineRequest struct {
	StartTime string `form:"start_time" binding:"required"` // RFC3339
	EndTime   string `form:"end_time" binding:"required"`   // RFC3339
	Interval  string `form:"interval" binding:"required"`   // e.g., "1 hour"
}

// GetActivityTimeline gets activity timeline aggregated by time bucket
// @Summary Get activity timeline
// @Description Get activity timeline aggregated over time buckets
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Param start_time query string true "Start time (RFC3339)"
// @Param end_time query string true "End time (RFC3339)"
// @Param interval query string true "Aggregation interval (e.g., '1 hour')"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/audit/timeline [get]
func (h *AuditLogHandler) GetActivityTimeline(c *gin.Context) {
	var req GetActivityTimelineRequest
	if !BindQuery(c, &req) {
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	timeline, err := h.auditRepo.GetActivityTimeline(c.Request.Context(), startTime, endTime, req.Interval)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, timeline)
}

// GetAuditStatistics gets audit log statistics
// @Summary Get audit statistics
// @Description Get overall audit log statistics
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/audit/statistics [get]
func (h *AuditLogHandler) GetAuditStatistics(c *gin.Context) {
	stats, err := h.auditRepo.GetStatistics(c.Request.Context())
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, stats)
}

// GetDistinctActionsRequest is empty as it has no parameters
type GetDistinctActionsRequest struct{}

// GetDistinctActions gets all distinct actions
// @Summary Get distinct actions
// @Description Get all distinct action types in audit logs
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/audit/actions [get]
func (h *AuditLogHandler) GetDistinctActions(c *gin.Context) {
	actions, err := h.auditRepo.GetDistinctActions(c.Request.Context())
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, actions)
}

// GetDistinctResourceTypes gets all distinct resource types
// @Summary Get distinct resource types
// @Description Get all distinct resource types in audit logs
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/audit/resource-types [get]
func (h *AuditLogHandler) GetDistinctResourceTypes(c *gin.Context) {
	resourceTypes, err := h.auditRepo.GetDistinctResourceTypes(c.Request.Context())
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, resourceTypes)
}

// SearchAuditLogsRequest represents a request to search audit logs
type SearchAuditLogsRequest struct {
	Query string `form:"query" binding:"required"`
	Limit int    `form:"limit" binding:"min=1,max=1000"`
}

// SearchAuditLogs performs full-text search on audit logs
// @Summary Search audit logs
// @Description Search audit logs by text query
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Param query query string true "Search query"
// @Param limit query int false "Limit results" default(100)
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/audit/search [get]
func (h *AuditLogHandler) SearchAuditLogs(c *gin.Context) {
	var req SearchAuditLogsRequest
	if !BindQuery(c, &req) {
		return
	}

	limit := defaultInt(req.Limit, 100)

	logs, err := h.auditRepo.SearchLogs(c.Request.Context(), req.Query, limit)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, logs)
}
