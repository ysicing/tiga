package cluster

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/middleware"
	"github.com/ysicing/tiga/internal/repository"
)

// ListResourceHistory godoc
// @Summary List resource operation history
// @Description List resource operation history for a cluster with optional filters
// @Tags k8s-resource-history
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param resource_type query string false "Resource type (e.g., Pod, CloneSet)"
// @Param resource_name query string false "Resource name"
// @Param namespace query string false "Namespace"
// @Param api_group query string false "API group (for CRDs)"
// @Param api_version query string false "API version"
// @Param operation_type query string false "Operation type (create, update, delete, apply, scale, restart)"
// @Param success query bool false "Filter by success status"
// @Param start_time query string false "Start time (RFC3339 format)"
// @Param end_time query string false "End time (RFC3339 format)"
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20, max: 100)"
// @Success 200 {object} map[string]interface{} "code=200, data={items:[], total:int, page:int, page_size:int}"
// @Failure 400 {object} map[string]interface{} "code=400, message=Invalid parameters"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id}/resource-history [get]
// @Security Bearer
func (h *ClusterHandler) ListResourceHistory(c *gin.Context) {
	ctx := c.Request.Context()

	// Get cluster from context
	cluster, ok := middleware.GetClusterFromContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to get cluster from context",
		})
		return
	}

	// Parse query parameters
	filter := &repository.ResourceHistoryFilter{
		ResourceType:  c.Query("resource_type"),
		ResourceName:  c.Query("resource_name"),
		Namespace:     c.Query("namespace"),
		APIGroup:      c.Query("api_group"),
		APIVersion:    c.Query("api_version"),
		OperationType: c.Query("operation_type"),
		Page:          1,
		PageSize:      20,
	}

	// Parse success filter
	if successStr := c.Query("success"); successStr != "" {
		if success, err := strconv.ParseBool(successStr); err == nil {
			filter.Success = &success
		}
	}

	// Parse time filters
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &startTime
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &endTime
		}
	}

	// Parse pagination
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filter.Page = page
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
			filter.PageSize = pageSize
			if filter.PageSize > 100 {
				filter.PageSize = 100 // Cap at 100
			}
		}
	}

	// Query history
	histories, total, err := h.historyRepo.ListByCluster(ctx, cluster.ID, filter)
	if err != nil {
		logrus.Errorf("Failed to list resource history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to list resource history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Resource history retrieved successfully",
		"data": gin.H{
			"items":     histories,
			"total":     total,
			"page":      filter.Page,
			"page_size": filter.PageSize,
		},
	})
}

// GetResourceHistory godoc
// @Summary Get resource operation history detail
// @Description Get detailed information about a specific resource operation
// @Tags k8s-resource-history
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param history_id path string true "History ID (UUID)"
// @Success 200 {object} map[string]interface{} "code=200, data={history}"
// @Failure 404 {object} map[string]interface{} "code=404, message=History not found"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id}/resource-history/{history_id} [get]
// @Security Bearer
func (h *ClusterHandler) GetResourceHistory(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse history ID
	historyIDStr := c.Param("history_id")
	historyID, err := uuid.Parse(historyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Invalid history ID format",
		})
		return
	}

	// Retrieve history
	history, err := h.historyRepo.GetByID(ctx, historyID)
	if err != nil {
		logrus.Errorf("Failed to get resource history %s: %v", historyID, err)
		c.JSON(http.StatusNotFound, gin.H{
			"code":    http.StatusNotFound,
			"message": "Resource history not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Resource history retrieved successfully",
		"data":    history,
	})
}

// DeleteResourceHistory godoc
// @Summary Delete resource operation history
// @Description Delete a specific resource operation history record
// @Tags k8s-resource-history
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param history_id path string true "History ID (UUID)"
// @Success 200 {object} map[string]interface{} "code=200, message=success"
// @Failure 404 {object} map[string]interface{} "code=404, message=History not found"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id}/resource-history/{history_id} [delete]
// @Security Bearer
func (h *ClusterHandler) DeleteResourceHistory(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse history ID
	historyIDStr := c.Param("history_id")
	historyID, err := uuid.Parse(historyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Invalid history ID format",
		})
		return
	}

	// Delete history
	if err := h.historyRepo.Delete(ctx, historyID); err != nil {
		logrus.Errorf("Failed to delete resource history %s: %v", historyID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to delete resource history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Resource history deleted successfully",
	})
}
