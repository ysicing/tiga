package cluster

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/middleware"
	"github.com/ysicing/tiga/pkg/handlers/resources/kruise"
)

// ListCloneSets godoc
// @Summary List CloneSets
// @Description List all OpenKruise CloneSets in a namespace
// @Tags k8s-crd
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param namespace query string false "Namespace (default: all namespaces)"
// @Success 200 {object} map[string]interface{} "code=200, data={items:[], total:int}"
// @Failure 404 {object} map[string]interface{} "code=404, message=CRD not installed"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id}/clonesets [get]
// @Security Bearer
func (h *ClusterHandler) ListCloneSets(c *gin.Context) {
	ctx := c.Request.Context()

	// Get cluster from context (set by ClusterContext middleware)
	cluster, ok := middleware.GetClusterFromContext(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to get cluster from context",
		})
		return
	}

	// Get namespace from query parameter (optional)
	namespace := c.Query("namespace")

	// Create K8s client
	// TODO: Use injected client cache
	client, err := h.createK8sClient(cluster)
	if err != nil {
		logrus.Errorf("Failed to create K8s client: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "Failed to connect to cluster",
		})
		return
	}

	// Create CloneSet handler
	handler := kruise.NewCloneSetHandler(client)

	// List CloneSets
	list, err := handler.List(ctx, namespace)
	if err != nil {
		logrus.Errorf("Failed to list CloneSets: %v", err)
		// Check if it's a CRD not found error
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    http.StatusNotFound,
				"message": "CustomResourceDefinition 'clonesets.apps.kruise.io' not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to list CloneSets",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "CloneSets retrieved successfully",
		"data": gin.H{
			"items": list.Items,
			"total": len(list.Items),
		},
	})
}

// ScaleCloneSet godoc
// @Summary Scale CloneSet
// @Description Scale an OpenKruise CloneSet to a specified replica count
// @Tags k8s-crd
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param name path string true "CloneSet name"
// @Param namespace query string true "Namespace"
// @Param body body map[string]interface{} true "Scale request {replicas: int}"
// @Success 200 {object} map[string]interface{} "code=200, message=success"
// @Failure 400 {object} map[string]interface{} "code=400, message=Invalid request"
// @Failure 404 {object} map[string]interface{} "code=404, message=CloneSet not found"
// @Router /api/v1/k8s/clusters/{id}/clonesets/{name}/scale [put]
// @Security Bearer
func (h *ClusterHandler) ScaleCloneSet(c *gin.Context) {
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

	// Get parameters
	name := c.Param("name")
	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Parse request body
	var req struct {
		Replicas int32 `json:"replicas" binding:"required,min=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Invalid request: replicas is required",
		})
		return
	}

	// Create K8s client
	client, err := h.createK8sClient(cluster)
	if err != nil {
		logrus.Errorf("Failed to create K8s client: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "Failed to connect to cluster",
		})
		return
	}

	// Create CloneSet handler
	handler := kruise.NewCloneSetHandler(client)

	// Scale CloneSet
	if err := handler.SetReplicas(ctx, namespace, name, req.Replicas); err != nil {
		logrus.Errorf("Failed to scale CloneSet %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to scale CloneSet",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "CloneSet scaled successfully",
	})
}

// RestartCloneSet godoc
// @Summary Restart CloneSet
// @Description Trigger a rolling restart of an OpenKruise CloneSet
// @Tags k8s-crd
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param name path string true "CloneSet name"
// @Param namespace query string true "Namespace"
// @Success 200 {object} map[string]interface{} "code=200, message=success"
// @Failure 404 {object} map[string]interface{} "code=404, message=CloneSet not found"
// @Router /api/v1/k8s/clusters/{id}/clonesets/{name}/restart [post]
// @Security Bearer
func (h *ClusterHandler) RestartCloneSet(c *gin.Context) {
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

	// Get parameters
	name := c.Param("name")
	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Create K8s client
	client, err := h.createK8sClient(cluster)
	if err != nil {
		logrus.Errorf("Failed to create K8s client: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "Failed to connect to cluster",
		})
		return
	}

	// Create CloneSet handler
	handler := kruise.NewCloneSetHandler(client)

	// Restart CloneSet
	if err := handler.Restart(ctx, namespace, name); err != nil {
		logrus.Errorf("Failed to restart CloneSet %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to restart CloneSet",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "CloneSet restart triggered successfully",
	})
}

// isNotFoundError checks if an error is a "not found" error
func isNotFoundError(err error) bool {
	return err != nil && (err.Error() == "NotFound" ||
		contains(err.Error(), "not found") ||
		contains(err.Error(), "no matches for kind"))
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
