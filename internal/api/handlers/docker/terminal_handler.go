package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	authservices "github.com/ysicing/tiga/internal/services/auth"
	dockerservices "github.com/ysicing/tiga/internal/services/docker"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
	"gorm.io/gorm"
)

// TerminalHandler handles Docker container terminal operations
type TerminalHandler struct {
	db                 *gorm.DB
	agentForwarder     *dockerservices.AgentForwarder
	instanceService    *dockerservices.DockerInstanceService
	jwtManager         *authservices.JWTManager
	recordingRepo      repository.TerminalRecordingRepositoryInterface
	recordingDir       string // Directory to store recording files

	// Session management
	sessions sync.Map // session_id -> *TerminalSession
}

// NewTerminalHandler creates a new TerminalHandler
func NewTerminalHandler(
	db *gorm.DB,
	agentForwarder *dockerservices.AgentForwarder,
	instanceService *dockerservices.DockerInstanceService,
	jwtManager *authservices.JWTManager,
	recordingRepo repository.TerminalRecordingRepositoryInterface,
) *TerminalHandler {
	// Default recording directory
	recordingDir := os.Getenv("TERMINAL_RECORDING_DIR")
	if recordingDir == "" {
		recordingDir = "./data/terminal-recordings"
	}

	// Ensure recording directory exists
	if err := os.MkdirAll(recordingDir, 0755); err != nil {
		logrus.WithError(err).Error("Failed to create recording directory")
	}

	handler := &TerminalHandler{
		db:              db,
		agentForwarder:  agentForwarder,
		instanceService: instanceService,
		jwtManager:      jwtManager,
		recordingRepo:   recordingRepo,
		recordingDir:    recordingDir,
	}

	// Cleanup expired sessions every 5 minutes
	go handler.cleanupExpiredSessions()

	return handler
}

// TerminalSession represents a terminal session
type TerminalSession struct {
	ID          uuid.UUID
	InstanceID  uuid.UUID
	ContainerID string
	Shell       string
	Rows        int
	Cols        int
	CreatedAt   time.Time
	ExpiresAt   time.Time
	UserID      uuid.UUID
	Username    string
	ClientIP    string

	// Recording support
	Recording       bool
	RecordingID     uuid.UUID
	RecordingBuffer *bytes.Buffer
	RecordingMutex  sync.Mutex
	StartTime       time.Time
}

// TerminalMessage represents WebSocket messages
type TerminalMessage struct {
	Type     string `json:"type"`                // "input", "resize", "ping", "output", "error", "pong", "exit"
	Data     string `json:"data,omitempty"`      // Terminal input/output
	Rows     int    `json:"rows,omitempty"`      // Terminal rows (for resize)
	Cols     int    `json:"cols,omitempty"`      // Terminal cols (for resize)
	Code     string `json:"code,omitempty"`      // Error code
	Message  string `json:"message,omitempty"`   // Error message
	ExitCode int32  `json:"exit_code,omitempty"` // Process exit code
}

// CreateTerminalSessionRequest represents the request to create a terminal session
type CreateTerminalSessionRequest struct {
	Shell string            `json:"shell,omitempty"` // Default: /bin/sh
	Rows  int               `json:"rows,omitempty"`  // Default: 30
	Cols  int               `json:"cols,omitempty"`  // Default: 120
	Env   map[string]string `json:"env,omitempty"`   // Environment variables
}

