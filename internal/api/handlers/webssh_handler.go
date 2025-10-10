package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/api/middleware"
	"github.com/ysicing/tiga/internal/models"
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
	sessionMgr   *webssh.SessionManager
	terminalMgr  *host.TerminalManager
	agentManager *host.AgentManager
	db           *gorm.DB
}

// NewWebSSHHandler creates a new WebSSH handler
func NewWebSSHHandler(sessionMgr *webssh.SessionManager, terminalMgr *host.TerminalManager, agentMgr *host.AgentManager, db *gorm.DB) *WebSSHHandler {
	return &WebSSHHandler{
		sessionMgr:   sessionMgr,
		terminalMgr:  terminalMgr,
		agentManager: agentMgr,
		db:           db,
	}
}

// CreateSession creates a WebSSH session
func (h *WebSSHHandler) CreateSession(c *gin.Context) {
	var req struct {
		HostID           string `json:"host_id" binding:"required"` // UUID string
		Width            int    `json:"width"`
		Height           int    `json:"height"`
		RecordingEnabled *bool  `json:"recording_enabled"` // Optional, defaults to true
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": "Invalid request"})
		return
	}

	// Default recording enabled to true if not specified
	recordingEnabled := true
	if req.RecordingEnabled != nil {
		recordingEnabled = *req.RecordingEnabled
	}

	// Resolve authenticated user
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 40100, "message": "User not authenticated"})
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
	logrus.Infof("[WebSSH] Creating session: %s for host: %s", streamID, hostUUID)

	// Create terminal session
	h.terminalMgr.CreateSession(streamID, hostUUID, conn.UUID)
	logrus.Debugf("[WebSSH] Terminal session created: %s", streamID)

	// Create webssh session for recording
	clientIP := c.ClientIP()
	wsSession, err := h.sessionMgr.CreateSession(c.Request.Context(), streamID, userID, hostUUID, req.Width, req.Height, clientIP, recordingEnabled)
	if err != nil {
		logrus.Errorf("Failed to create webssh session: %v", err)
		// Rollback terminal session if we cannot persist metadata
		h.terminalMgr.CloseSession(streamID)
		// Don't fail the terminal creation if session recording fails
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to create session"})
		return
	}

	// Notify Agent to create terminal via gRPC Task
	// The Agent will connect to IOStream with the streamID
	if err := h.notifyAgentCreateTerminal(conn.UUID, streamID); err != nil {
		h.terminalMgr.CloseSession(streamID)
		_ = h.sessionMgr.CloseSession(c.Request.Context(), wsSession.SessionID, "agent setup failed")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to notify agent"})
		return
	}

	// Record activity log
	activityLog := &models.HostActivityLog{
		HostNodeID:  hostUUID,
		UserID:      &userID,
		Action:      models.ActivityTerminalCreated,
		ActionType:  models.ActivityTypeTerminal,
		Description: fmt.Sprintf("WebSSH terminal session created (Session ID: %s)", streamID),
		Metadata:    fmt.Sprintf(`{"session_id":"%s","ws_session_id":"%s"}`, streamID, wsSession.SessionID),
		ClientIP:    clientIP,
		UserAgent:   c.Request.UserAgent(),
	}
	if err := h.db.Create(activityLog).Error; err != nil {
		logrus.Warnf("Failed to record terminal creation activity: %v", err)
		// Don't fail the request if activity logging fails
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

	// Load session metadata
	wsSession, err := h.sessionMgr.GetSession(streamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 40404, "message": "Session not found"})
		return
	}

	// Get terminal session
	session, exists := h.terminalMgr.GetSession(streamID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"code": 40404, "message": "Terminal session not found"})
		return
	}

	// Upgrade to WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("WebSocket upgrade failed: %v", err)
		return
	}
	defer ws.Close()
	defer func() {
		// Close backend resources when the websocket terminates
		if err := h.terminalMgr.CloseSession(streamID); err != nil {
			logrus.Debugf("terminal session close error: %v", err)
		}
		if err := h.sessionMgr.CloseSession(c.Request.Context(), wsSession.SessionID, "client disconnected"); err != nil {
			logrus.Debugf("session close error: %v", err)
		}

		// Record terminal closed activity
		activityLog := &models.HostActivityLog{
			HostNodeID:  wsSession.HostNodeID,
			UserID:      &wsSession.UserID,
			Action:      models.ActivityTerminalClosed,
			ActionType:  models.ActivityTypeTerminal,
			Description: fmt.Sprintf("WebSSH terminal session closed (Session ID: %s)", streamID),
			Metadata:    fmt.Sprintf(`{"session_id":"%s","ws_session_id":"%s","status":"%s"}`, streamID, wsSession.SessionID, wsSession.Status),
			ClientIP:    wsSession.ClientIP,
		}
		if err := h.db.Create(activityLog).Error; err != nil {
			logrus.Warnf("Failed to record terminal closure activity: %v", err)
		}
	}()

	logrus.Infof("WebSocket connected for session: %s", streamID)

	// Send connected message using new protocol
	connMsg, _ := webssh.NewMessage(webssh.MessageTypeConnected, &webssh.ConnectedMessage{
		SessionID: streamID,
		HostName:  "", // TODO: hydrate with actual host name once repository is available
		HostID:    wsSession.HostNodeID.String(),
		Cols:      wsSession.Cols,
		Rows:      wsSession.Rows,
	})
	if msgBytes, err := json.Marshal(connMsg); err == nil {
		ws.WriteMessage(websocket.TextMessage, msgBytes)
	}

	// Set WebSocket timeouts
	ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker using new protocol
	done := make(chan struct{})
	defer close(done)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				h.sessionMgr.UpdateActivity(wsSession.SessionID)
				pingMsg, _ := webssh.NewMessage(webssh.MessageTypePing, nil)
				if msgBytes, err := json.Marshal(pingMsg); err == nil {
					if err := ws.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
						return
					}
				}
			case <-done:
				return
			}
		}
	}()

	// Error handler helper
	sendError := func(code string, message string) {
		errMsg, _ := webssh.NewMessage(webssh.MessageTypeError, &webssh.ErrorMessage{
			Code:    code,
			Message: message,
		})
		if msgBytes, err := json.Marshal(errMsg); err == nil {
			ws.WriteMessage(websocket.TextMessage, msgBytes)
		}
	}

	// Read from WebSocket and send to Agent
	go func() {
		defer session.SendToAgent([]byte{0xff}) // Signal end
		for {
			messageType, data, err := ws.ReadMessage()
			if err != nil {
				logrus.Errorf("WebSocket read error: %v", err)
				return
			}
			h.sessionMgr.UpdateActivity(wsSession.SessionID)

			if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
				// Parse message using our new protocol
				msg, err := webssh.ParseMessage(data)
				if err != nil {
					sendError(webssh.ErrCodeInvalidInput, "Invalid message format")
					continue
				}

				switch msg.Type {
				case webssh.MessageTypeInput:
					inputMsg, err := msg.GetInputMessage()
					if err != nil {
						sendError(webssh.ErrCodeInvalidInput, "Invalid input message")
						continue
					}
					// Decode base64 input for binary safety
					inputBytes, err := base64.StdEncoding.DecodeString(inputMsg.Input)
					if err != nil {
						sendError(webssh.ErrCodeInvalidInput, "Invalid base64 input")
						continue
					}
					// Record input
					h.sessionMgr.RecordInput(wsSession.SessionID, inputBytes)
					session.SendToAgent(append([]byte{0x00}, inputBytes...))

				case webssh.MessageTypeResize:
					resizeMsg, err := msg.GetResizeMessage()
					if err != nil {
						sendError(webssh.ErrCodeInvalidInput, "Invalid resize message")
						continue
					}
					// Update recorder size
					h.sessionMgr.ResizeRecorder(wsSession.SessionID, resizeMsg.Cols, resizeMsg.Rows)
					resizeData, _ := json.Marshal(map[string]int{"cols": resizeMsg.Cols, "rows": resizeMsg.Rows})
					session.SendToAgent(append([]byte{0x01}, resizeData...))

				case webssh.MessageTypeCommand:
					cmdMsg, err := msg.GetCommandMessage()
					if err != nil {
						sendError(webssh.ErrCodeInvalidInput, "Invalid command message")
						continue
					}
					if cmdMsg.Command == "close" {
						return
					}

				case webssh.MessageTypePing:
					// Send pong response
					pongMsg, _ := webssh.NewMessage(webssh.MessageTypePong, nil)
					if msgBytes, err := json.Marshal(pongMsg); err == nil {
						ws.WriteMessage(websocket.TextMessage, msgBytes)
					}
				}
			}
		}
	}()

	// Read from Agent and send to WebSocket
	for {
		data, err := session.ReceiveFromAgent()
		if err != nil {
			// Get error details for better diagnostics
			errorMessage := "Connection to agent lost"

			if lastErr := session.GetLastError(); lastErr != nil {
				logrus.Errorf("Agent connection error (detailed): %v", lastErr)

				// Check if error is potentially recoverable
				if host.IsRecoverableError(lastErr) {
					errorMessage = "Connection interrupted, please try reconnecting"
					logrus.Warnf("Recoverable error detected for session %s", wsSession.SessionID)
				}
			}

			logrus.Errorf("Failed to receive from agent: %v", err)
			sendError(webssh.ErrCodeConnectionClosed, errorMessage)
			break
		}

		// Record output
		h.sessionMgr.RecordOutput(wsSession.SessionID, data)

		// Encode output as base64 for binary safety
		h.sessionMgr.UpdateActivity(wsSession.SessionID)
		outputMsg, _ := webssh.NewMessage(webssh.MessageTypeOutput, &webssh.OutputMessage{
			Output: base64.StdEncoding.EncodeToString(data),
		})
		if msgBytes, err := json.Marshal(outputMsg); err == nil {
			if err := ws.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
				logrus.Errorf("WebSocket write error: %v", err)
				break
			}
		}
	}

	// Send disconnected message
	disconnMsg, _ := webssh.NewMessage(webssh.MessageTypeDisconnected, nil)
	if msgBytes, err := json.Marshal(disconnMsg); err == nil {
		ws.WriteMessage(websocket.TextMessage, msgBytes)
	}

	logrus.Infof("WebSocket closed for session: %s", streamID)
}

