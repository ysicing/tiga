package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/services/host"
	"github.com/ysicing/tiga/internal/services/webssh"
	"github.com/ysicing/tiga/proto"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  10240,
	WriteBufferSize: 10240,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: Add proper origin checking
	},
}

// WebSSHHandler handles WebSSH operations
type WebSSHHandler struct {
	sessionMgr      *webssh.SessionManager
	terminalMgr     *host.TerminalManager
	agentManager    *host.AgentManager
}

// NewWebSSHHandler creates a new WebSSH handler
func NewWebSSHHandler(sessionMgr *webssh.SessionManager, terminalMgr *host.TerminalManager, agentMgr *host.AgentManager) *WebSSHHandler {
	return &WebSSHHandler{
		sessionMgr:      sessionMgr,
		terminalMgr:     terminalMgr,
		agentManager:    agentMgr,
	}
}

// CreateSession creates a WebSSH session
func (h *WebSSHHandler) CreateSession(c *gin.Context) {
	var req struct {
		HostID string `json:"host_id" binding:"required"` // UUID string
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid request"})
		return
	}

	// Parse host UUID
	hostUUID, err := uuid.Parse(req.HostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid host ID"})
		return
	}

	// Get host info from agent manager
	conn := h.agentManager.GetConnectionByHostID(hostUUID)
	if conn == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 40404, "message": "Host not online"})
		return
	}

	// Generate session ID
	streamID := uuid.New().String()

	// Create terminal session
	_ = h.terminalMgr.CreateSession(streamID, hostUUID, conn.UUID)

	// Create webssh session for recording
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001") // TODO: Get from auth context
	clientIP := c.ClientIP()
	wsSession, err := h.sessionMgr.CreateSession(c.Request.Context(), userID, hostUUID, req.Width, req.Height, clientIP)
	if err != nil {
		logrus.Errorf("Failed to create webssh session: %v", err)
	}

	// Notify Agent to create terminal via gRPC Task
	// The Agent will connect to IOStream with the streamID
	if err := h.notifyAgentCreateTerminal(conn.UUID, streamID); err != nil {
		h.terminalMgr.CloseSession(streamID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to notify agent"})
		return
	}

	// Build WebSocket URL
	scheme := "wss"
	if c.Request.TLS == nil {
		scheme = "ws"
	}
	wsURL := scheme + "://" + c.Request.Host + "/api/v1/vms/webssh/" + streamID

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"session_id":    streamID,
			"ws_session_id": wsSession.SessionID,
			"websocket_url": wsURL,
			"host_id":       req.HostID,
		},
	})
}

// ListSessions lists WebSSH sessions
func (h *WebSSHHandler) ListSessions(c *gin.Context) {
	sessions := h.sessionMgr.ListActiveSessions()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    sessions,
	})
}

// CloseSession closes a WebSSH session
func (h *WebSSHHandler) CloseSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	if err := h.sessionMgr.CloseSession(c.Request.Context(), sessionID, "User requested"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to close session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "会话已关闭"})
}

// HandleWebSocket handles WebSocket connection for terminal
func (h *WebSSHHandler) HandleWebSocket(c *gin.Context) {
	streamID := c.Param("session_id")

	// Get terminal session
	session, exists := h.terminalMgr.GetSession(streamID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"code": 40404, "message": "Session not found"})
		return
	}

	// Upgrade to WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("WebSocket upgrade failed: %v", err)
		return
	}
	defer ws.Close()

	logrus.Infof("WebSocket connected for session: %s", streamID)

	// Set WebSocket timeouts
	ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	done := make(chan struct{})
	defer close(done)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Read from WebSocket and send to Agent
	go func() {
		defer session.SendToAgent([]byte{0xff}) // Signal end
		for {
			messageType, data, err := ws.ReadMessage()
			if err != nil {
				logrus.Errorf("WebSocket read error: %v", err)
				return
			}

			if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
				// Parse message from frontend
				var msg map[string]interface{}
				if err := json.Unmarshal(data, &msg); err == nil {
					if msgType, ok := msg["type"].(string); ok {
						switch msgType {
						case "resize":
							// Window resize: {"type":"resize","cols":80,"rows":24}
							cols := int(msg["cols"].(float64))
							rows := int(msg["rows"].(float64))
							resizeData, _ := json.Marshal(map[string]int{"cols": cols, "rows": rows})
							session.SendToAgent(append([]byte{0x01}, resizeData...))
						case "input":
							// Terminal input: {"type":"input","data":"ls\n"}
							if input, ok := msg["data"].(string); ok {
								session.SendToAgent(append([]byte{0x00}, []byte(input)...))
							}
						}
						continue
					}
				}

				// Fallback: treat as raw input
				session.SendToAgent(append([]byte{0x00}, data...))
			}
		}
	}()

	// Read from Agent and send to WebSocket
	for {
		data, err := session.ReceiveFromAgent()
		if err != nil {
			logrus.Errorf("Failed to receive from agent: %v", err)
			break
		}

		// Send output to WebSocket
		if err := ws.WriteMessage(websocket.BinaryMessage, data); err != nil {
			logrus.Errorf("WebSocket write error: %v", err)
			break
		}
	}

	logrus.Infof("WebSocket closed for session: %s", streamID)
}

// notifyAgentCreateTerminal sends a task to Agent to create terminal
func (h *WebSSHHandler) notifyAgentCreateTerminal(uuid, streamID string) error {
	// Queue terminal task for the Agent
	task := &proto.AgentTask{
		TaskId:   uuid + "-terminal-" + streamID,
		TaskType: "terminal",
		Params: map[string]string{
			"stream_id": streamID,
		},
	}

	if err := h.agentManager.QueueTask(uuid, task); err != nil {
		logrus.Errorf("Failed to queue terminal task: %v", err)
		return err
	}

	logrus.Infof("Queued terminal task for agent %s, session %s", uuid, streamID)
	return nil
}
