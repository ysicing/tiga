package cluster

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Get godoc
// @Summary Get cluster by ID
// @Description Get a single Kubernetes cluster by ID
// @Tags k8s-clusters
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Success 200 {object} map[string]interface{} "code=200, data=cluster"
// @Failure 400 {object} map[string]interface{} "code=400, message=Invalid ID"
// @Failure 404 {object} map[string]interface{} "code=404, message=Cluster not found"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id} [get]
// @Security Bearer
func (h *ClusterHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse cluster ID from URL parameter
	idStr := c.Param("id")
	clusterID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Invalid cluster ID format",
		})
		return
	}

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
			"message": fmt.Sprintf("Failed to retrieve cluster: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Cluster retrieved successfully",
		"data":    cluster,
	})
}
