package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"

	"github.com/ysicing/tiga/internal/models"
	k8sservice "github.com/ysicing/tiga/internal/services/k8s"
	"github.com/ysicing/tiga/pkg/cluster"
	"github.com/ysicing/tiga/pkg/common"
	"github.com/ysicing/tiga/pkg/kube"
	"github.com/ysicing/tiga/pkg/rbac"
)

type TerminalHandler struct {
	recordingService *k8sservice.TerminalRecordingService
}

func NewTerminalHandler(recordingService *k8sservice.TerminalRecordingService) *TerminalHandler {
	return &TerminalHandler{
		recordingService: recordingService,
	}
}

// HandleTerminalWebSocket handles WebSocket connections for terminal sessions
func (h *TerminalHandler) HandleTerminalWebSocket(c *gin.Context) {
	// Get cluster info from context
	cs := c.MustGet("cluster").(*cluster.ClientSet)

	// Get path parameters
	namespace := c.Param("namespace")
	podName := c.Param("podName")
	container := c.Query("container")

	if namespace == "" || podName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "namespace and podName are required"})
		return
	}

	user := c.MustGet("user").(models.User)

	websocket.Handler(func(ws *websocket.Conn) {
		var recordingID uuid.UUID
		var recordingStopFunc func()

		defer func() {
			_ = ws.Close()
			// Stop recording if it was started
			if recordingStopFunc != nil {
				recordingStopFunc()
			}
		}()

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		if !rbac.CanAccess(user, "pods", "exec", cs.Name, namespace) {
			h.sendErrorMessage(
				ws,
				rbac.NoAccess(user.Key(), string(common.VerbExec), "pods", namespace, cs.Name),
			)
			return
		}

		// Start pod terminal recording (T020 integration)
		k8sSession, recorder, recordingModel, err := h.recordingService.StartPodTerminalRecording(
			ctx,
			user.ID,
			cs.Name,
			namespace,
			podName,
			container,
			80, // default width
			24, // default height
		)
		if err != nil {
			logrus.Errorf("Failed to start terminal recording: %v", err)
			h.sendErrorMessage(ws, fmt.Sprintf("Failed to start terminal recording: %v", err))
			return
		}

		recordingID = recordingModel.ID
		recordingStopFunc = func() {
			if err := h.recordingService.StopRecording(ctx, recordingID, "session_ended"); err != nil {
				logrus.Errorf("Failed to stop recording: %v", err)
			}
		}

		// Wrap the WebSocket connection with recording decorator
		wrappedConn := kube.NewRecordingWebSocketWrapper(ws, recorder)

		// Create terminal session with wrapped connection
		session := kube.NewTerminalSessionWithRecording(cs.K8sClient, wrappedConn, namespace, podName, container, k8sSession)
		defer session.Close()

		if err := session.Start(ctx, "exec"); err != nil {
			logrus.Errorf("Terminal session error: %v", err)
		}
	}).ServeHTTP(c.Writer, c.Request)
}

// sendErrorMessage sends an error message through WebSocket
func (h *TerminalHandler) sendErrorMessage(conn *websocket.Conn, message string) {
	msg := map[string]interface{}{
		"type": "error",
		"data": message,
	}
	if err := websocket.JSON.Send(conn, msg); err != nil {
		logrus.Errorf("Failed to send error message: %v", err)
	}
}
