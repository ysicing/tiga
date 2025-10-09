package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/host"
)

// HostHandler handles host management HTTP requests
type HostHandler struct {
	hostService *host.HostService
}

// NewHostHandler creates a new host handler
func NewHostHandler(hostService *host.HostService) *HostHandler {
	return &HostHandler{
		hostService: hostService,
	}
}

// CreateHost godoc
// @Summary Create a new host node
// @Tags hosts
// @Accept json
// @Produce json
// @Param host body models.HostNode true "Host information"
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/hosts [post]
func (h *HostHandler) CreateHost(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		Note         string `json:"note"`
		PublicNote   string `json:"public_note"`
		DisplayIndex int    `json:"display_index"`
		HideForGuest bool   `json:"hide_for_guest"`

		// Billing information
		Cost         float64 `json:"cost"`
		RenewalType  string  `json:"renewal_type"`
		PurchaseDate *string `json:"purchase_date"`
		ExpiryDate   *string `json:"expiry_date"`
		AutoRenew    bool    `json:"auto_renew"`
		TrafficLimit int64   `json:"traffic_limit"`

		// Group
		GroupName string `json:"group_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid request parameters",
			"details": err.Error(),
		})
		return
	}

	// Parse dates if provided
	var expiryDate *time.Time
	if req.ExpiryDate != nil && *req.ExpiryDate != "" {
		t, err := time.Parse("2006-01-02", *req.ExpiryDate)
		if err == nil {
			expiryDate = &t
		}
	}

	var purchaseDate *time.Time
	if req.PurchaseDate != nil && *req.PurchaseDate != "" {
		t, err := time.Parse("2006-01-02", *req.PurchaseDate)
		if err == nil {
			purchaseDate = &t
		}
	}

	hostNode := &models.HostNode{
		Name:         req.Name,
		Note:         req.Note,
		PublicNote:   req.PublicNote,
		DisplayIndex: req.DisplayIndex,
		HideForGuest: req.HideForGuest,
		Cost:         req.Cost,
		RenewalType:  req.RenewalType,
		PurchaseDate: purchaseDate,
		ExpiryDate:   expiryDate,
		AutoRenew:    req.AutoRenew,
		TrafficLimit: req.TrafficLimit,
		GroupName:    req.GroupName,
	}

	if err := h.hostService.CreateHost(c.Request.Context(), hostNode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to create host",
			"details": err.Error(),
		})
		return
	}

	// Generate agent install command
	installCmd, _ := h.hostService.GetAgentInstallCommand(c.Request.Context(), hostNode.ID)

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id":                hostNode.ID,
			"uuid":              hostNode.ID.String(),
			"name":              hostNode.Name,
			"agent_install_cmd": installCmd,
			"note":              hostNode.Note,
			"public_note":       hostNode.PublicNote,
			"display_index":     hostNode.DisplayIndex,
			"hide_for_guest":    hostNode.HideForGuest,
			"cost":              hostNode.Cost,
			"renewal_type":      hostNode.RenewalType,
			"purchase_date":     hostNode.PurchaseDate,
			"expiry_date":       hostNode.ExpiryDate,
			"auto_renew":        hostNode.AutoRenew,
			"traffic_limit":     hostNode.TrafficLimit,
			"traffic_used":      hostNode.TrafficUsed,
			"group_name":        hostNode.GroupName,

			"created_at": hostNode.CreatedAt,
			"updated_at": hostNode.UpdatedAt,
		},
	})
}

// ListHosts godoc
// @Summary List hosts
// @Tags hosts
// @Produce json
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Param group_name query string false "Group name filter"
// @Param search query string false "Search keyword"
// @Param sort query string false "Sort field"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/hosts [get]
func (h *HostHandler) ListHosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	groupName := c.Query("group_name")
	search := c.Query("search")
	sort := c.Query("sort")

	filter := repository.HostFilter{
		Page:      page,
		PageSize:  pageSize,
		GroupName: groupName,
		Search:    search,
		Sort:      sort,
	}

	hosts, total, err := h.hostService.ListHosts(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to list hosts",
		})
		return
	}

	// Convert to response format
	items := make([]map[string]interface{}, len(hosts))
	for i, host := range hosts {
		item := map[string]interface{}{
			"id":             host.ID,
			"uuid":           host.ID.String(),
			"name":           host.Name,
			"note":           host.Note,
			"public_note":    host.PublicNote,
			"display_index":  host.DisplayIndex,
			"hide_for_guest": host.HideForGuest,
			"cost":           host.Cost,
			"renewal_type":   host.RenewalType,
			"purchase_date":  host.PurchaseDate,
			"expiry_date":    host.ExpiryDate,
			"auto_renew":     host.AutoRenew,
			"traffic_limit":  host.TrafficLimit,
			"traffic_used":   host.TrafficUsed,
			"group_name":     host.GroupName,

			"online":     host.Online,
			"created_at": host.CreatedAt,
		}

		if host.LastActive != nil {
			item["last_active"] = host.LastActive
		}

		if host.HostInfo != nil {
			item["host_info"] = gin.H{
				"platform":         host.HostInfo.Platform,
				"platform_version": host.HostInfo.PlatformVersion,
				"arch":             host.HostInfo.Arch,
				"cpu_model":        host.HostInfo.CPUModel,
				"cpu_cores":        host.HostInfo.CPUCores,
				"mem_total":        host.HostInfo.MemTotal,
				"disk_total":       host.HostInfo.DiskTotal,
			}
		}

		// Get current state
		if state, err := h.hostService.GetHostState(c.Request.Context(), host.ID); err == nil {
			item["current_state"] = gin.H{
				"cpu_usage":     state.CPUUsage,
				"mem_usage":     state.MemUsage,
				"disk_usage":    state.DiskUsage,
				"net_in_speed":  state.NetInSpeed,
				"net_out_speed": state.NetOutSpeed,
			}
		}

		items[i] = item
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items":     items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetHost godoc
// @Summary Get host details
// @Tags hosts
// @Produce json
// @Param id path int true "Host ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/hosts/{id} [get]
func (h *HostHandler) GetHost(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid host ID",
		})
		return
	}

	host, err := h.hostService.GetHost(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40404,
			"message": "Host not found",
		})
		return
	}

	data := gin.H{
		"id":             host.ID,
		"uuid":           host.ID.String(),
		"name":           host.Name,
		"note":           host.Note,
		"public_note":    host.PublicNote,
		"display_index":  host.DisplayIndex,
		"hide_for_guest": host.HideForGuest,
		"cost":           host.Cost,
		"renewal_type":   host.RenewalType,
		"purchase_date":  host.PurchaseDate,
		"expiry_date":    host.ExpiryDate,
		"auto_renew":     host.AutoRenew,
		"traffic_limit":  host.TrafficLimit,
		"traffic_used":   host.TrafficUsed,
		"group_name":     host.GroupName,

		"online":     host.Online,
		"created_at": host.CreatedAt,
		"updated_at": host.UpdatedAt,
	}

	if host.LastActive != nil {
		data["last_active"] = host.LastActive
	}

	if host.HostInfo != nil {
		data["host_info"] = host.HostInfo
	}

	// Get agent connection info
	// TODO: Get from AgentConnection table

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    data,
	})
}

// UpdateHost godoc
// @Summary Update host
// @Tags hosts
// @Accept json
// @Produce json
// @Param id path int true "Host ID"
// @Param host body models.HostNode true "Host information"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/hosts/{id} [put]
func (h *HostHandler) UpdateHost(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid host ID",
		})
		return
	}

	var req struct {
		Name         string `json:"name"`
		Note         string `json:"note"`
		PublicNote   string `json:"public_note"`
		DisplayIndex int    `json:"display_index"`
		HideForGuest bool   `json:"hide_for_guest"`

		// Billing information
		Cost         float64 `json:"cost"`
		RenewalType  string  `json:"renewal_type"`
		PurchaseDate *string `json:"purchase_date"`
		ExpiryDate   *string `json:"expiry_date"`
		AutoRenew    bool    `json:"auto_renew"`
		TrafficLimit int64   `json:"traffic_limit"`

		// Group
		GroupName string `json:"group_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid request",
		})
		return
	}

	host, err := h.hostService.GetHost(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40404,
			"message": "Host not found",
		})
		return
	}

	// Parse dates if provided
	if req.PurchaseDate != nil && *req.PurchaseDate != "" {
		t, err := time.Parse("2006-01-02", *req.PurchaseDate)
		if err == nil {
			host.PurchaseDate = &t
		}
	}

	if req.ExpiryDate != nil && *req.ExpiryDate != "" {
		t, err := time.Parse("2006-01-02", *req.ExpiryDate)
		if err == nil {
			host.ExpiryDate = &t
		}
	}

	// Update fields
	host.Name = req.Name
	host.Note = req.Note
	host.PublicNote = req.PublicNote
	host.DisplayIndex = req.DisplayIndex
	host.HideForGuest = req.HideForGuest
	host.Cost = req.Cost
	host.RenewalType = req.RenewalType
	host.AutoRenew = req.AutoRenew
	host.TrafficLimit = req.TrafficLimit
	host.GroupName = req.GroupName

	if err := h.hostService.UpdateHost(c.Request.Context(), host); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to update host",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    host,
	})
}

