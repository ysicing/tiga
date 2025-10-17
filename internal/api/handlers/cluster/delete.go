package cluster

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Delete godoc
// @Summary Delete a cluster
// @Description Soft delete a Kubernetes cluster
// @Tags k8s-clusters
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Success 200 {object} map[string]interface{} "code=200, message=success"
// @Failure 400 {object} map[string]interface{} "code=400, message=Invalid ID"
// @Failure 404 {object} map[string]interface{} "code=404, message=Cluster not found"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id} [delete]
// @Security Bearer
func (h *ClusterHandler) Delete(c *gin.Context) {
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

	// Check if cluster exists
	_, err = h.clusterRepo.GetByID(ctx, clusterID)
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

	// Perform soft delete
	if err := h.clusterRepo.Delete(ctx, clusterID); err != nil {
		logrus.Errorf("Failed to delete cluster %s: %v", clusterID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to delete cluster",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Cluster deleted successfully",
	})
}
