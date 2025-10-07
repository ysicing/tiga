package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ysicing/tiga/internal/services/host"
)

// WebSocketHandler handles real-time WebSocket connections
type WebSocketHandler struct {
	stateCollector *host.StateCollector
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(stateCollector *host.StateCollector) *WebSocketHandler {
	return &WebSocketHandler{stateCollector: stateCollector}
}

// HostMonitor handles WebSocket for host monitoring
func (h *WebSocketHandler) HostMonitor(c *gin.Context) {
	// TODO: Upgrade to WebSocket
	// Parse subscribe request to get host IDs
	// Create subscriber
	// Stream state updates

	c.JSON(http.StatusOK, gin.H{
		"message": "WebSocket endpoint for host monitoring",
		"usage":   "Send {action: 'subscribe', host_ids: [1,2,3]}",
	})
}

// ServiceProbe handles WebSocket for service probe updates
func (h *WebSocketHandler) ServiceProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "WebSocket endpoint for service probe",
	})
}

// AlertEvents handles WebSocket for alert event updates
func (h *WebSocketHandler) AlertEvents(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "WebSocket endpoint for alert events",
	})
}