// DeleteHost godoc
// @Summary Delete host
// @Tags hosts
// @Produce json
// @Param id path int true "Host ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/hosts/{id} [delete]
func (h *HostHandler) DeleteHost(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid host ID",
		})
		return
	}

	if err := h.hostService.DeleteHost(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to delete host",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "主机已删除",
	})
}

// GetCurrentState godoc
// @Summary Get current state
// @Tags hosts
// @Produce json
// @Param id path int true "Host ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/hosts/{id}/state/current [get]
func (h *HostHandler) GetCurrentState(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid host ID",
		})
		return
	}

	state, err := h.hostService.GetHostState(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40404,
			"message": "State not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    state,
	})
}

// GetHistoryState godoc
// @Summary Get historical states
// @Tags hosts
// @Produce json
// @Param id path int true "Host ID"
// @Param start query string true "Start time (RFC3339)"
// @Param end query string true "End time (RFC3339)"
// @Param interval query string false "Interval (auto/1m/5m/1h/1d)"
// @Param metrics query string false "Metrics (comma-separated)"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/hosts/{id}/state/history [get]
func (h *HostHandler) GetHistoryState(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid host ID",
		})
		return
	}

	start := c.Query("start")
	end := c.Query("end")
	interval := c.DefaultQuery("interval", "auto")

	// Parse times
	startTime, _ := time.Parse(time.RFC3339, start)
	endTime, _ := time.Parse(time.RFC3339, end)

	// Get historical states
	states, err := h.hostService.GetHostStateHistory(c.Request.Context(), id, start, end, interval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to get history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"start":    startTime,
			"end":      endTime,
			"interval": interval,
			"points":   states,
		},
	})
}
