package docker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/services/host"

	basehandlers "github.com/ysicing/tiga/internal/api/handlers"
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

	// Get Docker instance to find associated agent
	var instance models.DockerInstance
	if err := h.db.Where("id = ?", instanceID).First(&instance).Error; err != nil {
		basehandlers.RespondInternalError(c, fmt.Errorf("failed to find Docker instance: %w", err))
		return
	}

	// Create stream session for single stats query (stream=false)
	params := map[string]string{
		"stream": "false",
	}

	sessionInterface, err := h.dockerStreamManager.CreateSession(instanceID, instance.AgentID.String(), "get_stats", containerID, "", params)
	if err != nil {
		logrus.WithError(err).Error("Failed to create Docker stream session")
		basehandlers.RespondInternalError(c, err)
		return
	}

	// Type assert to *host.DockerStreamSession
	session, ok := sessionInterface.(*host.DockerStreamSession)
	if !ok {
		logrus.Error("Failed to cast session to *host.DockerStreamSession")
		basehandlers.RespondInternalError(c, fmt.Errorf("internal error: invalid session type"))
		return
	}
	defer h.dockerStreamManager.CloseSession(session.SessionID)

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

	// Get Docker instance to find associated agent
	var instance models.DockerInstance
	if err := h.db.Where("id = ?", instanceID).First(&instance).Error; err != nil {
		basehandlers.RespondInternalError(c, fmt.Errorf("failed to find Docker instance: %w", err))
		return
	}

	// Create stream session for continuous stats (stream=true)
	params := map[string]string{
		"stream": "true",
	}

	sessionInterface, err := h.dockerStreamManager.CreateSession(instanceID, instance.AgentID.String(), "get_stats", containerID, "", params)
	if err != nil {
		logrus.WithError(err).Error("Failed to create Docker stream session")
		basehandlers.RespondInternalError(c, err)
		return
	}

	// Type assert to *host.DockerStreamSession
	session, ok := sessionInterface.(*host.DockerStreamSession)
	if !ok {
		logrus.Error("Failed to cast session to *host.DockerStreamSession")
		basehandlers.RespondInternalError(c, fmt.Errorf("internal error: invalid session type"))
		return
	}
	defer h.dockerStreamManager.CloseSession(session.SessionID)

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
