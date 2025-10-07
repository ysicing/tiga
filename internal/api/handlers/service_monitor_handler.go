package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
		HostID           uint   `json:"host_id"`
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
		HostNodeID:      req.HostID,
		Enabled:         req.Enabled,
		NotifyOnFailure: req.NotifyOnFailure,
	}

	if err := h.probeService.CreateMonitor(c.Request.Context(), mon); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to create monitor"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": mon})
}

// GetMonitor gets a service monitor
func (h *ServiceMonitorHandler) GetMonitor(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	mon, err := h.probeService.GetMonitor(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 40404, "message": "Monitor not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": mon})
}

// UpdateMonitor updates a service monitor
func (h *ServiceMonitorHandler) UpdateMonitor(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		Interval int  `json:"interval"`
		Enabled  bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid request"})
		return
	}

	mon, err := h.probeService.GetMonitor(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 40404, "message": "Monitor not found"})
		return
	}

	mon.Interval = req.Interval
	mon.Enabled = req.Enabled

	if err := h.probeService.UpdateMonitor(c.Request.Context(), mon); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to update"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": mon})
}

// DeleteMonitor deletes a service monitor
func (h *ServiceMonitorHandler) DeleteMonitor(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	if err := h.probeService.DeleteMonitor(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to delete"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// TriggerProbe triggers manual probe
func (h *ServiceMonitorHandler) TriggerProbe(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	result, err := h.probeService.TriggerManualProbe(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to trigger probe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// GetAvailability gets availability statistics
func (h *ServiceMonitorHandler) GetAvailability(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	period := c.DefaultQuery("period", "24h")

	stats, err := h.probeService.GetAvailabilityStats(c.Request.Context(), uint(id), period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": stats})
}
