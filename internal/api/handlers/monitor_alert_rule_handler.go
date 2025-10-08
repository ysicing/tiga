package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// MonitorAlertRuleHandler handles monitor alert rule operations
type MonitorAlertRuleHandler struct {
	alertRepo repository.MonitorAlertRepository
}

// NewMonitorAlertRuleHandler creates a new monitor alert rule handler
func NewMonitorAlertRuleHandler(alertRepo repository.MonitorAlertRepository) *MonitorAlertRuleHandler {
	return &MonitorAlertRuleHandler{alertRepo: alertRepo}
}

// CreateRule creates a new alert rule
// @Summary Create alert rule
// @Description Create a new monitor alert rule
// @Tags Monitor Alerts
// @Accept json
// @Produce json
// @Param rule body models.MonitorAlertRule true "Alert rule"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/alert-rules [post]
func (h *MonitorAlertRuleHandler) CreateRule(c *gin.Context) {
	var rule models.MonitorAlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid request"})
		return
	}

	if err := h.alertRepo.CreateRule(c.Request.Context(), &rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to create rule"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": rule})
}

// GetRule retrieves an alert rule by ID
// @Summary Get alert rule
// @Description Get an alert rule by ID
// @Tags Monitor Alerts
// @Produce json
// @Param id path string true "Rule ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/alert-rules/{id} [get]
func (h *MonitorAlertRuleHandler) GetRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid rule ID"})
		return
	}

	rule, err := h.alertRepo.GetRuleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 40404, "message": "Rule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": rule})
}

// ListRules lists alert rules
// @Summary List alert rules
// @Description List all alert rules with filtering
// @Tags Monitor Alerts
// @Produce json
// @Param type query string false "Rule type (host/service)"
// @Param enabled query bool false "Filter by enabled status"
// @Param severity query string false "Filter by severity"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/alert-rules [get]
func (h *MonitorAlertRuleHandler) ListRules(c *gin.Context) {
	filter := repository.MonitorAlertRuleFilter{
		Page:     getIntQuery(c, "page", 1),
		PageSize: getIntQuery(c, "page_size", 20),
		Type:     c.Query("type"),
		Severity: c.Query("severity"),
	}

	if enabled := c.Query("enabled"); enabled != "" {
		val := enabled == "true"
		filter.Enabled = &val
	}

	rules, total, err := h.alertRepo.ListRules(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to list rules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": rules,
			"total": total,
			"page":  filter.Page,
			"page_size": filter.PageSize,
		},
	})
}

// UpdateRule updates an alert rule
// @Summary Update alert rule
// @Description Update an existing alert rule
// @Tags Monitor Alerts
// @Accept json
// @Produce json
// @Param id path string true "Rule ID"
// @Param rule body models.MonitorAlertRule true "Alert rule"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/alert-rules/{id} [put]
func (h *MonitorAlertRuleHandler) UpdateRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid rule ID"})
		return
	}

	rule, err := h.alertRepo.GetRuleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 40404, "message": "Rule not found"})
		return
	}

	if err := c.ShouldBindJSON(rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid request"})
		return
	}

	if err := h.alertRepo.UpdateRule(c.Request.Context(), rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to update rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": rule})
}

// DeleteRule deletes an alert rule
// @Summary Delete alert rule
// @Description Delete an alert rule and its events
// @Tags Monitor Alerts
// @Produce json
// @Param id path string true "Rule ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/alert-rules/{id} [delete]
func (h *MonitorAlertRuleHandler) DeleteRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid rule ID"})
		return
	}

	if err := h.alertRepo.DeleteRule(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to delete rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// ListEvents lists alert events
// @Summary List alert events
// @Description List all alert events with filtering
// @Tags Monitor Alerts
// @Produce json
// @Param rule_id query int false "Filter by rule ID"
// @Param status query string false "Filter by status (firing/acknowledged/resolved)"
// @Param severity query string false "Filter by severity"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/alert-events [get]
func (h *MonitorAlertRuleHandler) ListEvents(c *gin.Context) {
	filter := repository.MonitorAlertEventFilter{
		Page:     getIntQuery(c, "page", 1),
		PageSize: getIntQuery(c, "page_size", 20),
		RuleID:   uint(getIntQuery(c, "rule_id", 0)),
		Status:   c.Query("status"),
		Severity: c.Query("severity"),
	}

	events, total, err := h.alertRepo.ListEvents(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to list events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": events,
			"total": total,
			"page":  filter.Page,
			"page_size": filter.PageSize,
		},
	})
}

// AcknowledgeEvent acknowledges an alert event
// @Summary Acknowledge alert event
// @Description Mark an alert event as acknowledged
// @Tags Monitor Alerts
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Param body body map[string]string true "Acknowledgment note"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/alert-events/{id}/acknowledge [post]
func (h *MonitorAlertRuleHandler) AcknowledgeEvent(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid event ID"})
		return
	}

	var req struct {
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid request"})
		return
	}

	// TODO: Get user ID from auth context
	userID := uuid.New() // Placeholder

	if err := h.alertRepo.AcknowledgeEvent(c.Request.Context(), eventID, userID, req.Note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to acknowledge event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// ResolveEvent resolves an alert event
// @Summary Resolve alert event
// @Description Mark an alert event as resolved
// @Tags Monitor Alerts
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Param body body map[string]string true "Resolution note"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/alert-events/{id}/resolve [post]
func (h *MonitorAlertRuleHandler) ResolveEvent(c *gin.Context) {
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid event ID"})
		return
	}

	var req struct {
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid request"})
		return
	}

	// TODO: Get user ID from auth context
	userID := uuid.New() // Placeholder

	if err := h.alertRepo.ResolveEvent(c.Request.Context(), eventID, userID, req.Note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to resolve event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// getIntQuery is a helper function to get int query parameters
func getIntQuery(c *gin.Context, key string, defaultValue int) int {
	if val := c.Query(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}