// CreateTerminalSession godoc
// @Summary Create Docker container terminal session
// @Description Create a WebSocket terminal session for a Docker container
// @Tags docker-terminal
// @Accept json
// @Produce json
// @Param instance_id path string true "Docker Instance ID"
// @Param container_id path string true "Container ID"
// @Param body body CreateTerminalSessionRequest true "Terminal options"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /api/v1/docker/instances/{instance_id}/containers/{container_id}/terminal [post]
func (h *TerminalHandler) CreateTerminalSession(c *gin.Context) {
	instanceIDStr := c.Param("instance_id")
	containerID := c.Param("container_id")

	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_INSTANCE_ID",
				"message": "Invalid Docker instance ID",
			},
		})
		return
	}

	// Parse request body
	var req CreateTerminalSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use defaults if no body provided
		req = CreateTerminalSessionRequest{}
	}

	// Set defaults
	if req.Shell == "" {
		req.Shell = "/bin/sh"
	}
	if req.Rows == 0 {
		req.Rows = 30
	}
	if req.Cols == 0 {
		req.Cols = 120
	}

	// Get Docker instance
	instance, err := h.instanceService.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INSTANCE_NOT_FOUND",
				"message": "Docker instance not found",
			},
		})
		return
	}

	// Check if instance is online
	if instance.HealthStatus != "online" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "SERVICE_UNAVAILABLE",
				"message": "Docker实例离线",
			},
		})
		return
	}

	// Get container details to verify it exists and is running
	container, err := h.agentForwarder.GetContainer(instance.AgentID, &pb.GetContainerRequest{
		ContainerId: containerID,
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CONTAINER_NOT_FOUND",
				"message": "容器不存在",
			},
		})
		return
	}

	// Check if container is running
	if container.Container == nil || container.Container.State == nil || !container.Container.State.Running {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "容器未在运行",
			},
		})
		return
	}

	// Get user info from JWT context
	userIDVal, _ := c.Get("user_id")
	usernameVal, _ := c.Get("username")

	userID, _ := userIDVal.(uuid.UUID)
	username, _ := usernameVal.(string)
	clientIP := c.ClientIP()

	// Create session with recording enabled by default
	now := time.Now()
	recordingID := uuid.New()
	session := &TerminalSession{
		ID:              uuid.New(),
		InstanceID:      instanceID,
		ContainerID:     containerID,
		Shell:           req.Shell,
		Rows:            req.Rows,
		Cols:            req.Cols,
		CreatedAt:       now,
		ExpiresAt:       now.Add(30 * time.Minute), // 30 minutes TTL
		UserID:          userID,
		Username:        username,
		ClientIP:        clientIP,
		Recording:       true, // Enable recording by default
		RecordingID:     recordingID,
		RecordingBuffer: new(bytes.Buffer),
		StartTime:       now,
	}

	// Create recording metadata in database
	recording := &models.TerminalRecording{
		BaseModel:   models.BaseModel{ID: recordingID},
		SessionID:   session.ID,
		InstanceID:  instanceID,
		ContainerID: containerID,
		UserID:      userID,
		Username:    username,
		StartedAt:   now,
		Rows:        req.Rows,
		Cols:        req.Cols,
		Shell:       req.Shell,
		ClientIP:    clientIP,
		Format:      "asciinema",
		StoragePath: "", // Will be set when session ends
	}

	if err := h.recordingRepo.Create(c.Request.Context(), recording); err != nil {
		logrus.WithError(err).Error("Failed to create recording metadata")
		// Continue anyway - recording is optional
	}

	// Write asciinema header to recording buffer
	header := models.AsciinemaHeader{
		Version:   2,
		Width:     req.Cols,
		Height:    req.Rows,
		Timestamp: now.Unix(),
		Title:     fmt.Sprintf("Docker Terminal - %s", containerID),
		Env: map[string]string{
			"SHELL": req.Shell,
			"TERM":  "xterm-256color",
		},
	}
	headerJSON, _ := json.Marshal(header)
	session.RecordingBuffer.Write(headerJSON)
	session.RecordingBuffer.WriteString("\n")

	// Store session
	h.sessions.Store(session.ID.String(), session)

	// Build WebSocket URL
	scheme := "ws"
	if c.Request.TLS != nil {
		scheme = "wss"
	}
	wsURL := fmt.Sprintf("%s://%s/api/v1/docker/terminal/%s", scheme, c.Request.Host, session.ID)

	logrus.WithFields(logrus.Fields{
		"session_id":   session.ID,
		"instance_id":  instanceID,
		"container_id": containerID,
		"user":         username,
	}).Info("Docker terminal session created")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"session_id": session.ID.String(),
			"ws_url":     wsURL,
			"expires_at": session.ExpiresAt.Format(time.RFC3339),
		},
	})
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (adjust in production)
	},
}

