package cluster

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/middleware"
	"github.com/ysicing/tiga/pkg/handlers/resources/kruise"
)

// ListDaemonSets godoc
// @Summary List Advanced DaemonSets
// @Description List all OpenKruise Advanced DaemonSets in a namespace
// @Tags k8s-crd
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param namespace query string false "Namespace (default: all namespaces)"
// @Success 200 {object} map[string]interface{} "code=200, data={items:[], total:int}"
// @Failure 404 {object} map[string]interface{} "code=404, message=CRD not installed"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id}/daemonsets [get]
// @Security Bearer
func (h *ClusterHandler) ListDaemonSets(c *gin.Context) {
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
	client, err := h.createK8sClient(cluster)
	if err != nil {
		logrus.Errorf("Failed to create K8s client: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "Failed to connect to cluster",
		})
		return
	}

	// Create DaemonSet handler
	handler := kruise.NewDaemonSetHandler(client)

	// List DaemonSets
	list, err := handler.List(ctx, namespace)
	if err != nil {
		logrus.Errorf("Failed to list DaemonSets: %v", err)
		// Check if it's a CRD not found error
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    http.StatusNotFound,
				"message": "CustomResourceDefinition 'daemonsets.apps.kruise.io' not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to list DaemonSets",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "DaemonSets retrieved successfully",
		"data": gin.H{
			"items": list.Items,
			"total": len(list.Items),
		},
	})
}

// RestartDaemonSet godoc
// @Summary Restart Advanced DaemonSet
// @Description Trigger a rolling restart of an OpenKruise Advanced DaemonSet
// @Tags k8s-crd
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param name path string true "DaemonSet name"
// @Param namespace query string true "Namespace"
// @Success 200 {object} map[string]interface{} "code=200, message=success"
// @Failure 404 {object} map[string]interface{} "code=404, message=DaemonSet not found"
// @Router /api/v1/k8s/clusters/{id}/daemonsets/{name}/restart [post]
// @Security Bearer
func (h *ClusterHandler) RestartDaemonSet(c *gin.Context) {
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

	// Create DaemonSet handler
	handler := kruise.NewDaemonSetHandler(client)

	// Restart DaemonSet
	if err := handler.Restart(ctx, namespace, name); err != nil {
		logrus.Errorf("Failed to restart DaemonSet %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to restart DaemonSet",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "DaemonSet restart triggered successfully",
	})
}
