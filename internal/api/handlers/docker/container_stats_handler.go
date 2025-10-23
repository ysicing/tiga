package docker

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	basehandlers "github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/services/docker"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// ContainerStatsHandler handles Docker container stats API requests
type ContainerStatsHandler struct {
	agentForwarder *docker.AgentForwarder
}

// NewContainerStatsHandler creates a new ContainerStatsHandler
func NewContainerStatsHandler(agentForwarder *docker.AgentForwarder) *ContainerStatsHandler {
	return &ContainerStatsHandler{
		agentForwarder: agentForwarder,
	}
}

// GetContainerStats godoc
// @Summary Get container stats (single query)
// @Description Get current resource usage statistics for a Docker container
// @Tags docker-containers
// @Accept json
// @Produce json
// @Param instance_id path string true "Docker Instance ID (UUID)"
// @Param container_id path string true "Container ID or name"
// @Success 200 {object} handlers.SuccessResponse
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{instance_id}/containers/{container_id}/stats [get]
// @Security BearerAuth
func (h *ContainerStatsHandler) GetContainerStats(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("instance_id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	containerID := c.Param("container_id")
	if containerID == "" {
		basehandlers.RespondBadRequest(c, fmt.Errorf("container_id is required"))
		return
	}

	// Create stats request (stream=false for single query)
	req := &pb.GetContainerStatsRequest{
		ContainerId: containerID,
		Stream:      false,
	}

	// Get stats stream from agent
	stream, err := h.agentForwarder.GetContainerStats(instanceID, req)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id":  instanceID,
			"container_id": containerID,
		}).Error("Failed to get container stats stream")
		basehandlers.RespondInternalError(c, err)
		return
	}

	// Read first (and only) stats message
	stats, err := stream.Recv()
	if err != nil {
		if err == io.EOF {
			basehandlers.RespondNotFound(c, fmt.Errorf("no stats available for container"))
			return
		}
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id":  instanceID,
			"container_id": containerID,
		}).Error("Failed to receive container stats")
		basehandlers.RespondInternalError(c, err)
		return
	}

	basehandlers.RespondSuccess(c, stats)
}

// GetContainerStatsStream godoc
// @Summary Get container stats (streaming)
// @Description Stream real-time resource usage statistics for a Docker container using Server-Sent Events (SSE)
// @Tags docker-containers
// @Accept json
// @Produce text/event-stream
// @Param instance_id path string true "Docker Instance ID (UUID)"
// @Param container_id path string true "Container ID or name"
// @Success 200 {string} string "SSE stream of container stats"
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{instance_id}/containers/{container_id}/stats/stream [get]
// @Security BearerAuth
func (h *ContainerStatsHandler) GetContainerStatsStream(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("instance_id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	containerID := c.Param("container_id")
	if containerID == "" {
		basehandlers.RespondBadRequest(c, fmt.Errorf("container_id is required"))
		return
	}

	// Create stats request (stream=true for continuous streaming)
	req := &pb.GetContainerStatsRequest{
		ContainerId: containerID,
		Stream:      true,
	}

	// Get stats stream from agent
	stream, err := h.agentForwarder.GetContainerStats(instanceID, req)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id":  instanceID,
			"container_id": containerID,
		}).Error("Failed to get container stats stream")
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
	}).Info("Starting container stats stream")

	// Stream stats to client
	for {
		select {
		case <-clientGone:
			// Client disconnected
			logrus.WithFields(logrus.Fields{
				"instance_id":  instanceID,
				"container_id": containerID,
			}).Info("Client disconnected from stats stream")
			return

		default:
			// Read stats from gRPC stream
			stats, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					// Stream ended normally
					logrus.WithFields(logrus.Fields{
						"instance_id":  instanceID,
						"container_id": containerID,
					}).Info("Stats stream ended")
					return
				}
				// Stream error
				logrus.WithError(err).WithFields(logrus.Fields{
					"instance_id":  instanceID,
					"container_id": containerID,
				}).Error("Error receiving container stats")

				// Send error event
				errorData := map[string]interface{}{
					"error":   err.Error(),
					"message": "Failed to receive stats",
				}
				errorJSON, _ := json.Marshal(errorData)
				fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", errorJSON)
				flusher.Flush()
				return
			}

			// Convert stats to JSON
			statsJSON, err := json.Marshal(stats)
			if err != nil {
				logrus.WithError(err).Error("Failed to marshal stats to JSON")
				continue
			}

			// Send stats as SSE event
			// Format: data: {json}\n\n
			fmt.Fprintf(c.Writer, "data: %s\n\n", statsJSON)
			flusher.Flush()
		}
	}
}
