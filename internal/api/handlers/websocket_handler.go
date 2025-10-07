package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/ysicing/tiga/internal/services/host"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: Add proper origin checking in production
	},
}

// WebSocketHandler handles real-time WebSocket connections
type WebSocketHandler struct {
	stateCollector *host.StateCollector
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(stateCollector *host.StateCollector) *WebSocketHandler {
	return &WebSocketHandler{stateCollector: stateCollector}
}

// WebSocketMessage represents messages between client and server
type WebSocketMessage struct {
	Action  string   `json:"action"`  // "subscribe" | "unsubscribe" | "state_update"
	HostIDs []string `json:"host_ids,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// HostMonitor handles WebSocket for host monitoring
func (h *WebSocketHandler) HostMonitor(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("Failed to upgrade WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Generate unique subscriber ID
	subscriberID := uuid.New().String()
	var subscriber *host.StateSubscriber
	var subMutex sync.RWMutex

	// Ensure cleanup on exit
	defer func() {
		subMutex.Lock()
		if subscriber != nil {
			h.stateCollector.Unsubscribe(subscriberID)
		}
		subMutex.Unlock()
	}()

	logrus.Infof("[WebSocket] Client connected: %s", subscriberID)

	// Channel to handle graceful shutdown
	done := make(chan struct{})

	// Goroutine to read messages from client
	go func() {
		defer close(done)
		for {
			var msg WebSocketMessage
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logrus.Warnf("[WebSocket] Read error: %v", err)
				}
				return
			}

			logrus.Debugf("[WebSocket] Received message: %+v", msg)

			switch msg.Action {
			case "subscribe":
				// Parse host IDs
				var hostIDs []uuid.UUID
				if len(msg.HostIDs) > 0 {
					for _, idStr := range msg.HostIDs {
						if id, err := uuid.Parse(idStr); err == nil {
							hostIDs = append(hostIDs, id)
						}
					}
				}

				subMutex.Lock()
				// Unsubscribe existing subscription if any
				if subscriber != nil {
					h.stateCollector.Unsubscribe(subscriberID)
				}

				// Create new subscription
				subscriber = h.stateCollector.Subscribe(subscriberID, hostIDs)
				subMutex.Unlock()

				logrus.Infof("[WebSocket] Subscribed to %d hosts", len(hostIDs))

			case "unsubscribe":
				subMutex.Lock()
				if subscriber != nil {
					h.stateCollector.Unsubscribe(subscriberID)
					subscriber = nil
					logrus.Infof("[WebSocket] Unsubscribed: %s", subscriberID)
				}
				subMutex.Unlock()
			}
		}
	}()

	// Goroutine to send state updates to client
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				// Send ping to keep connection alive
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					logrus.Warnf("[WebSocket] Ping error: %v", err)
					return
				}
			default:
				// Check if subscriber exists before reading from channel
				subMutex.RLock()
				sub := subscriber
				subMutex.RUnlock()

				if sub != nil {
					select {
					case state, ok := <-sub.Channel:
						if !ok {
							return
						}

						// Send state update to client
						msg := WebSocketMessage{
							Action: "state_update",
							Data: map[string]interface{}{
								"host_id":         state.HostNodeID.String(),
								"timestamp":       state.Timestamp,
								"cpu_usage":       state.CPUUsage,
								"load_1":          state.Load1,
								"load_5":          state.Load5,
								"load_15":         state.Load15,
								"mem_used":        state.MemUsed,
								"mem_usage":       state.MemUsage,
								"swap_used":       state.SwapUsed,
								"disk_used":       state.DiskUsed,
								"disk_usage":      state.DiskUsage,
								"net_in_transfer": state.NetInTransfer,
								"net_out_transfer": state.NetOutTransfer,
								"net_in_speed":    state.NetInSpeed,
								"net_out_speed":   state.NetOutSpeed,
								"tcp_conn_count":  state.TCPConnCount,
								"udp_conn_count":  state.UDPConnCount,
								"process_count":   state.ProcessCount,
								"uptime":          state.Uptime,
							},
						}

						if err := conn.WriteJSON(msg); err != nil {
							logrus.Warnf("[WebSocket] Write error: %v", err)
							return
						}
					default:
						// No data available, sleep briefly to avoid busy loop
						time.Sleep(100 * time.Millisecond)
					}
				} else {
					// No subscriber yet, sleep briefly
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}()

	// Wait for done signal
	<-done
	logrus.Infof("[WebSocket] Client disconnected: %s", subscriberID)
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
