package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	basehandlers "github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/services/docker"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// ContainerLogsHandler handles Docker container logs API requests
type ContainerLogsHandler struct {
	agentForwarder *docker.AgentForwarder
}

// NewContainerLogsHandler creates a new ContainerLogsHandler
func NewContainerLogsHandler(agentForwarder *docker.AgentForwarder) *ContainerLogsHandler {
	return &ContainerLogsHandler{
		agentForwarder: agentForwarder,
	}
}

// GetContainerLogs godoc
// @Summary Get container logs (historical)
// @Description Get historical logs from a Docker container
// @Tags docker-containers
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param container_id path string true "Container ID or name"
// @Param tail query int false "Number of lines to show from the end (default: 100)"
// @Param since query string false "Show logs since timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)"
// @Param timestamps query boolean false "Show timestamps (default: false)"
// @Param stdout query boolean false "Show stdout (default: true)"
// @Param stderr query boolean false "Show stderr (default: true)"
// @Success 200 {object} handlers.SuccessResponse{data=[]object}
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
	tailStr := c.DefaultQuery("tail", "100")
	sinceStr := c.Query("since")
	timestamps := c.DefaultQuery("timestamps", "false") == "true"
	stdout := c.DefaultQuery("stdout", "true") == "true"
	stderr := c.DefaultQuery("stderr", "true") == "true"

	// Parse since parameter (Unix timestamp or relative time)
	var since int64
	if sinceStr != "" {
		// Try to parse as Unix timestamp
		if ts, err := strconv.ParseInt(sinceStr, 10, 64); err == nil {
			since = ts
		}
		// TODO: Support relative time like "42m" (requires time parsing)
	}

	// Create logs request (follow=false for historical logs)
	req := &pb.GetContainerLogsRequest{
		ContainerId: containerID,
		Follow:      false,
		Tail:        tailStr,
		Since:       since,
		Timestamps:  timestamps,
		Stdout:      stdout,
		Stderr:      stderr,
	}

	// Get logs stream from agent
	stream, err := h.agentForwarder.GetContainerLogs(instanceID, req)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id":  instanceID,
			"container_id": containerID,
		}).Error("Failed to get container logs stream")
		basehandlers.RespondInternalError(c, err)
		return
	}

	// Collect all log entries
	var logs []interface{}
	for {
		logEntry, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				// Stream ended normally
				break
			}
			logrus.WithError(err).WithFields(logrus.Fields{
				"instance_id":  instanceID,
				"container_id": containerID,
			}).Error("Failed to receive container logs")
			basehandlers.RespondInternalError(c, err)
			return
		}

		logs = append(logs, logEntry)
	}

	basehandlers.RespondSuccess(c, logs)
}

// GetContainerLogsStream godoc
// @Summary Get container logs (streaming)
// @Description Stream real-time logs from a Docker container using Server-Sent Events (SSE)
// @Tags docker-containers
// @Accept json
// @Produce text/event-stream
// @Param id path string true "Docker Instance ID (UUID)"
// @Param container_id path string true "Container ID or name"
// @Param tail query int false "Number of lines to show from the end (default: 100)"
// @Param since query string false "Show logs since timestamp or relative"
// @Param timestamps query boolean false "Show timestamps (default: false)"
// @Param stdout query boolean false "Show stdout (default: true)"
// @Param stderr query boolean false "Show stderr (default: true)"
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
	tailStr := c.DefaultQuery("tail", "100")
	sinceStr := c.Query("since")
	timestamps := c.DefaultQuery("timestamps", "false") == "true"
	stdout := c.DefaultQuery("stdout", "true") == "true"
	stderr := c.DefaultQuery("stderr", "true") == "true"

	// Parse since parameter (Unix timestamp or relative time)
	var since int64
	if sinceStr != "" {
		// Try to parse as Unix timestamp
		if ts, err := strconv.ParseInt(sinceStr, 10, 64); err == nil {
			since = ts
		}
		// TODO: Support relative time like "42m" (requires time parsing)
	}

	// Create logs request (follow=true for streaming)
	req := &pb.GetContainerLogsRequest{
		ContainerId: containerID,
		Follow:      true,
		Tail:        tailStr,
		Since:       since,
		Timestamps:  timestamps,
		Stdout:      stdout,
		Stderr:      stderr,
	}

	// Get logs stream from agent
	stream, err := h.agentForwarder.GetContainerLogs(instanceID, req)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id":  instanceID,
			"container_id": containerID,
		}).Error("Failed to get container logs stream")
		basehandlers.RespondInternalError(c, err)
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create a channel to signal when to stop
	clientGone := c.Request.Context().Done()

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
		"tail":         tailStr,
		"timestamps":   timestamps,
	}).Info("Starting container logs stream")

	// Stream logs to client
	for {
		select {
		case <-clientGone:
			// Client disconnected
			logrus.WithFields(logrus.Fields{
				"instance_id":  instanceID,
				"container_id": containerID,
			}).Info("Client disconnected from logs stream")
			return

		default:
			// Read log entry from gRPC stream
			logEntry, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					// Stream ended normally
					logrus.WithFields(logrus.Fields{
						"instance_id":  instanceID,
						"container_id": containerID,
					}).Info("Logs stream ended")
					return
				}
				// Stream error
				logrus.WithError(err).WithFields(logrus.Fields{
					"instance_id":  instanceID,
					"container_id": containerID,
				}).Error("Error receiving container logs")

				// Send error event
				errorData := map[string]interface{}{
					"error":   err.Error(),
					"message": "Failed to receive logs",
				}
				errorJSON, _ := json.Marshal(errorData)
				fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", errorJSON)
				flusher.Flush()
				return
			}

			// Convert log entry to JSON
			logJSON, err := json.Marshal(logEntry)
			if err != nil {
				logrus.WithError(err).Error("Failed to marshal log entry to JSON")
				continue
			}

			// Send log as SSE event
			// Format: data: {json}\n\n
			fmt.Fprintf(c.Writer, "data: %s\n\n", logJSON)
			flusher.Flush()
		}
	}
}
