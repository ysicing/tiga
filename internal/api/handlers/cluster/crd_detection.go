package cluster

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/pkg/handlers/resources/crd"
	"github.com/ysicing/tiga/pkg/kube"
)

// DetectCRDs godoc
// @Summary Detect installed CRDs
// @Description Detect installed Custom Resource Definitions in the cluster
// @Tags k8s-clusters
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Success 200 {object} map[string]interface{} "code=200, data={kruise:{installed:bool, crds:[]}, ...}"
// @Failure 400 {object} map[string]interface{} "code=400, message=Invalid ID"
// @Failure 404 {object} map[string]interface{} "code=404, message=Cluster not found"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id}/crds [get]
// @Security Bearer
func (h *ClusterHandler) DetectCRDs(c *gin.Context) {
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

	// Get or create K8s client
	// TODO: This should be injected or retrieved from a service
	// For now, create a temporary client
	client, err := h.createK8sClient(cluster)
	if err != nil {
		logrus.Errorf("Failed to create K8s client for cluster %s: %v", cluster.Name, err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "Failed to connect to cluster",
		})
		return
	}

	// Create detection service
	detector := crd.NewCRDDetectionService()

	// Detect all CRDs
	result, err := detector.DetectAll(ctx, client)
	if err != nil {
		logrus.Errorf("Failed to detect CRDs for cluster %s: %v", cluster.Name, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to detect CRDs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "CRD detection completed",
		"data":    result,
	})
}

// createK8sClient is a helper to create K8s client (temporary implementation)
// TODO: Move this to a service layer or inject client cache
func (h *ClusterHandler) createK8sClient(cluster interface{}) (*kube.K8sClient, error) {
	// This is a placeholder - in production, should use injected client cache
	// For now, return an error to indicate not implemented
	return nil, nil
}
