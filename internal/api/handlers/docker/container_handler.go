package docker

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	basehandlers "github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/services/docker"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// ContainerHandler handles Docker container API requests
type ContainerHandler struct {
	containerService *docker.ContainerService
	agentForwarder   *docker.AgentForwarderV2
}

// NewContainerHandler creates a new ContainerHandler
func NewContainerHandler(
	containerService *docker.ContainerService,
	agentForwarder *docker.AgentForwarderV2,
) *ContainerHandler {
	return &ContainerHandler{
		containerService: containerService,
		agentForwarder:   agentForwarder,
	}
}

// GetContainers godoc
// @Summary List containers
// @Description Get a list of Docker containers for a specific instance with pagination and filtering
// @Tags docker-containers
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20, max: 100)"
// @Param all query boolean false "Show all containers (default: false, only running)"
// @Param name query string false "Filter by container name"
// @Success 200 {object} handlers.PaginatedResponse
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/containers [get]
// @Security BearerAuth
func (h *ContainerHandler) GetContainers(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Parse filter parameters
	all := c.DefaultQuery("all", "false") == "true"
	nameFilter := c.Query("name")

	// Build filter string for Docker API
	filterStr := ""
	if nameFilter != "" {
		filterStr = "name=" + nameFilter
	}

	// Get containers from agent
	req := &pb.ListContainersRequest{
		All:     all,
		Filters: filterStr,
		Limit:   int32(pageSize),
		Page:    int32(page),
	}

	resp, err := h.agentForwarder.ListContainers(instanceID, req)
	if err != nil {
		logrus.WithError(err).WithField("instance_id", instanceID).Error("Failed to list containers")
		basehandlers.RespondInternalError(c, err)
		return
	}

	// For simplicity, return all containers (pagination can be done client-side for now)
	// TODO: Implement server-side pagination if needed
	total := int64(len(resp.Containers))

	basehandlers.RespondPaginated(c, resp.Containers, page, pageSize, total)
}

// GetContainer godoc
// @Summary Get container details
// @Description Get detailed information about a specific Docker container
// @Tags docker-containers
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param container_id path string true "Container ID or name"
// @Success 200 {object} handlers.SuccessResponse
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/containers/{container_id} [get]
// @Security BearerAuth
func (h *ContainerHandler) GetContainer(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	containerID := c.Param("container_id")
	if containerID == "" {
		basehandlers.RespondBadRequest(c, gin.Error{Err: err, Type: gin.ErrorTypeBind})
		return
	}

	// Get container details from agent
	req := &pb.GetContainerRequest{
		ContainerId: containerID,
	}
	resp, err := h.agentForwarder.GetContainer(instanceID, req)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id":  instanceID,
			"container_id": containerID,
		}).Error("Failed to inspect container")
		basehandlers.RespondNotFound(c, err)
		return
	}

	basehandlers.RespondSuccess(c, resp.Container)
}

// StartContainerRequest represents the request body for starting a container
type StartContainerRequest struct {
	ContainerID string `json:"container_id" binding:"required"`
}

// StartContainer godoc
// @Summary Start container
// @Description Start a Docker container
// @Tags docker-containers
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body StartContainerRequest true "Start container request"
// @Success 200 {object} handlers.SuccessResponse{data=map[string]interface{}}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/containers/start [post]
// @Security BearerAuth
func (h *ContainerHandler) StartContainer(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	var req StartContainerRequest
	if !basehandlers.BindJSON(c, &req) {
		return
	}

	// Get user info for audit logging
	userID := c.GetString("user_id")
	username := c.GetString("username")
	clientIP := c.ClientIP()

	var userIDPtr *uuid.UUID
	if userID != "" {
		uid, _ := uuid.Parse(userID)
		userIDPtr = &uid
	}

	// Start container through service layer (with audit logging)
	err = h.containerService.StartContainer(c.Request.Context(), instanceID, req.ContainerID, userIDPtr, username, clientIP)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id":  instanceID,
			"container_id": req.ContainerID,
			"user":         username,
		}).Error("Failed to start container")
		basehandlers.RespondInternalError(c, err)
		return
	}

	basehandlers.RespondSuccess(c, map[string]interface{}{
		"message":      "Container started successfully",
		"container_id": req.ContainerID,
	})
}

