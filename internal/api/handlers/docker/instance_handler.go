package docker

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	basehandlers "github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/docker"
)

// InstanceHandler handles Docker instance API requests
type InstanceHandler struct {
	instanceService *docker.DockerInstanceService
	agentForwarder  *docker.AgentForwarderV2
}

// NewInstanceHandler creates a new InstanceHandler
func NewInstanceHandler(
	instanceService *docker.DockerInstanceService,
	agentForwarder *docker.AgentForwarderV2,
) *InstanceHandler {
	return &InstanceHandler{
		instanceService: instanceService,
		agentForwarder:  agentForwarder,
	}
}

// DockerInstanceResponse represents the API response for a Docker instance with Agent info
type DockerInstanceResponse struct {
	ID          string   `json:"id"`
	AgentID     string   `json:"agent_id"`
	AgentName   string   `json:"agent_name"`
	Name        string   `json:"name"`
	Host        string   `json:"host"`        // Agent IP address
	Port        int      `json:"port"`        // gRPC port (default: 50051)
	Description string   `json:"description"`
	Status      string   `json:"status"`      // online, offline, unknown, archived

	// Docker daemon info
	Version          string `json:"version,omitempty"`
	APIVersion       string `json:"api_version,omitempty"`
	OS               string `json:"os,omitempty"`
	Architecture     string `json:"architecture,omitempty"`
	KernelVersion    string `json:"kernel_version,omitempty"`
	OperatingSystem  string `json:"operating_system,omitempty"`

	// Resource counts
	TotalContainers   int    `json:"total_containers,omitempty"`
	RunningContainers int    `json:"running_containers,omitempty"`
	TotalImages       int    `json:"total_images,omitempty"`

	// Timestamps
	LastSeenAt string    `json:"last_seen_at,omitempty"`
	CreatedAt  string    `json:"created_at"`
	UpdatedAt  string    `json:"updated_at"`

	// Metadata
	Labels      map[string]string `json:"labels,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
}

// GetInstances godoc
// @Summary List Docker instances
// @Description Get a list of Docker instances with pagination and filtering
// @Tags docker-instances
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20, max: 100)"
// @Param status query string false "Filter by health status (online, offline, archived, unknown)"
// @Param agent_id query string false "Filter by Agent ID (UUID)"
// @Param search query string false "Search by name or description"
// @Success 200 {object} handlers.PaginatedResponse{data=[]DockerInstanceResponse}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances [get]
// @Security BearerAuth
func (h *InstanceHandler) GetInstances(c *gin.Context) {
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
	filter := &repository.DockerInstanceFilter{
		Page:         page,
		PageSize:     pageSize,
		Name:         c.Query("name"),
		HealthStatus: c.Query("status"),
	}

	// Parse AgentID if provided
	if agentIDStr := c.Query("agent_id"); agentIDStr != "" {
		agentID, err := uuid.Parse(agentIDStr)
		if err != nil {
			basehandlers.RespondBadRequest(c, err)
			return
		}
		filter.AgentID = &agentID
	}

	instances, total, err := h.instanceService.ListInstances(c.Request.Context(), filter)
	if err != nil {
		logrus.WithError(err).Error("Failed to list Docker instances")
		basehandlers.RespondInternalError(c, err)
		return
	}

	// Convert to response DTOs with Agent information
	responses := make([]DockerInstanceResponse, len(instances))
	for i, instance := range instances {
		responses[i] = h.toResponse(c.Request.Context(), instance)
	}

	basehandlers.RespondPaginated(c, responses, page, pageSize, total)
}

// GetInstance godoc
// @Summary Get Docker instance
// @Description Get a Docker instance by ID
// @Tags docker-instances
// @Accept json
// @Produce json
// @Param id path string true "Instance ID (UUID)"
// @Success 200 {object} handlers.SuccessResponse{data=DockerInstanceResponse}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id} [get]
// @Security BearerAuth
func (h *InstanceHandler) GetInstance(c *gin.Context) {
	id, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	instance, err := h.instanceService.GetByID(c.Request.Context(), id)
	if err != nil {
		logrus.WithError(err).WithField("instance_id", id).Error("Failed to get Docker instance")
		basehandlers.RespondNotFound(c, err)
		return
	}

	response := h.toResponse(c.Request.Context(), instance)
	basehandlers.RespondSuccess(c, response)
}

// toResponse converts a DockerInstance model to DockerInstanceResponse with Agent information
func (h *InstanceHandler) toResponse(ctx context.Context, instance *models.DockerInstance) DockerInstanceResponse {
	db := ctx.Value("db").(*gorm.DB)

	response := DockerInstanceResponse{
		ID:                instance.ID.String(),
		AgentID:           instance.AgentID.String(),
		Name:              instance.Name,
		Description:       instance.Description,
		Status:            instance.HealthStatus,
		Version:           instance.DockerVersion,
		APIVersion:        instance.APIVersion,
		OS:                instance.OperatingSystem,
		Architecture:      instance.Architecture,
		KernelVersion:     instance.KernelVersion,
		OperatingSystem:   instance.OperatingSystem,
		TotalContainers:   instance.ContainerCount,
		TotalImages:       instance.ImageCount,
		Port:              50051, // Default gRPC port
		Tags:              instance.Tags,
		CreatedAt:         instance.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         instance.UpdatedAt.Format(time.RFC3339),
	}

	if !instance.LastConnectedAt.IsZero() {
		response.LastSeenAt = instance.LastConnectedAt.Format(time.RFC3339)
	}

	// Query Agent information from agent_connections table
	var agentConn models.AgentConnection
	if err := db.Where("id = ?", instance.AgentID).First(&agentConn).Error; err == nil {
		response.Host = agentConn.IPAddress

		// Get host node name as agent_name
		var hostNode models.HostNode
		if err := db.Where("id = ?", agentConn.HostNodeID).First(&hostNode).Error; err == nil {
			response.AgentName = hostNode.Name
		}
	} else {
		logrus.WithError(err).WithField("agent_id", instance.AgentID).Warn("Failed to get agent connection info")
		response.Host = "unknown"
		response.AgentName = "unknown"
	}

	return response
}

// CreateInstanceRequest represents the request body for creating a Docker instance
type CreateInstanceRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	AgentID     string   `json:"agent_id" binding:"required,uuid"`
	Tags        []string `json:"tags"`
}

// CreateInstance godoc
// @Summary Create Docker instance
// @Description Manually register a Docker instance (Note: Auto-discovery is preferred via Agent)
// @Tags docker-instances
// @Accept json
// @Produce json
// @Param request body CreateInstanceRequest true "Instance creation request"
// @Success 201 {object} handlers.SuccessResponse{data=models.DockerInstance}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances [post]
// @Security BearerAuth
func (h *InstanceHandler) CreateInstance(c *gin.Context) {
	var req CreateInstanceRequest
	if !basehandlers.BindJSON(c, &req) {
		return
	}

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	instance, err := h.instanceService.CreateManualInstance(
		c.Request.Context(),
		req.Name,
		req.Description,
		agentID,
		req.Tags,
	)
	if err != nil {
		logrus.WithError(err).WithField("agent_id", agentID).Error("Failed to create Docker instance")
		basehandlers.RespondInternalError(c, err)
		return
	}

	logrus.WithFields(logrus.Fields{
		"instance_id":   instance.ID,
		"instance_name": instance.Name,
		"agent_id":      agentID,
	}).Info("Docker instance created manually")

	basehandlers.RespondCreated(c, instance)
}

// UpdateInstanceRequest represents the request body for updating a Docker instance
type UpdateInstanceRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Tags        []string `json:"tags"`
}

// UpdateInstance godoc
// @Summary Update Docker instance
// @Description Update Docker instance metadata (name, description, labels)
// @Tags docker-instances
// @Accept json
// @Produce json
// @Param id path string true "Instance ID (UUID)"
// @Param request body UpdateInstanceRequest true "Instance update request"
// @Success 200 {object} handlers.SuccessResponse{data=models.DockerInstance}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id} [put]
// @Security BearerAuth
func (h *InstanceHandler) UpdateInstance(c *gin.Context) {
	id, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	var req UpdateInstanceRequest
	if !basehandlers.BindJSON(c, &req) {
		return
	}

	// Get existing instance
	instance, err := h.instanceService.GetByID(c.Request.Context(), id)
	if err != nil {
		logrus.WithError(err).WithField("instance_id", id).Error("Failed to get Docker instance for update")
		basehandlers.RespondNotFound(c, err)
		return
	}

	// Update fields
	updated := false
	if req.Name != nil && *req.Name != instance.Name {
		instance.Name = *req.Name
		updated = true
	}
	if req.Description != nil && *req.Description != instance.Description {
		instance.Description = *req.Description
		updated = true
	}
	if req.Tags != nil {
		instance.Tags = req.Tags
		updated = true
	}

	if !updated {
		basehandlers.RespondSuccess(c, instance)
		return
	}

	// Save updates
	if err := h.instanceService.UpdateFromModel(c.Request.Context(), instance); err != nil {
		logrus.WithError(err).WithField("instance_id", id).Error("Failed to update Docker instance")
		basehandlers.RespondInternalError(c, err)
		return
	}

	logrus.WithFields(logrus.Fields{
		"instance_id":   instance.ID,
		"instance_name": instance.Name,
	}).Info("Docker instance updated")

	basehandlers.RespondSuccess(c, instance)
}

// DeleteInstance godoc
// @Summary Delete Docker instance
// @Description Archive a Docker instance (soft delete)
// @Tags docker-instances
// @Accept json
// @Produce json
// @Param id path string true "Instance ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/{id} [delete]
// @Security BearerAuth
func (h *InstanceHandler) DeleteInstance(c *gin.Context) {
	id, err := basehandlers.ParseUUID(c.Param("id"))
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	// Archive the instance (soft delete)
	if err := h.instanceService.Archive(c.Request.Context(), id); err != nil {
		logrus.WithError(err).WithField("instance_id", id).Error("Failed to archive Docker instance")
		basehandlers.RespondInternalError(c, err)
		return
	}

	logrus.WithField("instance_id", id).Info("Docker instance archived")

	basehandlers.RespondNoContent(c)
}

// TestConnectionRequest represents the request body for testing connection
type TestConnectionRequest struct {
	AgentID string `json:"agent_id" binding:"required,uuid"`
}

// TestConnectionResponse represents the response for connection test
type TestConnectionResponse struct {
	Success      bool                   `json:"success"`
	Message      string                 `json:"message"`
	DockerInfo   map[string]interface{} `json:"docker_info,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// TestConnection godoc
// @Summary Test Docker connection
// @Description Test connection to Docker daemon via Agent
// @Tags docker-instances
// @Accept json
// @Produce json
// @Param request body TestConnectionRequest true "Connection test request"
// @Success 200 {object} handlers.SuccessResponse{data=TestConnectionResponse}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances/test-connection [post]
// @Security BearerAuth
func (h *InstanceHandler) TestConnection(c *gin.Context) {
	var req TestConnectionRequest
	if !basehandlers.BindJSON(c, &req) {
		return
	}

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		basehandlers.RespondBadRequest(c, err)
		return
	}

	// Try to get Docker info from agent
	dockerInfo, err := h.agentForwarder.GetDockerInfo(agentID)
	if err != nil {
		logrus.WithError(err).WithField("agent_id", agentID).Warn("Docker connection test failed")

		response := TestConnectionResponse{
			Success:      false,
			Message:      "Failed to connect to Docker daemon",
			ErrorMessage: err.Error(),
		}
		basehandlers.RespondSuccess(c, response)
		return
	}

	// Build info map
	infoMap := map[string]interface{}{
		"version":            dockerInfo.Info.Driver,           // Using driver as version
		"api_version":        "",                               // Not available in SystemInfo
		"min_api_version":    "",                               // Not available in SystemInfo
		"storage_driver":     dockerInfo.Info.Driver,
		"operating_system":   dockerInfo.Info.OperatingSystem,
		"architecture":       dockerInfo.Info.Architecture,
		"kernel_version":     dockerInfo.Info.KernelVersion,
		"mem_total":          dockerInfo.Info.MemTotal,
		"n_cpu":              dockerInfo.Info.Ncpu,
		"containers":         dockerInfo.Info.Containers,
		"containers_running": dockerInfo.Info.ContainersRunning,
		"containers_paused":  dockerInfo.Info.ContainersPaused,
		"containers_stopped": dockerInfo.Info.ContainersStopped,
		"images":             dockerInfo.Info.Images,
	}

	response := TestConnectionResponse{
		Success:    true,
		Message:    "Docker connection successful",
		DockerInfo: infoMap,
	}

	logrus.WithFields(logrus.Fields{
		"agent_id":       agentID,
		"docker_version": dockerInfo.Info.Driver,  // Using driver as version
	}).Info("Docker connection test successful")

	basehandlers.RespondSuccess(c, response)
}
