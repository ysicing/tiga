package docker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	basehandlers "github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/services/host"
	"github.com/ysicing/tiga/proto"
)

// ContainerLogsHandler handles Docker container logs API requests
type ContainerLogsHandler struct {
	dockerStreamManager *host.DockerStreamManager
	agentManager        *host.AgentManager
	db                  *gorm.DB
}

// NewContainerLogsHandler creates a new ContainerLogsHandler
func NewContainerLogsHandler(dockerStreamManager *host.DockerStreamManager, agentManager *host.AgentManager, db *gorm.DB) *ContainerLogsHandler {
	return &ContainerLogsHandler{
		dockerStreamManager: dockerStreamManager,
		agentManager:        agentManager,
		db:                  db,
	}
}

// GetContainerLogs godoc
// @Summary Get container logs (historical)
// @Description Get historical logs from a Docker container (non-streaming)
// @Tags docker-containers
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param container_id path string true "Container ID or name"
// @Param tail query string false "Number of lines to show from the end (default: 100)"
// @Param since query string false "Show logs since timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)"
// @Param timestamps query boolean false "Show timestamps (default: false)"
// @Success 200 {object} handlers.SuccessResponse{data=[]string}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/containers/{container_id}/logs [get]
// @Security BearerAuth
func (h *ContainerLogsHandler) GetContainerLogs(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	containerID := c.Param("container_id")
	if containerID == "" {
		basehandlers.RespondBadRequest(c, fmt.Errorf("container_id is required"))
		return
	}

	// Parse query parameters
	tail := c.DefaultQuery("tail", "100")
	since := c.Query("since")
	timestamps := c.DefaultQuery("timestamps", "false")

	// Create stream session for historical logs (follow=false)
	params := map[string]string{
		"tail":       tail,
		"follow":     "false",
		"timestamps": timestamps,
	}
	if since != "" {
		params["since"] = since
	}

	session, err := h.dockerStreamManager.CreateSession(instanceID, "get_logs", containerID, "", params)
	if err != nil {
		logrus.WithError(err).Error("Failed to create Docker stream session")
		basehandlers.RespondInternalError(c, err)
		return
	}
	defer h.dockerStreamManager.CloseSession(session.SessionID)

	// Trigger Agent to connect
	if err := h.triggerAgentConnection(session); err != nil {
		logrus.WithError(err).Error("Failed to trigger Agent connection")
		basehandlers.RespondInternalError(c, err)
		return
	}

	// Wait for Agent to be ready
	if err := session.WaitForReady(10 * time.Second); err != nil {
		logrus.WithError(err).Error("Agent failed to become ready")
		basehandlers.RespondInternalError(c, fmt.Errorf("agent not ready: %w", err))
		return
	}

	// Collect logs until stream closes
	var logs []string
	timeout := time.After(30 * time.Second) // 30 second timeout for historical logs

	for {
		select {
		case data, ok := <-session.DataChan:
			if !ok {
				// Channel closed, stream ended
				basehandlers.RespondSuccess(c, logs)
				return
			}
			logs = append(logs, string(data.Data))

		case streamErr, ok := <-session.ErrorChan:
			if ok {
				logrus.WithFields(logrus.Fields{
					"session_id": session.SessionID,
					"error":      streamErr.Error,
				}).Error("Stream error")
				basehandlers.RespondInternalError(c, fmt.Errorf("stream error: %s", streamErr.Error))
				return
			}

		case closeMsg, ok := <-session.CloseChan:
			if ok {
				logrus.WithFields(logrus.Fields{
					"session_id": session.SessionID,
					"reason":     closeMsg.Reason,
				}).Info("Stream closed by agent")
			}
			basehandlers.RespondSuccess(c, logs)
			return

		case <-timeout:
			logrus.Warn("Timeout waiting for logs")
			basehandlers.RespondInternalError(c, fmt.Errorf("timeout waiting for logs"))
			return
		}
	}
}

