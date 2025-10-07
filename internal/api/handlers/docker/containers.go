package docker

import (
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/managers"
)

// ContainerHandler handles Docker container operations
type ContainerHandler struct {
	instanceRepo repository.InstanceRepository
}

// NewContainerHandler creates a new container handler
func NewContainerHandler(instanceRepo repository.InstanceRepository) *ContainerHandler {
	return &ContainerHandler{
		instanceRepo: instanceRepo,
	}
}

// ListContainers handles GET /api/v1/docker/instances/{id}/containers
func (h *ContainerHandler) ListContainers(c *gin.Context) {
	instanceIDStr := c.Param("id")
	all := c.DefaultQuery("all", "false") == "true"

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

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

	// List containers
	containers, err := manager.ListContainers(c.Request.Context(), all)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"containers": containers,
		"count":      len(containers),
	})
}

// GetContainer handles GET /api/v1/docker/instances/{id}/containers/{container}
func (h *ContainerHandler) GetContainer(c *gin.Context) {
	instanceIDStr := c.Param("id")
	containerID := c.Param("container")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

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

	// Get container
	containerJSON, err := manager.GetContainer(c.Request.Context(), containerID)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, containerJSON)
}

// CreateContainer handles POST /api/v1/docker/instances/{id}/containers
func (h *ContainerHandler) CreateContainer(c *gin.Context) {
	instanceIDStr := c.Param("id")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	var request struct {
		Name          string            `json:"name" binding:"required"`
		Image         string            `json:"image" binding:"required"`
		Cmd           []string          `json:"cmd"`
		Env           []string          `json:"env"`
		Ports         map[string]string `json:"ports"`
		Volumes       map[string]string `json:"volumes"`
		RestartPolicy string            `json:"restart_policy"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

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

	// Build container config
	config := &container.Config{
		Image: request.Image,
		Cmd:   request.Cmd,
		Env:   request.Env,
	}

	hostConfig := &container.HostConfig{}
	if request.RestartPolicy != "" {
		hostConfig.RestartPolicy = container.RestartPolicy{
			Name: container.RestartPolicyMode(request.RestartPolicy),
		}
	}

	// Create container
	containerID, err := manager.CreateContainer(c.Request.Context(), config, hostConfig, request.Name)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondCreated(c, gin.H{
		"id":   containerID,
		"name": request.Name,
	})
}

// StartContainer handles POST /api/v1/docker/instances/{id}/containers/{container}/start
func (h *ContainerHandler) StartContainer(c *gin.Context) {
	instanceIDStr := c.Param("id")
	containerID := c.Param("container")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

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

	// Start container
	if err := manager.StartContainer(c.Request.Context(), containerID); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccessWithMessage(c, nil, "container started successfully")
}

// StopContainer handles POST /api/v1/docker/instances/{id}/containers/{container}/stop
func (h *ContainerHandler) StopContainer(c *gin.Context) {
	instanceIDStr := c.Param("id")
	containerID := c.Param("container")
	timeoutStr := c.DefaultQuery("timeout", "10")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil {
		handlers.RespondBadRequest(c, fmt.Errorf("invalid timeout"))
		return
	}

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

	// Stop container
	if err := manager.StopContainer(c.Request.Context(), containerID, &timeout); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccessWithMessage(c, nil, "container stopped successfully")
}

// RestartContainer handles POST /api/v1/docker/instances/{id}/containers/{container}/restart
func (h *ContainerHandler) RestartContainer(c *gin.Context) {
	instanceIDStr := c.Param("id")
	containerID := c.Param("container")
	timeoutStr := c.DefaultQuery("timeout", "10")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil {
		handlers.RespondBadRequest(c, fmt.Errorf("invalid timeout"))
		return
	}

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

	// Restart container
	if err := manager.RestartContainer(c.Request.Context(), containerID, &timeout); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccessWithMessage(c, nil, "container restarted successfully")
}

// DeleteContainer handles DELETE /api/v1/docker/instances/{id}/containers/{container}
func (h *ContainerHandler) DeleteContainer(c *gin.Context) {
	instanceIDStr := c.Param("id")
	containerID := c.Param("container")
	force := c.DefaultQuery("force", "false") == "true"
	volumes := c.DefaultQuery("volumes", "false") == "true"

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

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

	// Remove container
	if err := manager.RemoveContainer(c.Request.Context(), containerID, force, volumes); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccessWithMessage(c, nil, "container deleted successfully")
}

// GetContainerStats handles GET /api/v1/docker/instances/{id}/containers/{container}/stats
func (h *ContainerHandler) GetContainerStats(c *gin.Context) {
	instanceIDStr := c.Param("id")
	containerID := c.Param("container")

	// Parse UUID
	instanceID, err := handlers.ParseUUID(instanceIDStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

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

	// Get stats
	stats, err := manager.GetContainerStats(c.Request.Context(), containerID)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, stats)
}
