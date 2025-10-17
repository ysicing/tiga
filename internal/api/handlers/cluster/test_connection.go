package cluster

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// TestConnectionRequest represents the request body for testing connection
type TestConnectionRequest struct {
	Config string `json:"config" binding:"required"` // Kubeconfig content
}

// TestConnection godoc
// @Summary Test cluster connection
// @Description Test connection to a Kubernetes cluster using provided kubeconfig
// @Tags k8s-clusters
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param config body TestConnectionRequest false "Kubeconfig for testing (optional, uses stored config if not provided)"
// @Success 200 {object} map[string]interface{} "code=200, data={connected:true, server_version:string}"
// @Failure 400 {object} map[string]interface{} "code=400, message=Invalid request"
// @Failure 404 {object} map[string]interface{} "code=404, message=Cluster not found"
// @Failure 503 {object} map[string]interface{} "code=503, message=Connection failed"
// @Router /api/v1/k8s/clusters/{id}/test-connection [post]
// @Security Bearer
func (h *ClusterHandler) TestConnection(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

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

	// Try to parse request body for optional config override
	var req TestConnectionRequest
	configToTest := cluster.Config
	if err := c.ShouldBindJSON(&req); err == nil && req.Config != "" {
		// Use provided config for testing
		configToTest = req.Config
	}

	// Decrypt stored config if using it
	if configToTest == cluster.Config {
		// TODO: Decrypt config using utils.DecryptString
		// For now, assume config is stored in plain text or already decrypted
	}

	// Parse kubeconfig
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(configToTest))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": fmt.Sprintf("Invalid kubeconfig: %v", err),
		})
		return
	}

	// Create client
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": fmt.Sprintf("Failed to create client: %v", err),
			"data": gin.H{
				"connected": false,
				"error":     err.Error(),
			},
		})
		return
	}

	// Test connection by calling ServerVersion
	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": fmt.Sprintf("Connection failed: %v", err),
			"data": gin.H{
				"connected": false,
				"error":     err.Error(),
			},
		})
		return
	}

	// Connection successful
	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "Connection test successful",
		"data": gin.H{
			"connected":      true,
			"server_version": version.GitVersion,
			"platform":       version.Platform,
		},
	})
}