// ListAllSessions lists all WebSSH sessions (historical and active)
func (h *WebSSHHandler) ListAllSessions(c *gin.Context) {
	var sessions []models.WebSSHSession

	// Parse query parameters for pagination and filtering
	page := 1
	pageSize := 20
	status := c.Query("status")        // active or closed
	hostID := c.Query("host_id")       // filter by host
	userID := c.Query("user_id")       // filter by user
	startDate := c.Query("start_date") // filter by start date

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	// Build query
	query := h.db.Model(&models.WebSSHSession{}).
		Preload("HostNode").
		Preload("User")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if hostID != "" {
		query = query.Where("host_node_id = ?", hostID)
	}
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if startDate != "" {
		query = query.Where("start_time >= ?", startDate)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get paginated results
	offset := (page - 1) * pageSize
	if err := query.Order("start_time DESC").Limit(pageSize).Offset(offset).Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": "Failed to fetch sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"sessions":    sessions,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetSessionDetail gets details of a specific session
func (h *WebSSHHandler) GetSessionDetail(c *gin.Context) {
	sessionID := c.Param("session_id")

	var session models.WebSSHSession
	if err := h.db.Preload("HostNode").Preload("User").
		Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 40404, "message": "Session not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    session,
	})
}

// GetRecording returns the recording file for playback
func (h *WebSSHHandler) GetRecording(c *gin.Context) {
	sessionID := c.Param("session_id")

	// Get session from database
	var session models.WebSSHSession
	if err := h.db.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 40404, "message": "Session not found"})
		return
	}

	// Check if recording exists
	if !session.RecordingEnabled || session.RecordingPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"code": 40404, "message": "Recording not available for this session"})
		return
	}

	// Serve the recording file (asciicast format is NDJSON, not JSON)
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s.cast", sessionID))
	c.File(session.RecordingPath)
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