// HandleWebSocketTerminal godoc
// @Summary WebSocket Docker container terminal
// @Description Establish WebSocket connection for Docker container terminal
// @Tags docker-terminal
// @Param session_id path string true "Session ID"
// @Param token query string true "JWT Token"
// @Router /api/v1/docker/terminal/{session_id} [get]
func (h *TerminalHandler) HandleWebSocketTerminal(c *gin.Context) {
	sessionIDStr := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	// Verify JWT token from query parameter (WebSocket doesn't support headers)
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		return
	}

	// Validate JWT token
	claims, err := h.jwtManager.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Load session
	sessionVal, ok := h.sessions.Load(sessionID.String())
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found or expired"})
		return
	}
	session := sessionVal.(*TerminalSession)

	// Check if session expired
	if time.Now().After(session.ExpiresAt) {
		h.sessions.Delete(sessionID.String())
		c.JSON(http.StatusGone, gin.H{"error": "Session expired"})
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to upgrade to WebSocket")
		return
	}
	defer conn.Close()

	logrus.WithFields(logrus.Fields{
		"session_id":   session.ID,
		"user":         claims.Username,
		"container_id": session.ContainerID,
	}).Info("WebSocket terminal connection established")

	// Get Docker instance
	instance, err := h.instanceService.GetByID(c.Request.Context(), session.InstanceID)
	if err != nil {
		h.sendError(conn, "INSTANCE_NOT_FOUND", "Docker instance not found")
		return
	}

	// Create gRPC stream context
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Start bidirectional gRPC stream with Agent
	stream, err := h.agentForwarder.ExecContainer(instance.AgentID)
	if err != nil {
		h.sendError(conn, "EXEC_FAILED", fmt.Sprintf("Failed to start exec session: %v", err))
		return
	}

	// Send ExecStart message to Agent
	startReq := &pb.ExecRequest{
		Request: &pb.ExecRequest_Start{
			Start: &pb.ExecStart{
				ContainerId:  session.ContainerID,
				Cmd:          []string{session.Shell},
				Tty:          true,
				AttachStdin:  true,
				AttachStdout: true,
				AttachStderr: true,
			},
		},
	}

	if err := stream.Send(startReq); err != nil {
		h.sendError(conn, "EXEC_FAILED", fmt.Sprintf("Failed to send exec start: %v", err))
		return
	}

	// Send welcome message
	h.sendOutput(conn, session, "Connected to container terminal\r\n")

	// Goroutine: Read from Agent gRPC stream and write to WebSocket
	go func() {
		defer func() {
			// Finalize recording when stream ends
			h.finalizeRecording(context.Background(), session)
		}()

		for {
			resp, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					h.sendExit(conn, 0)
				} else {
					h.sendError(conn, "AGENT_DISCONNECTED", fmt.Sprintf("Agent stream error: %v", err))
				}
				cancel()
				return
			}

			switch msg := resp.Response.(type) {
			case *pb.ExecResponse_Output:
				h.sendOutput(conn, session, string(msg.Output.Data))
			case *pb.ExecResponse_Exit:
				h.sendExit(conn, msg.Exit.ExitCode)
				cancel()
				return
			case *pb.ExecResponse_Error:
				h.sendError(conn, "EXEC_ERROR", msg.Error.Message)
				cancel()
				return
			}
		}
	}()

	// Heartbeat tracker
	lastHeartbeat := time.Now()
	heartbeatMu := sync.Mutex{}

	// Goroutine: Heartbeat monitor
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				heartbeatMu.Lock()
				if time.Since(lastHeartbeat) > 2*time.Minute {
					heartbeatMu.Unlock()
					h.sendError(conn, "SESSION_TIMEOUT", "Session timeout due to inactivity")
					cancel()
					return
				}
				heartbeatMu.Unlock()
			}
		}
	}()

	// Main loop: Read from WebSocket and forward to Agent gRPC
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var msg TerminalMessage
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logrus.WithError(err).Warn("WebSocket unexpected close")
				}
				cancel()
				return
			}

			switch msg.Type {
			case "input":
				// Record input frame
				h.recordFrame(session, "i", []byte(msg.Data))

				// Forward input to Agent
				inputReq := &pb.ExecRequest{
					Request: &pb.ExecRequest_Input{
						Input: &pb.ExecInput{
							Data: []byte(msg.Data),
						},
					},
				}
				if err := stream.Send(inputReq); err != nil {
					logrus.WithError(err).Error("Failed to send input to Agent")
					cancel()
					return
				}

				// Update heartbeat
				heartbeatMu.Lock()
				lastHeartbeat = time.Now()
				heartbeatMu.Unlock()

			case "resize":
				// Forward resize to Agent
				resizeReq := &pb.ExecRequest{
					Request: &pb.ExecRequest_Resize{
						Resize: &pb.ExecResize{
							Width:  uint32(msg.Cols),
							Height: uint32(msg.Rows),
						},
					},
				}
				if err := stream.Send(resizeReq); err != nil {
					logrus.WithError(err).Warn("Failed to send resize to Agent")
				}

			case "ping":
				// Respond with pong
				h.sendPong(conn)

				// Update heartbeat
				heartbeatMu.Lock()
				lastHeartbeat = time.Now()
				heartbeatMu.Unlock()
			}
		}
	}
}

// sendOutput sends terminal output to WebSocket client and records it
func (h *TerminalHandler) sendOutput(conn *websocket.Conn, session *TerminalSession, data string) {
	msg := TerminalMessage{
		Type: "output",
		Data: data,
	}
	if err := conn.WriteJSON(msg); err != nil {
		logrus.WithError(err).Warn("Failed to send output to WebSocket")
	}

	// Record output frame
	h.recordFrame(session, "o", []byte(data))
}

