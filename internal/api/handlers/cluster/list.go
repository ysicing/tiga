package cluster

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/repository"
)

// ClusterHandler handles cluster-related HTTP requests (Phase 3 更新)
type ClusterHandler struct {
	clusterRepo repository.ClusterRepositoryInterface
	historyRepo repository.ResourceHistoryRepositoryInterface
	cfg         *config.Config
}

// NewClusterHandler creates a new ClusterHandler instance
func NewClusterHandler(
	clusterRepo repository.ClusterRepositoryInterface,
	historyRepo repository.ResourceHistoryRepositoryInterface,
	cfg *config.Config,
) *ClusterHandler {
	return &ClusterHandler{
		clusterRepo: clusterRepo,
		historyRepo: historyRepo,
		cfg:         cfg,
	}
}

// List godoc
// @Summary List all clusters
// @Description Get a list of all Kubernetes clusters
// @Tags k8s-clusters
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "code=200, data={clusters:[], total:int}"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters [get]
// @Security Bearer
func (h *ClusterHandler) List(c *gin.Context) {
	ctx := c.Request.Context()

	clusters, err := h.clusterRepo.List(ctx)
	if err != nil {
		logrus.Errorf("Failed to list clusters: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": fmt.Sprintf("Failed to retrieve clusters: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Clusters retrieved successfully",
		"data": gin.H{
			"clusters": clusters,
			"total":    len(clusters),
		},
	})
}
