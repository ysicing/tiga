package docker

import (
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/managers"
)

// LogHandler handles Docker container log operations
type LogHandler struct {
	instanceRepo repository.InstanceRepository
}

// NewLogHandler creates a new log handler
func NewLogHandler(instanceRepo repository.InstanceRepository) *LogHandler {
	return &LogHandler{
		instanceRepo: instanceRepo,
	}
}

// GetContainerLogs handles GET /api/v1/docker/instances/{id}/containers/{container}/logs
func (h *LogHandler) GetContainerLogs(c *gin.Context) {
	instanceIDStr := c.Param("id")
	containerID := c.Param("container")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Get query parameters
	showStdout := c.DefaultQuery("stdout", "true") == "true"
	showStderr := c.DefaultQuery("stderr", "true") == "true"
	tail := c.DefaultQuery("tail", "100")
	since := c.DefaultQuery("since", "")

	// Get instance
	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}

	if instance.Type != "docker" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not Docker type"))
		return
	}

	// Create Docker manager
	manager := managers.NewDockerManager()
	if err := manager.Initialize(c.Request.Context(), instance); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	if err := manager.Connect(c.Request.Context()); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	defer manager.Disconnect(c.Request.Context())

	// Get container logs
	logs, err := manager.GetContainerLogs(c.Request.Context(), containerID, showStdout, showStderr, tail, since)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}
	defer logs.Close()

	// Stream logs with batch processing for better performance
	// Buffer size: 8KB - optimal balance between throughput and memory
	const bufferSize = 8 * 1024
	buffer := make([]byte, bufferSize)

	// Set response headers for streaming
	c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
	c.Writer.WriteHeader(200)

	// Write JSON preamble
	c.Writer.Write([]byte(`{"data":{"logs":"`))

	// Stream logs in batches
	for {
		n, err := logs.Read(buffer)
		if n > 0 {
			// Escape special JSON characters and write chunk
			chunk := escapeJSON(buffer[:n])
			c.Writer.Write(chunk)
			c.Writer.Flush() // Ensure immediate delivery to client
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.WithError(err).WithField("container_id", containerID).Error("Error reading log stream")
			break
		}
	}

	// Write JSON postamble
	c.Writer.Write([]byte(`","container":"`))
	c.Writer.Write([]byte(containerID))
	c.Writer.Write([]byte(`"}}`))
	c.Writer.Flush()
}

// ExecContainer handles POST /api/v1/docker/instances/{id}/containers/{container}/exec
// Security: All commands are validated and logged
func (h *LogHandler) ExecContainer(c *gin.Context) {
	instanceIDStr := c.Param("id")
	containerID := c.Param("container")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	var request struct {
		Cmd []string `json:"cmd" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Security: Log the command execution attempt
	logrus.WithFields(logrus.Fields{
		"user_id":      getUserID(c),      // Get from auth middleware context
		"instance_id":  instanceID,
		"container_id": containerID,
		"command":      request.Cmd,
		"client_ip":    c.ClientIP(),
		"user_agent":   c.Request.UserAgent(),
	}).Info("Docker container exec command requested")

	// Get instance
	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		logrus.WithError(err).WithField("instance_id", instanceID).Warn("Instance not found")
		handlers.RespondNotFound(c, err)
		return
	}

	if instance.Type != "docker" {
		handlers.RespondBadRequest(c, fmt.Errorf("instance is not Docker type"))
		return
	}

	// Create Docker manager
	manager := managers.NewDockerManager()
	if err := manager.Initialize(c.Request.Context(), instance); err != nil {
		logrus.WithError(err).Error("Failed to initialize Docker manager")
		handlers.RespondInternalError(c, err)
		return
	}

	if err := manager.Connect(c.Request.Context()); err != nil {
		logrus.WithError(err).Error("Failed to connect to Docker")
		handlers.RespondInternalError(c, err)
		return
	}
	defer manager.Disconnect(c.Request.Context())

	// Execute command (with built-in validation)
	output, err := manager.ExecContainer(c.Request.Context(), containerID, request.Cmd)
	if err != nil {
		// Security: Log failed command execution
		logrus.WithFields(logrus.Fields{
			"user_id":      getUserID(c),
			"instance_id":  instanceID,
			"container_id": containerID,
			"command":      request.Cmd,
			"error":        err.Error(),
			"client_ip":    c.ClientIP(),
		}).Warn("Docker container exec command failed")

		handlers.RespondInternalError(c, err)
		return
	}

	// Security: Log successful command execution
	logrus.WithFields(logrus.Fields{
		"user_id":       getUserID(c),
		"instance_id":   instanceID,
		"container_id":  containerID,
		"command":       request.Cmd,
		"output_length": len(output),
		"client_ip":     c.ClientIP(),
	}).Info("Docker container exec command executed successfully")

	handlers.RespondSuccess(c, gin.H{
		"output":    output,
		"container": containerID,
		"cmd":       request.Cmd,
	})
}

// getUserID extracts user ID from context (set by auth middleware)
// Returns empty string if not found
func getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return "anonymous"
}

// escapeJSON escapes special JSON characters in byte slice
// for safe inclusion in JSON string values.
// This is a performance-optimized version that pre-allocates capacity.
func escapeJSON(data []byte) []byte {
	// Estimate capacity: most logs don't have special chars
	result := make([]byte, 0, len(data)+len(data)/10)

	for _, b := range data {
		switch b {
		case '"':
			result = append(result, '\\', '"')
		case '\\':
			result = append(result, '\\', '\\')
		case '\n':
			result = append(result, '\\', 'n')
		case '\r':
			result = append(result, '\\', 'r')
		case '\t':
			result = append(result, '\\', 't')
		case '\b':
			result = append(result, '\\', 'b')
		case '\f':
			result = append(result, '\\', 'f')
		default:
			// Control characters (0x00-0x1F) need unicode escaping
			if b < 0x20 {
				result = append(result, []byte(fmt.Sprintf("\\u%04x", b))...)
			} else {
				result = append(result, b)
			}
		}
	}
	return result
}