// StopContainerRequest represents the request body for stopping a container
type StopContainerRequest struct {
	ContainerID string `json:"container_id" binding:"required"`
	Timeout     *int32 `json:"timeout"` // Optional timeout in seconds
}

// StopContainer godoc
// @Summary Stop container
// @Description Stop a running Docker container
// @Tags docker-containers
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body StopContainerRequest true "Stop container request"
// @Success 200 {object} handlers.SuccessResponse{data=map[string]interface{}}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/containers/stop [post]
// @Security BearerAuth
func (h *ContainerHandler) StopContainer(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	var req StopContainerRequest
	if !basehandlers.BindJSON(c, &req) {
		return
	}

	// Get user info for audit logging
	userID := c.GetString("user_id")
	username := c.GetString("username")
	clientIP := c.ClientIP()

	var userIDPtr *uuid.UUID
	if userID != "" {
		uid, _ := uuid.Parse(userID)
		userIDPtr = &uid
	}

	// Stop container through service layer (with audit logging)
	timeout := int32(10) // default 10 seconds
	if req.Timeout != nil {
		timeout = *req.Timeout
	}
	err = h.containerService.StopContainer(c.Request.Context(), instanceID, req.ContainerID, timeout, userIDPtr, username, clientIP)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id":  instanceID,
			"container_id": req.ContainerID,
			"user":         username,
		}).Error("Failed to stop container")
		basehandlers.RespondInternalError(c, err)
		return
	}

	basehandlers.RespondSuccess(c, map[string]interface{}{
		"message":      "Container stopped successfully",
		"container_id": req.ContainerID,
	})
}

// RestartContainerRequest represents the request body for restarting a container
type RestartContainerRequest struct {
	ContainerID string `json:"container_id" binding:"required"`
	Timeout     *int32 `json:"timeout"` // Optional timeout in seconds
}

// RestartContainer godoc
// @Summary Restart container
// @Description Restart a Docker container
// @Tags docker-containers
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body RestartContainerRequest true "Restart container request"
// @Success 200 {object} handlers.SuccessResponse{data=map[string]interface{}}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/containers/restart [post]
// @Security BearerAuth
func (h *ContainerHandler) RestartContainer(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	var req RestartContainerRequest
	if !basehandlers.BindJSON(c, &req) {
		return
	}

	// Get user info for audit logging
	userID := c.GetString("user_id")
	username := c.GetString("username")
	clientIP := c.ClientIP()

	var userIDPtr *uuid.UUID
	if userID != "" {
		uid, _ := uuid.Parse(userID)
		userIDPtr = &uid
	}

	// Restart container through service layer (with audit logging)
	timeout := int32(10) // default 10 seconds
	if req.Timeout != nil {
		timeout = *req.Timeout
	}
	err = h.containerService.RestartContainer(c.Request.Context(), instanceID, req.ContainerID, timeout, userIDPtr, username, clientIP)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id":  instanceID,
			"container_id": req.ContainerID,
			"user":         username,
		}).Error("Failed to restart container")
		basehandlers.RespondInternalError(c, err)
		return
	}

	basehandlers.RespondSuccess(c, map[string]interface{}{
		"message":      "Container restarted successfully",
		"container_id": req.ContainerID,
	})
}

// PauseContainerRequest represents the request body for pausing a container
type PauseContainerRequest struct {
	ContainerID string `json:"container_id" binding:"required"`
}

