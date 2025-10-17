package cluster

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/services/prometheus"
)

// RediscoverPrometheus godoc
// @Summary Trigger Prometheus rediscovery
// @Description Manually trigger Prometheus service discovery for a cluster (async operation)
// @Tags k8s-clusters
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Success 202 {object} map[string]interface{} "code=202, message=Task started, data={cluster_id, task_started_at}"
// @Failure 400 {object} map[string]interface{} "code=400, message=Invalid ID"
// @Failure 404 {object} map[string]interface{} "code=404, message=Cluster not found"
// @Failure 409 {object} map[string]interface{} "code=409, message=Discovery task already running"
// @Router /api/v1/k8s/clusters/{id}/prometheus/rediscover [post]
// @Security Bearer
func (h *ClusterHandler) RediscoverPrometheus(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse cluster ID
	idStr := c.Param("id")
	clusterID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Invalid cluster ID format",
		})
		return
	}

	// Get cluster from database
	cluster, err := h.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    http.StatusNotFound,
				"message": "Cluster not found",
			})
			return
		}
		logrus.Errorf("Failed to get cluster %s: %v", clusterID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to retrieve cluster",
		})
		return
	}

	// Check if a discovery task is already running
	// This requires the prometheus discovery service to be injected
	// For now, we'll just trigger the discovery

	// TODO: Inject prometheus.AutoDiscoveryService into handler
	// For MVP, we'll create a temporary service instance
	// In production, this should be injected via DI
	discoveryService := prometheus.NewAutoDiscoveryService(h.clusterRepo, nil, h.cfg)

	// Check if task is already running
	if discoveryService.IsTaskRunning(clusterID) {
		c.JSON(http.StatusConflict, gin.H{
			"code":    http.StatusConflict,
			"message": "Discovery task is already running for this cluster",
		})
		return
	}

	// Trigger async discovery
	discoveryService.TriggerDiscoveryAsync(cluster)

	// Return 202 Accepted (async operation)
	c.JSON(http.StatusAccepted, gin.H{
		"code":    http.StatusAccepted,
		"message": "Prometheus discovery task started, check prometheus_url field later",
		"data": gin.H{
			"cluster_id":      clusterID,
			"task_started_at": time.Now().Format(time.RFC3339),
		},
	})
}
