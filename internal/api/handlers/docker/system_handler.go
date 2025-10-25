package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/services/docker"

	basehandlers "github.com/ysicing/tiga/internal/api/handlers"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// SystemHandler handles Docker system-level operations
type SystemHandler struct {
	agentForwarder *docker.AgentForwarderV2
}

// NewSystemHandler creates a new SystemHandler instance
func NewSystemHandler(agentForwarder *docker.AgentForwarderV2) *SystemHandler {
	return &SystemHandler{
		agentForwarder: agentForwarder,
	}
}

// GetSystemInfo gets Docker system information
// @Summary Get Docker system information
// @Description Get detailed system information from the Docker daemon
// @Tags Docker System
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Success 200 {object} map[string]interface{} "System information"
// @Failure 400 {object} map[string]interface{} "Invalid instance ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/system/info [get]
func (h *SystemHandler) GetSystemInfo(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Forward request to agent
	req := &pb.GetSystemInfoRequest{}
	resp, err := h.agentForwarder.GetSystemInfo(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to get system info: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}

// GetVersion gets Docker version information
// @Summary Get Docker version
// @Description Get Docker version and build information
// @Tags Docker System
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Success 200 {object} map[string]interface{} "Version information"
// @Failure 400 {object} map[string]interface{} "Invalid instance ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/system/version [get]
func (h *SystemHandler) GetVersion(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Forward request to agent
	req := &pb.GetVersionRequest{}
	resp, err := h.agentForwarder.GetVersion(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to get version: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}

// GetDiskUsage gets Docker disk usage information
// @Summary Get Docker disk usage
// @Description Get disk usage information for images, containers, volumes, and build cache
// @Tags Docker System
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Success 200 {object} map[string]interface{} "Disk usage information"
// @Failure 400 {object} map[string]interface{} "Invalid instance ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/system/disk-usage [get]
func (h *SystemHandler) GetDiskUsage(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Forward request to agent
	req := &pb.GetDiskUsageRequest{}
	resp, err := h.agentForwarder.GetDiskUsage(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to get disk usage: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}

// Ping pings the Docker daemon
// @Summary Ping Docker daemon
// @Description Check if Docker daemon is responsive
// @Tags Docker System
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Success 200 {object} map[string]interface{} "Ping response"
// @Failure 400 {object} map[string]interface{} "Invalid instance ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/system/ping [get]
func (h *SystemHandler) Ping(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Forward request to agent
	req := &pb.PingRequest{}
	resp, err := h.agentForwarder.Ping(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to ping: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}

// GetEventsStream streams Docker events in real-time
// @Summary Stream Docker events
// @Description Stream Docker events in real-time using Server-Sent Events (SSE)
// @Tags Docker System
// @Accept json
// @Produce text/event-stream
// @Param id path string true "Docker Instance ID (UUID)"
// @Param since query string false "Show events since timestamp or relative (e.g., '10m')"
// @Param until query string false "Show events until timestamp or relative"
// @Param filters query string false "Event filters in JSON format"
// @Success 200 {object} map[string]interface{} "SSE stream of Docker events"
// @Failure 400 {object} map[string]interface{} "Invalid instance ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/system/events/stream [get]
func (h *SystemHandler) GetEventsStream(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Get query parameters
	since := c.DefaultQuery("since", "")
	until := c.DefaultQuery("until", "")
	filters := c.DefaultQuery("filters", "")

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Get flusher
	flusher, ok := c.Writer.(interface{ Flush() })
	if !ok {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("streaming not supported"))
		return
	}

	// Start streaming
	req := &pb.GetEventsRequest{
		Since:   since,
		Until:   until,
		Filters: filters,
	}
	stream, err := h.agentForwarder.GetEvents(instanceID, req)
	if err != nil {
		// Send error event
		errorJSON, _ := json.Marshal(map[string]interface{}{
			"error": err.Error(),
		})
		fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", errorJSON)
		flusher.Flush()
		return
	}

	// Monitor client disconnection
	clientGone := c.Request.Context().Done()

	// Stream events
	for {
		select {
		case <-clientGone:
			// Client disconnected
			return
		default:
			event, err := stream.Recv()
			if err == io.EOF {
				// Stream ended normally
				return
			}
			if err != nil {
				// Send error event
				errorJSON, _ := json.Marshal(map[string]interface{}{
					"error": err.Error(),
				})
				fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", errorJSON)
				flusher.Flush()
				return
			}

			// Send event data
			eventJSON, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON)
			flusher.Flush()
		}
	}
}