// sendError sends error message to WebSocket client
func (h *TerminalHandler) sendError(conn *websocket.Conn, code, message string) {
	msg := TerminalMessage{
		Type:    "error",
		Code:    code,
		Message: message,
	}
	conn.WriteJSON(msg)

	// Close connection after sending error
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

// sendExit sends exit message to WebSocket client
func (h *TerminalHandler) sendExit(conn *websocket.Conn, exitCode int32) {
	msg := TerminalMessage{
		Type:     "exit",
		ExitCode: exitCode,
	}
	conn.WriteJSON(msg)

	// Close connection after sending exit
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

// sendPong sends pong response to WebSocket client
func (h *TerminalHandler) sendPong(conn *websocket.Conn) {
	msg := TerminalMessage{
		Type: "pong",
	}
	if err := conn.WriteJSON(msg); err != nil {
		logrus.WithError(err).Warn("Failed to send pong to WebSocket")
	}
}

// cleanupExpiredSessions removes expired sessions every 5 minutes
func (h *TerminalHandler) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		deletedCount := 0

		h.sessions.Range(func(key, value interface{}) bool {
			session := value.(*TerminalSession)
			if now.After(session.ExpiresAt) {
				h.sessions.Delete(key)
				deletedCount++
			}
			return true
		})

		if deletedCount > 0 {
			logrus.WithField("count", deletedCount).Debug("Cleaned up expired terminal sessions")
		}
	}
}

// AuditTerminalAccess records terminal access in audit log
func (h *TerminalHandler) AuditTerminalAccess(ctx context.Context, session *TerminalSession, action string) {
	changes := map[string]interface{}{
		"session_id":   session.ID.String(),
		"instance_id":  session.InstanceID.String(),
		"container_id": session.ContainerID,
		"shell":        session.Shell,
		"action":       action,
	}

	auditLog := &models.AuditLog{
		UserID:       &session.UserID,
		Username:     session.Username,
		Action:       models.DockerActionContainerExec,
		ResourceType: "docker_terminal",
		ResourceID:   &session.ID,
		Description:  fmt.Sprintf("Docker terminal access: %s", action),
		Changes:      changes,
		Status:       "success",
	}

	if err := h.db.WithContext(ctx).Create(auditLog).Error; err != nil {
		logrus.WithError(err).Warn("Failed to create terminal access audit log")
	}
}

// recordFrame records a terminal frame to the recording buffer (asciinema format)
func (h *TerminalHandler) recordFrame(session *TerminalSession, frameType string, data []byte) {
	if !session.Recording || session.RecordingBuffer == nil {
		return
	}

	session.RecordingMutex.Lock()
	defer session.RecordingMutex.Unlock()

	// Calculate relative timestamp in seconds
	timestamp := time.Since(session.StartTime).Seconds()

	// Create asciinema frame: [timestamp, "o" or "i", "data"]
	frame := models.RecordingFrame{
		Timestamp: timestamp,
		Type:      frameType, // "o" for output, "i" for input
		Data:      string(data),
	}

	frameJSON, err := json.Marshal([]interface{}{frame.Timestamp, frame.Type, frame.Data})
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal recording frame")
		return
	}

	session.RecordingBuffer.Write(frameJSON)
	session.RecordingBuffer.WriteString("\n")
}

// finalizeRecording saves the recording to disk and updates database
func (h *TerminalHandler) finalizeRecording(ctx context.Context, session *TerminalSession) {
	if !session.Recording || session.RecordingBuffer == nil {
		return
	}

	// Get recording metadata from database
	recording, err := h.recordingRepo.GetBySessionID(ctx, session.ID)
	if err != nil {
		logrus.WithError(err).Error("Failed to get recording metadata")
		return
	}

	// Generate storage path
	dateDir := time.Now().Format("2006-01-02")
	recordingFilename := fmt.Sprintf("%s.cast", session.RecordingID)
	storagePath := filepath.Join(h.recordingDir, dateDir, recordingFilename)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(storagePath), 0755); err != nil {
		logrus.WithError(err).Error("Failed to create recording directory")
		return
	}

	// Write recording to file
	if err := os.WriteFile(storagePath, session.RecordingBuffer.Bytes(), 0644); err != nil {
		logrus.WithError(err).Error("Failed to write recording file")
		return
	}

	// Update recording metadata
	now := time.Now()
	recording.EndedAt = &now
	recording.Duration = int(now.Sub(session.StartTime).Seconds())
	recording.FileSize = int64(session.RecordingBuffer.Len())
	recording.StoragePath = storagePath

	if err := h.recordingRepo.Update(ctx, recording); err != nil {
		logrus.WithError(err).Error("Failed to update recording metadata")
		return
	}

	logrus.WithFields(logrus.Fields{
		"recording_id": session.RecordingID,
		"session_id":   session.ID,
		"duration":     recording.Duration,
		"file_size":    recording.FileSize,
		"path":         storagePath,
	}).Info("Terminal recording saved successfully")
}
