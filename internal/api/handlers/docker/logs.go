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

	// Read logs
	body, err := io.ReadAll(logs)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"logs":      string(body),
		"container": containerID,
	})
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
