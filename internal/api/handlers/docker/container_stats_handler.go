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

// ContainerStatsHandler handles Docker container stats API requests
type ContainerStatsHandler struct {
	dockerStreamManager *host.DockerStreamManager
	agentManager        *host.AgentManager
	db                  *gorm.DB
}

// NewContainerStatsHandler creates a new ContainerStatsHandler
func NewContainerStatsHandler(dockerStreamManager *host.DockerStreamManager, agentManager *host.AgentManager, db *gorm.DB) *ContainerStatsHandler {
	return &ContainerStatsHandler{
		dockerStreamManager: dockerStreamManager,
		agentManager:        agentManager,
		db:                  db,
	}
}

// GetContainerStats godoc
// @Summary Get container stats (single query)
// @Description Get current resource usage statistics for a Docker container
// @Tags docker-containers
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param container_id path string true "Container ID or name"
// @Success 200 {object} handlers.SuccessResponse
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/containers/{container_id}/stats [get]
// @Security BearerAuth
func (h *ContainerStatsHandler) GetContainerStats(c *gin.Context) {
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

	// Create stream session for single stats query (stream=false)
	params := map[string]string{
		"stream": "false",
	}

	session, err := h.dockerStreamManager.CreateSession(instanceID, "get_stats", containerID, "", params)
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

	// Wait for first stats message
	timeout := time.After(10 * time.Second)
	select {
	case data, ok := <-session.DataChan:
		if !ok {
			basehandlers.RespondInternalError(c, fmt.Errorf("stats channel closed"))
			return
		}
		// Parse JSON stats
		var stats interface{}
		if err := json.Unmarshal(data.Data, &stats); err != nil {
			logrus.WithError(err).Error("Failed to parse stats JSON")
			basehandlers.RespondInternalError(c, err)
			return
		}
		basehandlers.RespondSuccess(c, stats)
		return

	case streamErr, ok := <-session.ErrorChan:
		if ok {
			logrus.WithFields(logrus.Fields{
				"session_id": session.SessionID,
				"error":      streamErr.Error,
			}).Error("Stream error")
			basehandlers.RespondInternalError(c, fmt.Errorf("stream error: %s", streamErr.Error))
			return
		}

	case <-timeout:
		logrus.Warn("Timeout waiting for stats")
		basehandlers.RespondInternalError(c, fmt.Errorf("timeout waiting for stats"))
		return
	}
}

// GetContainerStatsStream godoc
// @Summary Get container stats (streaming)
// @Description Stream real-time resource usage statistics for a Docker container using Server-Sent Events (SSE)
// @Tags docker-containers
// @Accept json
// @Produce text/event-stream
// @Param id path string true "Docker Instance ID (UUID)"
// @Param container_id path string true "Container ID or name"
// @Success 200 {string} string "SSE stream of container stats"
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/containers/{container_id}/stats/stream [get]
// @Security BearerAuth
func (h *ContainerStatsHandler) GetContainerStatsStream(c *gin.Context) {
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

	// Create stream session for continuous stats (stream=true)
	params := map[string]string{
		"stream": "true",
	}

	session, err := h.dockerStreamManager.CreateSession(instanceID, "get_stats", containerID, "", params)
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
	}).Info("Starting container stats stream")

	// Monitor client disconnect
	clientGone := c.Request.Context().Done()

	// Stream stats to client
	for {
		select {
		case <-clientGone:
			// Client disconnected
			logrus.WithFields(logrus.Fields{
				"instance_id":  instanceID,
				"container_id": containerID,
				"session_id":   session.SessionID,
			}).Info("Client disconnected from stats stream")
			return

		case data, ok := <-session.DataChan:
			if !ok {
				// Channel closed, stream ended
				logrus.WithFields(logrus.Fields{
					"instance_id":  instanceID,
					"container_id": containerID,
					"session_id":   session.SessionID,
				}).Info("Stats stream ended normally")
				return
			}

			// Send stats data as SSE event
			statsData := map[string]interface{}{
				"type": data.DataType,
				"data": json.RawMessage(data.Data), // Already JSON from Docker API
			}
			statsJSON, _ := json.Marshal(statsData)
			fmt.Fprintf(c.Writer, "data: %s\n\n", statsJSON)
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
					"message": "Failed to receive stats",
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
func (h *ContainerStatsHandler) triggerAgentConnection(session *host.DockerStreamSession) error {
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
