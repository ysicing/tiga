package cluster

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ysicing/tiga/pkg/utils"
)

// UpdateClusterRequest represents the request body for updating a cluster
type UpdateClusterRequest struct {
	Name          *string `json:"name"`
	Description   *string `json:"description"`
	Config        *string `json:"config"` // Kubeconfig content
	PrometheusURL *string `json:"prometheus_url"`
	InCluster     *bool   `json:"in_cluster"`
	IsDefault     *bool   `json:"is_default"`
	Enable        *bool   `json:"enable"`
}

// Update godoc
// @Summary Update a cluster
// @Description Update an existing Kubernetes cluster (partial update supported)
// @Tags k8s-clusters
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param cluster body UpdateClusterRequest true "Cluster update data"
// @Success 200 {object} map[string]interface{} "code=200, data=cluster"
// @Failure 400 {object} map[string]interface{} "code=400, message=Validation error"
// @Failure 404 {object} map[string]interface{} "code=404, message=Cluster not found"
// @Failure 409 {object} map[string]interface{} "code=409, message=Cluster name already exists"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id} [put]
// @Security Bearer
func (h *ClusterHandler) Update(c *gin.Context) {
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

	var req UpdateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Build updates map (only include non-nil fields)
	updates := make(map[string]interface{})

	if req.Name != nil {
		// Check if new name conflicts with existing cluster
		if *req.Name != cluster.Name {
			existing, err := h.clusterRepo.GetByName(ctx, *req.Name)
			if err != nil && err != gorm.ErrRecordNotFound {
				logrus.Errorf("Failed to check cluster name: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    http.StatusInternalServerError,
					"message": "Failed to validate cluster name",
				})
				return
			}
			if existing != nil {
				c.JSON(http.StatusConflict, gin.H{
					"code":    http.StatusConflict,
					"message": "Cluster name already exists",
				})
				return
			}
		}
		updates["name"] = *req.Name
	}

	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if req.Config != nil {
		// Validate kubeconfig
		if _, err := clientcmd.RESTConfigFromKubeConfig([]byte(*req.Config)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": fmt.Sprintf("Invalid kubeconfig: %v", err),
			})
			return
		}
		// Encrypt kubeconfig
		updates["config"] = utils.EncryptStringWithKey(*req.Config, h.cfg.Security.EncryptionKey)
	}

	if req.PrometheusURL != nil {
		updates["prometheus_url"] = *req.PrometheusURL
	}

	if req.InCluster != nil {
		updates["in_cluster"] = *req.InCluster
	}

	if req.IsDefault != nil {
		// If setting as default, clear other defaults first
		if *req.IsDefault {
			if err := h.clusterRepo.ClearDefault(ctx); err != nil {
				logrus.Warnf("Failed to clear existing default: %v", err)
			}
		}
		updates["is_default"] = *req.IsDefault
	}

	if req.Enable != nil {
		updates["enable"] = *req.Enable
	}

	// Perform update
	if len(updates) > 0 {
		if err := h.clusterRepo.Update(ctx, clusterID, updates); err != nil {
			logrus.Errorf("Failed to update cluster %s: %v", clusterID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": fmt.Sprintf("Failed to update cluster: %v", err),
			})
			return
		}
	}

	// Fetch updated cluster
	updatedCluster, err := h.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		logrus.Errorf("Failed to fetch updated cluster: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Cluster updated but failed to retrieve updated data",
		})
		return
	}

	// Don't expose encrypted config
	updatedCluster.Config = ""

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Cluster updated successfully",
		"data":    updatedCluster,
	})
}
