package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/services/monitor"
)

// ServiceMonitorHandler handles service monitoring operations
type ServiceMonitorHandler struct {
	probeService *monitor.ServiceProbeService
}

// NewServiceMonitorHandler creates a new handler
func NewServiceMonitorHandler(probeService *monitor.ServiceProbeService) *ServiceMonitorHandler {
	return &ServiceMonitorHandler{probeService: probeService}
}

// CreateMonitor creates a new service monitor
func (h *ServiceMonitorHandler) CreateMonitor(c *gin.Context) {
	var req struct {
		Name             string `json:"name" binding:"required"`
		Type             string `json:"type" binding:"required"`
		Target           string `json:"target" binding:"required"`
		Interval         int    `json:"interval"`
		Timeout          int    `json:"timeout"`
		ProbeStrategy    string `json:"probe_strategy"`     // server/include/exclude/group
		ProbeNodeIDs     string `json:"probe_node_ids"`     // JSON array of UUIDs
		ProbeGroupName   string `json:"probe_group_name"`   // Node group name
		Enabled          bool   `json:"enabled"`
		NotifyOnFailure  bool   `json:"notify_on_failure"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid request"})
		return
	}

	mon := &models.ServiceMonitor{
		Name:            req.Name,
		Type:            models.ProbeType(req.Type),
		Target:          req.Target,
		Interval:        req.Interval,
		Timeout:         req.Timeout,
		ProbeStrategy:   models.ProbeStrategy(req.ProbeStrategy),
		ProbeNodeIDs:    req.ProbeNodeIDs,
		ProbeGroupName:  req.ProbeGroupName,
		Enabled:         req.Enabled,
		NotifyOnFailure: req.NotifyOnFailure,
	}

	// Default to server strategy if not specified
	if mon.ProbeStrategy == "" {
		mon.ProbeStrategy = models.ProbeStrategyServer
	}

	if err := h.probeService.CreateMonitor(c.Request.Context(), mon); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to create monitor"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": mon})
}

// GetMonitor gets a service monitor
func (h *ServiceMonitorHandler) GetMonitor(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid monitor ID"})
		return
	}

	mon, err := h.probeService.GetMonitor(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 40404, "message": "Monitor not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": mon})
}

// UpdateMonitor updates a service monitor
func (h *ServiceMonitorHandler) UpdateMonitor(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid monitor ID"})
		return
	}

	var req struct {
		Name             *string `json:"name"`
		Type             *string `json:"type"`
		Target           *string `json:"target"`
		Interval         *int    `json:"interval"`
		Timeout          *int    `json:"timeout"`
		ProbeStrategy    *string `json:"probe_strategy"`
		ProbeNodeIDs     *string `json:"probe_node_ids"`
		ProbeGroupName   *string `json:"probe_group_name"`
		Enabled          *bool   `json:"enabled"`
		NotifyOnFailure  *bool   `json:"notify_on_failure"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid request"})
		return
	}

	mon, err := h.probeService.GetMonitor(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 40404, "message": "Monitor not found"})
		return
	}

	// Update only provided fields
	if req.Name != nil {
		mon.Name = *req.Name
	}
	if req.Type != nil {
		mon.Type = models.ProbeType(*req.Type)
	}
	if req.Target != nil {
		mon.Target = *req.Target
	}
	if req.Interval != nil {
		mon.Interval = *req.Interval
	}
	if req.Timeout != nil {
		mon.Timeout = *req.Timeout
	}
	if req.ProbeStrategy != nil {
		mon.ProbeStrategy = models.ProbeStrategy(*req.ProbeStrategy)
	}
	if req.ProbeNodeIDs != nil {
		mon.ProbeNodeIDs = *req.ProbeNodeIDs
	}
	if req.ProbeGroupName != nil {
		mon.ProbeGroupName = *req.ProbeGroupName
	}
	if req.Enabled != nil {
		mon.Enabled = *req.Enabled
	}
	if req.NotifyOnFailure != nil {
		mon.NotifyOnFailure = *req.NotifyOnFailure
	}

	if err := h.probeService.UpdateMonitor(c.Request.Context(), mon); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to update"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": mon})
}

// DeleteMonitor deletes a service monitor
func (h *ServiceMonitorHandler) DeleteMonitor(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid monitor ID"})
		return
	}

	if err := h.probeService.DeleteMonitor(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to delete"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// TriggerProbe triggers manual probe
func (h *ServiceMonitorHandler) TriggerProbe(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid monitor ID"})
		return
	}

	result, err := h.probeService.TriggerManualProbe(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to trigger probe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// GetAvailability gets availability statistics
func (h *ServiceMonitorHandler) GetAvailability(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid monitor ID"})
		return
	}

	period := c.DefaultQuery("period", "24h")

	stats, err := h.probeService.GetAvailabilityStats(c.Request.Context(), id, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": stats})
}

// ListMonitors lists service monitors
func (h *ServiceMonitorHandler) ListMonitors(c *gin.Context) {
	// TODO: Add filter query parameters
	monitors, total, err := h.probeService.ListMonitors(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to list monitors"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": monitors,
			"total": total,
		},
	})
}

// GetHostProbeHistory gets probe history for a specific host (multi-line chart data)
// This endpoint returns probe histories grouped by service monitor, showing all probes
// executed by a specific host node to various targets
func (h *ServiceMonitorHandler) GetHostProbeHistory(c *gin.Context) {
	hostID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid host ID"})
		return
	}

	// Get time range (default to last 24 hours)
	hoursStr := c.DefaultQuery("hours", "24")
	hours, err := strconv.Atoi(hoursStr)
	if err != nil || hours <= 0 || hours > 720 {
		hours = 24
	}

	start := time.Now().Add(-time.Duration(hours) * time.Hour)
	end := time.Now()

	histories, err := h.probeService.GetHostProbeHistory(c.Request.Context(), hostID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to get probe history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": histories})
}

// GetOverview gets 30-day aggregated statistics for all service monitors
// This endpoint is used for the service overview page with availability heatmap
func (h *ServiceMonitorHandler) GetOverview(c *gin.Context) {
	overview, err := h.probeService.GetOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to get overview"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": overview})
}
