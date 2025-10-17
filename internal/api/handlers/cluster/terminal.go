package cluster

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"

	"github.com/ysicing/tiga/internal/api/middleware"
	"github.com/ysicing/tiga/pkg/kube"
)

// PodTerminal godoc
// @Summary Pod WebSocket terminal
// @Description WebSocket connection for pod terminal access
// @Tags k8s-terminal
// @Param id path string true "Cluster ID (UUID)"
// @Param namespace query string true "Pod namespace"
// @Param pod query string true "Pod name"
// @Param container query string false "Container name (default: first container)"
// @Router /api/v1/k8s/clusters/{id}/terminal [get]
// @Security Bearer
func (h *ClusterHandler) PodTerminal(c *gin.Context) {
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
	namespace := c.Query("namespace")
	podName := c.Query("pod")
	container := c.Query("container")

	if namespace == "" || podName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Missing required parameters: namespace, pod",
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

	// WebSocket handler
	handler := func(ws *websocket.Conn) {
		defer ws.Close()

		ctx := c.Request.Context()
		session := kube.NewTerminalSession(client, ws, namespace, podName, container)
		defer session.Close()

		logrus.Infof("Terminal session started: %s/%s (container: %s)", namespace, podName, container)

		if err := session.Start(ctx, "exec"); err != nil {
			logrus.Errorf("Terminal session error: %v", err)
			return
		}

		logrus.Infof("Terminal session closed: %s/%s", namespace, podName)
	}

	websocket.Handler(handler).ServeHTTP(c.Writer, c.Request)
}