// PauseContainer godoc
// @Summary Pause container
// @Description Pause a running Docker container
// @Tags docker-containers
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body PauseContainerRequest true "Pause container request"
// @Success 200 {object} handlers.SuccessResponse{data=map[string]interface{}}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/containers/pause [post]
// @Security BearerAuth
func (h *ContainerHandler) PauseContainer(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	var req PauseContainerRequest
	if !basehandlers.BindJSON(c, &req) {
		return
	}

	// Get user info for audit logging
	userID := c.GetString("user_id")
	username := c.GetString("username")
	clientIP := c.ClientIP()

	var userIDPtr *uuid.UUID
	if userID != "" {
		uid, _ := uuid.Parse(userID)
		userIDPtr = &uid
	}

	// Pause container through service layer (with audit logging)
	err = h.containerService.PauseContainer(c.Request.Context(), instanceID, req.ContainerID, userIDPtr, username, clientIP)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id":  instanceID,
			"container_id": req.ContainerID,
			"user":         username,
		}).Error("Failed to pause container")
		basehandlers.RespondInternalError(c, err)
		return
	}

	basehandlers.RespondSuccess(c, map[string]interface{}{
		"message":      "Container paused successfully",
		"container_id": req.ContainerID,
	})
}

// UnpauseContainerRequest represents the request body for unpausing a container
type UnpauseContainerRequest struct {
	ContainerID string `json:"container_id" binding:"required"`
}

// UnpauseContainer godoc
// @Summary Unpause container
// @Description Unpause a paused Docker container
// @Tags docker-containers
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body UnpauseContainerRequest true "Unpause container request"
// @Success 200 {object} handlers.SuccessResponse{data=map[string]interface{}}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/containers/unpause [post]
// @Security BearerAuth
func (h *ContainerHandler) UnpauseContainer(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	var req UnpauseContainerRequest
	if !basehandlers.BindJSON(c, &req) {
		return
	}

	// Get user info for audit logging
	userID := c.GetString("user_id")
	username := c.GetString("username")
	clientIP := c.ClientIP()

	var userIDPtr *uuid.UUID
	if userID != "" {
		uid, _ := uuid.Parse(userID)
		userIDPtr = &uid
	}

	// Unpause container through service layer (with audit logging)
	err = h.containerService.UnpauseContainer(c.Request.Context(), instanceID, req.ContainerID, userIDPtr, username, clientIP)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id":  instanceID,
			"container_id": req.ContainerID,
			"user":         username,
		}).Error("Failed to unpause container")
		basehandlers.RespondInternalError(c, err)
		return
	}

	basehandlers.RespondSuccess(c, map[string]interface{}{
		"message":      "Container unpaused successfully",
		"container_id": req.ContainerID,
	})
}

// DeleteContainerRequest represents the request body for deleting a container
type DeleteContainerRequest struct {
	ContainerID   string `json:"container_id" binding:"required"`
	Force         bool   `json:"force"`          // Force removal of running container
	RemoveVolumes bool   `json:"remove_volumes"` // Remove associated volumes
}

// DeleteContainer godoc
// @Summary Delete container
// @Description Delete a Docker container
// @Tags docker-containers
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body DeleteContainerRequest true "Delete container request"
// @Success 200 {object} handlers.SuccessResponse{data=map[string]interface{}}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id}/containers/delete [post]
// @Security BearerAuth
func (h *ContainerHandler) DeleteContainer(c *gin.Context) {
	instanceID, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	var req DeleteContainerRequest
	if !basehandlers.BindJSON(c, &req) {
		return
	}

	// Get user info for audit logging
	userID := c.GetString("user_id")
	username := c.GetString("username")
	clientIP := c.ClientIP()

	var userIDPtr *uuid.UUID
	if userID != "" {
		uid, _ := uuid.Parse(userID)
		userIDPtr = &uid
	}

	// Delete container through service layer (with audit logging)
	err = h.containerService.DeleteContainer(c.Request.Context(), instanceID, req.ContainerID, req.Force, req.RemoveVolumes, userIDPtr, username, clientIP)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"instance_id":  instanceID,
			"container_id": req.ContainerID,
			"user":         username,
		}).Error("Failed to delete container")
		basehandlers.RespondInternalError(c, err)
		return
	}

	basehandlers.RespondSuccess(c, map[string]interface{}{
		"message":      "Container deleted successfully",
		"container_id": req.ContainerID,
	})
}
