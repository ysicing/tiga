package cluster

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/middleware"
	"github.com/ysicing/tiga/pkg/handlers/resources/kruise"
)

// ListStatefulSets godoc
// @Summary List Advanced StatefulSets
// @Description List all OpenKruise Advanced StatefulSets in a namespace
// @Tags k8s-crd
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param namespace query string false "Namespace (default: all namespaces)"
// @Success 200 {object} map[string]interface{} "code=200, data={items:[], total:int}"
// @Failure 404 {object} map[string]interface{} "code=404, message=CRD not installed"
// @Failure 500 {object} map[string]interface{} "code=500, message=error"
// @Router /api/v1/k8s/clusters/{id}/statefulsets [get]
// @Security Bearer
func (h *ClusterHandler) ListStatefulSets(c *gin.Context) {
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

	// Create StatefulSet handler
	handler := kruise.NewStatefulSetHandler(client)

	// List StatefulSets
	list, err := handler.List(ctx, namespace)
	if err != nil {
		logrus.Errorf("Failed to list StatefulSets: %v", err)
		// Check if it's a CRD not found error
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    http.StatusNotFound,
				"message": "CustomResourceDefinition 'statefulsets.apps.kruise.io' not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to list StatefulSets",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "StatefulSets retrieved successfully",
		"data": gin.H{
			"items": list.Items,
			"total": len(list.Items),
		},
	})
}

// ScaleStatefulSet godoc
// @Summary Scale Advanced StatefulSet
// @Description Scale an OpenKruise Advanced StatefulSet to a specified replica count
// @Tags k8s-crd
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param name path string true "StatefulSet name"
// @Param namespace query string true "Namespace"
// @Param body body map[string]interface{} true "Scale request {replicas: int}"
// @Success 200 {object} map[string]interface{} "code=200, message=success"
// @Failure 400 {object} map[string]interface{} "code=400, message=Invalid request"
// @Failure 404 {object} map[string]interface{} "code=404, message=StatefulSet not found"
// @Router /api/v1/k8s/clusters/{id}/statefulsets/{name}/scale [put]
// @Security Bearer
func (h *ClusterHandler) ScaleStatefulSet(c *gin.Context) {
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

	// Create StatefulSet handler
	handler := kruise.NewStatefulSetHandler(client)

	// Scale StatefulSet
	if err := handler.SetReplicas(ctx, namespace, name, req.Replicas); err != nil {
		logrus.Errorf("Failed to scale StatefulSet %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to scale StatefulSet",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "StatefulSet scaled successfully",
	})
}

// RestartStatefulSet godoc
// @Summary Restart Advanced StatefulSet
// @Description Trigger a rolling restart of an OpenKruise Advanced StatefulSet
// @Tags k8s-crd
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID (UUID)"
// @Param name path string true "StatefulSet name"
// @Param namespace query string true "Namespace"
// @Success 200 {object} map[string]interface{} "code=200, message=success"
// @Failure 404 {object} map[string]interface{} "code=404, message=StatefulSet not found"
// @Router /api/v1/k8s/clusters/{id}/statefulsets/{name}/restart [post]
// @Security Bearer
func (h *ClusterHandler) RestartStatefulSet(c *gin.Context) {
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

	// Create StatefulSet handler
	handler := kruise.NewStatefulSetHandler(client)

	// Restart StatefulSet
	if err := handler.Restart(ctx, namespace, name); err != nil {
		logrus.Errorf("Failed to restart StatefulSet %s/%s: %v", namespace, name, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "Failed to restart StatefulSet",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "StatefulSet restart triggered successfully",
	})
}
