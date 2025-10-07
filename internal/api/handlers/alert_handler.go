package handlers

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/api/middleware"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// AlertHandler handles alert endpoints
type AlertHandler struct {
	alertRepo *repository.AlertRepository
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler(alertRepo *repository.AlertRepository) *AlertHandler {
	return &AlertHandler{
		alertRepo: alertRepo,
	}
}

// --- Alert Rules ---

// ListAlertRulesRequest represents a request to list alert rules
type ListAlertRulesRequest struct {
	InstanceID *string `form:"instance_id" binding:"omitempty,uuid"`
	Enabled    *bool   `form:"enabled"`
	Severity   string  `form:"severity"`
	Search     string  `form:"search"`
	Page       int     `form:"page" binding:"min=1"`
	PageSize   int     `form:"page_size" binding:"min=1,max=100"`
}

// ListAlertRules lists all alert rules with pagination
// @Summary List alert rules
// @Description Get a paginated list of alert rules
// @Tags alerts
// @Produce json
// @Security BearerAuth
// @Param instance_id query string false "Filter by instance ID (UUID)"
// @Param enabled query boolean false "Filter by enabled status"
// @Param severity query string false "Filter by severity (critical, warning, info)"
// @Param search query string false "Search in name, description"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} PaginatedResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/alerts/rules [get]
func (h *AlertHandler) ListAlertRules(c *gin.Context) {
	var req ListAlertRulesRequest
	if !BindQuery(c, &req) {
		return
	}

	// Set defaults
	req.Page = defaultInt(req.Page, 1)
	req.PageSize = defaultInt(req.PageSize, 20)
	req.PageSize = clamp(req.PageSize, 1, 100)

	// Build filter
	filter := &repository.ListRulesFilter{
		Enabled:  req.Enabled,
		Severity: req.Severity,
		Search:   req.Search,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	if req.InstanceID != nil {
		instanceID, err := ParseUUID(*req.InstanceID)
		if err != nil {
			RespondBadRequest(c, err)
			return
		}
		filter.InstanceID = &instanceID
	}

	// Get alert rules
	rules, total, err := h.alertRepo.ListRules(c.Request.Context(), filter)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondPaginated(c, rules, req.Page, req.PageSize, total)
}

// GetAlertRuleRequest represents a request to get an alert rule
type GetAlertRuleRequest struct {
	RuleID string `uri:"rule_id" binding:"required,uuid"`
}

// GetAlertRule gets an alert rule by ID
// @Summary Get alert rule
// @Description Get alert rule details by ID
// @Tags alerts
// @Produce json
// @Security BearerAuth
// @Param rule_id path string true "Rule ID (UUID)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/alerts/rules/{rule_id} [get]
func (h *AlertHandler) GetAlertRule(c *gin.Context) {
	var req GetAlertRuleRequest
	if !BindURI(c, &req) {
		return
	}

	ruleID, err := ParseUUID(req.RuleID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	rule, err := h.alertRepo.GetRuleByID(c.Request.Context(), ruleID)
	if err != nil {
		RespondNotFound(c, err)
		return
	}

	RespondSuccess(c, rule)
}

// CreateAlertRuleRequest represents a request to create an alert rule
type CreateAlertRuleRequest struct {
	Name                 string                 `json:"name" binding:"required"`
	Description          string                 `json:"description"`
	InstanceID           *string                `json:"instance_id,omitempty" binding:"omitempty,uuid"`
	RuleType             string                 `json:"rule_type" binding:"required,oneof=threshold anomaly rate"`
	RuleConfig           map[string]interface{} `json:"rule_config" binding:"required"`
	Severity             string                 `json:"severity" binding:"required,oneof=critical warning info"`
	NotificationChannels []string               `json:"notification_channels"`
	NotificationConfig   map[string]interface{} `json:"notification_config"`
	Enabled              bool                   `json:"enabled"`
}

// CreateAlertRule creates a new alert rule
// @Summary Create alert rule
// @Description Create a new alert rule
// @Tags alerts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateAlertRuleRequest true "Alert rule creation request"
// @Success 201 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/alerts/rules [post]
func (h *AlertHandler) CreateAlertRule(c *gin.Context) {
	var req CreateAlertRuleRequest
	if !BindJSON(c, &req) {
		return
	}

	// Parse instance ID if provided
	var instanceIDPtr *uuid.UUID
	if req.InstanceID != nil {
		instanceID, err := ParseUUID(*req.InstanceID)
		if err != nil {
			RespondBadRequest(c, err)
			return
		}
		instanceIDPtr = &instanceID
	}

	// Create alert rule
	rule := &models.Alert{
		Name:                 req.Name,
		Description:          req.Description,
		InstanceID:           instanceIDPtr,
		RuleType:             req.RuleType,
		RuleConfig:           req.RuleConfig,
		Severity:             req.Severity,
		NotificationChannels: req.NotificationChannels,
		NotificationConfig:   req.NotificationConfig,
		Enabled:              req.Enabled,
	}

	if err := h.alertRepo.CreateRule(c.Request.Context(), rule); err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondCreated(c, rule)
}

// UpdateAlertRuleRequest represents a request to update an alert rule
type UpdateAlertRuleRequest struct {
	RuleID               string                 `uri:"rule_id" binding:"required,uuid"`
	Name                 *string                `json:"name,omitempty"`
	Description          *string                `json:"description,omitempty"`
	RuleConfig           map[string]interface{} `json:"rule_config,omitempty"`
	Severity             *string                `json:"severity,omitempty" binding:"omitempty,oneof=critical warning info"`
	NotificationChannels []string               `json:"notification_channels,omitempty"`
	NotificationConfig   map[string]interface{} `json:"notification_config,omitempty"`
	Enabled              *bool                  `json:"enabled,omitempty"`
}

// UpdateAlertRule updates an alert rule
// @Summary Update alert rule
// @Description Update alert rule details
// @Tags alerts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param rule_id path string true "Rule ID (UUID)"
// @Param request body UpdateAlertRuleRequest true "Alert rule update request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/alerts/rules/{rule_id} [patch]
func (h *AlertHandler) UpdateAlertRule(c *gin.Context) {
	var req UpdateAlertRuleRequest
	if !BindURI(c, &req) {
		return
	}
	if !BindJSON(c, &req) {
		return
	}

	ruleID, err := ParseUUID(req.RuleID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	// Get existing rule
	rule, err := h.alertRepo.GetRuleByID(c.Request.Context(), ruleID)
	if err != nil {
		RespondNotFound(c, err)
		return
	}

	// Apply updates
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.RuleConfig != nil {
		rule.RuleConfig = req.RuleConfig
	}
	if req.Severity != nil {
		rule.Severity = *req.Severity
	}
	if req.NotificationChannels != nil {
		rule.NotificationChannels = req.NotificationChannels
	}
	if req.NotificationConfig != nil {
		rule.NotificationConfig = req.NotificationConfig
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	// Update rule
	if err := h.alertRepo.UpdateRule(c.Request.Context(), rule); err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, rule)
}

// DeleteAlertRule deletes an alert rule
// @Summary Delete alert rule
// @Description Delete an alert rule
// @Tags alerts
// @Produce json
// @Security BearerAuth
// @Param rule_id path string true "Rule ID (UUID)"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/alerts/rules/{rule_id} [delete]
func (h *AlertHandler) DeleteAlertRule(c *gin.Context) {
	var req GetAlertRuleRequest
	if !BindURI(c, &req) {
		return
	}

	ruleID, err := ParseUUID(req.RuleID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	if err := h.alertRepo.DeleteRule(c.Request.Context(), ruleID); err != nil {
		RespondNotFound(c, err)
		return
	}

	RespondNoContent(c)
}

// ToggleAlertRuleRequest represents a request to toggle alert rule
type ToggleAlertRuleRequest struct {
	RuleID  string `uri:"rule_id" binding:"required,uuid"`
	Enabled bool   `json:"enabled" binding:"required"`
}

// ToggleAlertRule enables or disables an alert rule
// @Summary Toggle alert rule
// @Description Enable or disable an alert rule
// @Tags alerts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param rule_id path string true "Rule ID (UUID)"
// @Param request body ToggleAlertRuleRequest true "Toggle request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/alerts/rules/{rule_id}/toggle [patch]
func (h *AlertHandler) ToggleAlertRule(c *gin.Context) {
	var req ToggleAlertRuleRequest
	if !BindURI(c, &req) {
		return
	}
	if !BindJSON(c, &req) {
		return
	}

	ruleID, err := ParseUUID(req.RuleID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	if err := h.alertRepo.ToggleRule(c.Request.Context(), ruleID, req.Enabled); err != nil {
		RespondNotFound(c, err)
		return
	}

	status := "disabled"
	if req.Enabled {
		status = "enabled"
	}

	RespondSuccessWithMessage(c, nil, fmt.Sprintf("alert rule %s", status))
}

// --- Alert Events ---

// ListAlertEventsRequest represents a request to list alert events
type ListAlertEventsRequest struct {
	AlertID    *string `form:"alert_id" binding:"omitempty,uuid"`
	InstanceID *string `form:"instance_id" binding:"omitempty,uuid"`
	Status     string  `form:"status"`
	Severity   string  `form:"severity"`
	StartTime  string  `form:"start_time"` // RFC3339
	EndTime    string  `form:"end_time"`   // RFC3339
	Page       int     `form:"page" binding:"min=1"`
	PageSize   int     `form:"page_size" binding:"min=1,max=100"`
}

// ListAlertEvents lists all alert events with pagination
// @Summary List alert events
// @Description Get a paginated list of alert events
// @Tags alerts
// @Produce json
// @Security BearerAuth
// @Param alert_id query string false "Filter by alert ID (UUID)"
// @Param instance_id query string false "Filter by instance ID (UUID)"
// @Param status query string false "Filter by status (firing, acknowledged, resolved)"
// @Param severity query string false "Filter by severity"
// @Param start_time query string false "Start time (RFC3339)"
// @Param end_time query string false "End time (RFC3339)"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} PaginatedResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/alerts/events [get]
func (h *AlertHandler) ListAlertEvents(c *gin.Context) {
	var req ListAlertEventsRequest
	if !BindQuery(c, &req) {
		return
	}

	// Set defaults
	req.Page = defaultInt(req.Page, 1)
	req.PageSize = defaultInt(req.PageSize, 20)
	req.PageSize = clamp(req.PageSize, 1, 100)

	// Build filter
	filter := &repository.ListEventsFilter{
		Status:   req.Status,
		Severity: req.Severity,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	if req.AlertID != nil {
		alertID, err := ParseUUID(*req.AlertID)
		if err != nil {
			RespondBadRequest(c, err)
			return
		}
		filter.AlertID = &alertID
	}

	if req.InstanceID != nil {
		instanceID, err := ParseUUID(*req.InstanceID)
		if err != nil {
			RespondBadRequest(c, err)
			return
		}
		filter.InstanceID = &instanceID
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

	// Get alert events
	events, total, err := h.alertRepo.ListEvents(c.Request.Context(), filter)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondPaginated(c, events, req.Page, req.PageSize, total)
}

// GetAlertEventRequest represents a request to get an alert event
type GetAlertEventRequest struct {
	EventID string `uri:"event_id" binding:"required,uuid"`
}

// GetAlertEvent gets an alert event by ID
// @Summary Get alert event
// @Description Get alert event details by ID
// @Tags alerts
// @Produce json
// @Security BearerAuth
// @Param event_id path string true "Event ID (UUID)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/alerts/events/{event_id} [get]
func (h *AlertHandler) GetAlertEvent(c *gin.Context) {
	var req GetAlertEventRequest
	if !BindURI(c, &req) {
		return
	}

	eventID, err := ParseUUID(req.EventID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	event, err := h.alertRepo.GetEventByID(c.Request.Context(), eventID)
	if err != nil {
		RespondNotFound(c, err)
		return
	}

	RespondSuccess(c, event)
}

// AcknowledgeAlertEventRequest represents a request to acknowledge an alert event
type AcknowledgeAlertEventRequest struct {
	EventID string `uri:"event_id" binding:"required,uuid"`
	Note    string `json:"note"`
}

// AcknowledgeAlertEvent acknowledges an alert event
// @Summary Acknowledge alert event
// @Description Acknowledge an alert event
// @Tags alerts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param event_id path string true "Event ID (UUID)"
// @Param request body AcknowledgeAlertEventRequest true "Acknowledgement request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/alerts/events/{event_id}/acknowledge [post]
func (h *AlertHandler) AcknowledgeAlertEvent(c *gin.Context) {
	var req AcknowledgeAlertEventRequest
	if !BindURI(c, &req) {
		return
	}
	if !BindJSON(c, &req) {
		return
	}

	eventID, err := ParseUUID(req.EventID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	// Get acknowledging user ID
	userID, err := middleware.GetUserID(c)
	if err != nil {
		RespondUnauthorized(c, err)
		return
	}

	// Acknowledge event
	if err := h.alertRepo.AcknowledgeEvent(c.Request.Context(), eventID, userID, req.Note); err != nil {
		RespondBadRequest(c, err)
		return
	}

	RespondSuccessWithMessage(c, nil, "alert event acknowledged")
}

// ResolveAlertEvent resolves an alert event
// @Summary Resolve alert event
// @Description Resolve an alert event
// @Tags alerts
// @Produce json
// @Security BearerAuth
// @Param event_id path string true "Event ID (UUID)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/alerts/events/{event_id}/resolve [post]
func (h *AlertHandler) ResolveAlertEvent(c *gin.Context) {
	var req GetAlertEventRequest
	if !BindURI(c, &req) {
		return
	}

	eventID, err := ParseUUID(req.EventID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	if err := h.alertRepo.ResolveEvent(c.Request.Context(), eventID); err != nil {
		RespondBadRequest(c, err)
		return
	}

	RespondSuccessWithMessage(c, nil, "alert event resolved")
}

// GetActiveAlertEvents gets all active alert events
// @Summary Get active alert events
// @Description Get all currently firing alert events
// @Tags alerts
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/alerts/events/active [get]
func (h *AlertHandler) GetActiveAlertEvents(c *gin.Context) {
	events, err := h.alertRepo.ListActiveEvents(c.Request.Context())
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, events)
}

// GetAlertStatistics gets alert statistics
// @Summary Get alert statistics
// @Description Get overall alert statistics
// @Tags alerts
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/alerts/statistics [get]
func (h *AlertHandler) GetAlertStatistics(c *gin.Context) {
	stats, err := h.alertRepo.GetStatistics(c.Request.Context())
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, stats)
}
