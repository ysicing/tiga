package cluster

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/pkg/utils"
)

// CreateClusterRequest represents the request body for creating a cluster
type CreateClusterRequest struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	Config        string `json:"config" binding:"required"` // Kubeconfig content
	PrometheusURL string `json:"prometheus_url"`
	InCluster     bool   `json:"in_cluster"`
	IsDefault     bool   `json:"is_default"`
}

// Create godoc
// @Summary Create a new cluster
// @Description Create a new Kubernetes cluster
// @Tags k8s-clusters
// @Accept json
// @Produce json
// @Param cluster body CreateClusterRequest true "Cluster data"
// @Success 201 {object} map[string]interface{} "code=201, data=cluster"
// @Failure 400 {object} map[string]interface{} "code=400, message=Validation error"
// @Failure 409 {object} map[string]interface{} "code=409, message=Cluster name already exists"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters [post]
// @Security Bearer
func (h *ClusterHandler) Create(c *gin.Context) {
	ctx := c.Request.Context()

	var req CreateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Validate kubeconfig
	if _, err := clientcmd.RESTConfigFromKubeConfig([]byte(req.Config)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": fmt.Sprintf("Invalid kubeconfig: %v", err),
		})
		return
	}

	// Check if cluster name already exists
	existing, err := h.clusterRepo.GetByName(ctx, req.Name)
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

	// Encrypt kubeconfig before storing
	encryptedConfig := utils.EncryptStringWithKey(req.Config, h.cfg.Security.EncryptionKey)

	// Create cluster model
	cluster := &models.Cluster{
		Name:          req.Name,
		Description:   req.Description,
		Config:        encryptedConfig,
		PrometheusURL: req.PrometheusURL,
		InCluster:     req.InCluster,
		IsDefault:     req.IsDefault,
		Enable:        true,
		HealthStatus:  models.ClusterHealthUnknown,
	}

	// If this is set as default, clear other defaults first
	if req.IsDefault {
		if err := h.clusterRepo.ClearDefault(ctx); err != nil {
			logrus.Warnf("Failed to clear existing default: %v", err)
		}
	}

	if err := h.clusterRepo.Create(ctx, cluster); err != nil {
		logrus.Errorf("Failed to create cluster: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": fmt.Sprintf("Failed to create cluster: %v", err),
		})
		return
	}

	// Decrypt config before returning (don't expose encrypted config to client)
	cluster.Config = ""

	c.JSON(http.StatusCreated, gin.H{
		"code":    http.StatusCreated,
		"message": "Cluster created successfully",
		"data":    cluster,
	})
}