// GetContainerLogsStream godoc
// @Summary Get container logs (streaming)
// @Description Stream real-time logs from a Docker container using Server-Sent Events (SSE)
// @Tags docker-containers
// @Accept json
// @Produce text/event-stream
// @Param id path string true "Docker Instance ID (UUID)"
// @Param container_id path string true "Container ID or name"
// @Param tail query string false "Number of lines to show from the end (default: 100)"
// @Param since query string false "Show logs since timestamp or relative"
// @Param timestamps query boolean false "Show timestamps (default: false)"
// @Success 200 {string} string "SSE stream of container logs"
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/containers/{container_id}/logs/stream [get]
// @Security BearerAuth
func (h *ContainerLogsHandler) GetContainerLogsStream(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	containerID := c.Param("container_id")
	if containerID == "" {
		basehandlers.RespondBadRequest(c, fmt.Errorf("container_id is required"))
		return
	}

	// Parse query parameters
	tail := c.DefaultQuery("tail", "100")
	since := c.Query("since")
	timestamps := c.DefaultQuery("timestamps", "false")

	// Create stream session for real-time logs (follow=true)
	params := map[string]string{
		"tail":       tail,
		"follow":     "true",
		"timestamps": timestamps,
	}
	if since != "" {
		params["since"] = since
	}

	session, err := h.dockerStreamManager.CreateSession(instanceID, "get_logs", containerID, "", params)
	if err != nil {
		logrus.WithError(err).Error("Failed to create Docker stream session")
		basehandlers.RespondInternalError(c, err)
		return
	}
	defer h.dockerStreamManager.CloseSession(session.SessionID)

	// Trigger Agent to connect
	if err := h.triggerAgentConnection(session); err != nil {
		logrus.WithError(err).Error("Failed to trigger Agent connection")
		basehandlers.RespondInternalError(c, err)
		return
	}

	// Wait for Agent to be ready
	if err := session.WaitForReady(10 * time.Second); err != nil {
		logrus.WithError(err).Error("Agent failed to become ready")
		basehandlers.RespondInternalError(c, fmt.Errorf("agent not ready: %w", err))
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create flusher for immediate data sending
	flusher, ok := c.Writer.(interface{ Flush() })
	if !ok {
		logrus.Error("Streaming not supported")
		basehandlers.RespondInternalError(c, fmt.Errorf("streaming not supported"))
		return
	}

	logrus.WithFields(logrus.Fields{
		"instance_id":  instanceID,
		"container_id": containerID,
		"session_id":   session.SessionID,
		"tail":         tail,
		"timestamps":   timestamps,
	}).Info("Starting container logs stream")

	// Monitor client disconnect
	clientGone := c.Request.Context().Done()

	// Stream logs to client
	for {
		select {
		case <-clientGone:
			// Client disconnected
			logrus.WithFields(logrus.Fields{
				"instance_id":  instanceID,
				"container_id": containerID,
				"session_id":   session.SessionID,
			}).Info("Client disconnected from logs stream")
			return

		case data, ok := <-session.DataChan:
			if !ok {
				// Channel closed, stream ended
				logrus.WithFields(logrus.Fields{
					"instance_id":  instanceID,
					"container_id": containerID,
					"session_id":   session.SessionID,
				}).Info("Logs stream ended normally")
				return
			}

			// Send log data as SSE event
			logData := map[string]interface{}{
				"type": data.DataType,
				"data": string(data.Data),
			}
			logJSON, _ := json.Marshal(logData)
			fmt.Fprintf(c.Writer, "data: %s\n\n", logJSON)
			flusher.Flush()

		case streamErr, ok := <-session.ErrorChan:
			if ok {
				logrus.WithFields(logrus.Fields{
					"session_id": session.SessionID,
					"error":      streamErr.Error,
				}).Error("Stream error")

				// Send error event
				errorData := map[string]interface{}{
					"error":   streamErr.Error,
					"message": "Failed to receive logs",
				}
				errorJSON, _ := json.Marshal(errorData)
				fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", errorJSON)
				flusher.Flush()
				return
			}

		case closeMsg, ok := <-session.CloseChan:
			if ok {
				logrus.WithFields(logrus.Fields{
					"session_id": session.SessionID,
					"reason":     closeMsg.Reason,
				}).Info("Stream closed by agent")

				// Send close event
				closeData := map[string]interface{}{
					"reason":  closeMsg.Reason,
					"message": "Stream closed",
				}
				closeJSON, _ := json.Marshal(closeData)
				fmt.Fprintf(c.Writer, "event: close\ndata: %s\n\n", closeJSON)
				flusher.Flush()
			}
			return
		}
	}
}

// triggerAgentConnection sends a task to Agent to initiate DockerStream connection
func (h *ContainerLogsHandler) triggerAgentConnection(session *host.DockerStreamSession) error {
	// Get Docker instance to find associated agent
	var instance models.DockerInstance
	if err := h.db.Where("id = ?", session.InstanceID).First(&instance).Error; err != nil {
		return fmt.Errorf("failed to find Docker instance: %w", err)
	}

	// Get agent connection to get host UUID
	var agentConn models.AgentConnection
	if err := h.db.Where("id = ?", instance.AgentID).First(&agentConn).Error; err != nil {
		return fmt.Errorf("failed to find agent connection: %w", err)
	}

	// Get host node to get UUID
	var hostNode models.HostNode
	if err := h.db.Where("id = ?", agentConn.HostNodeID).First(&hostNode).Error; err != nil {
		return fmt.Errorf("failed to find host node: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"session_id":  session.SessionID,
		"instance_id": session.InstanceID,
		"agent_id":    instance.AgentID,
		"host_uuid":   hostNode.ID.String(),
	}).Info("Triggering Agent DockerStream connection")

	// Create docker_stream task
	task := &proto.AgentTask{
		TaskId:   session.SessionID,
		TaskType: "docker_stream",
		Params: map[string]string{
			"session_id":   session.SessionID,
			"operation":    session.Operation,
			"container_id": session.ContainerID,
			"image_name":   session.ImageName,
		},
	}

	// Add session params
	for k, v := range session.Params {
		task.Params[k] = v
	}

	// Queue task (non-blocking, no wait for result)
	return h.agentManager.QueueTask(hostNode.ID.String(), task)
}
