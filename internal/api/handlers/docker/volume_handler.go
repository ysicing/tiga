package docker

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	basehandlers "github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/services/docker"
	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// VolumeHandler handles Docker volume operations
type VolumeHandler struct {
	agentForwarder *docker.AgentForwarderV2
}

// NewVolumeHandler creates a new VolumeHandler instance
func NewVolumeHandler(agentForwarder *docker.AgentForwarderV2) *VolumeHandler {
	return &VolumeHandler{
		agentForwarder: agentForwarder,
	}
}

// GetVolumes lists Docker volumes for a given instance
// @Summary List Docker volumes
// @Description Get a list of all volumes on the Docker instance
// @Tags Docker Volumes
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Success 200 {object} map[string]interface{} "Volume list response"
// @Failure 400 {object} map[string]interface{} "Invalid instance ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/volumes [get]
func (h *VolumeHandler) GetVolumes(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Forward request to agent
	req := &pb.ListVolumesRequest{}
	resp, err := h.agentForwarder.ListVolumes(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to list volumes: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}

// GetVolume gets details of a specific Docker volume
// @Summary Get Docker volume details
// @Description Get detailed information about a specific volume
// @Tags Docker Volumes
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param volume_name path string true "Volume name"
// @Success 200 {object} map[string]interface{} "Volume details"
// @Failure 400 {object} map[string]interface{} "Invalid parameters"
// @Failure 404 {object} map[string]interface{} "Volume not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/volumes/{volume_name} [get]
func (h *VolumeHandler) GetVolume(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Get volume name
	volumeName := c.Param("volume_name")
	if volumeName == "" {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("volume name is required"))
		return
	}

	// Forward request to agent
	req := &pb.GetVolumeRequest{
		Name: volumeName,
	}
	resp, err := h.agentForwarder.GetVolume(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to get volume: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}

// CreateVolumeRequest represents the request body for volume creation
type CreateVolumeRequest struct {
	Name       string            `json:"name" binding:"required"`
	Driver     string            `json:"driver"`
	DriverOpts map[string]string `json:"driver_opts"`
	Labels     map[string]string `json:"labels"`
}

// CreateVolume creates a new Docker volume
// @Summary Create Docker volume
// @Description Create a new volume on the Docker instance
// @Tags Docker Volumes
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body CreateVolumeRequest true "Volume creation parameters"
// @Success 200 {object} map[string]interface{} "Created volume details"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/volumes [post]
func (h *VolumeHandler) CreateVolume(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Parse request body
	var reqBody CreateVolumeRequest
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	// Forward request to agent
	req := &pb.CreateVolumeRequest{
		Name:       reqBody.Name,
		Driver:     reqBody.Driver,
		DriverOpts: reqBody.DriverOpts,
		Labels:     reqBody.Labels,
	}
	resp, err := h.agentForwarder.CreateVolume(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to create volume: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}

// DeleteVolumeRequest represents the request body for volume deletion
type DeleteVolumeRequest struct {
	Name  string `json:"name" binding:"required"`
	Force bool   `json:"force"`
}

// DeleteVolume deletes a Docker volume
// @Summary Delete Docker volume
// @Description Delete a volume from the Docker instance
// @Tags Docker Volumes
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body DeleteVolumeRequest true "Volume deletion parameters"
// @Success 200 {object} map[string]interface{} "Deletion result"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/volumes/delete [post]
func (h *VolumeHandler) DeleteVolume(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Parse request body
	var reqBody DeleteVolumeRequest
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	// Forward request to agent
	req := &pb.DeleteVolumeRequest{
		Name:  reqBody.Name,
		Force: reqBody.Force,
	}
	resp, err := h.agentForwarder.DeleteVolume(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to delete volume: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}

// PruneVolumesRequest represents the request body for volume pruning
type PruneVolumesRequest struct {
	Filters map[string]string `json:"filters"`
}

// PruneVolumes removes unused Docker volumes
// @Summary Prune unused Docker volumes
// @Description Remove all unused volumes from the Docker instance
// @Tags Docker Volumes
// @Accept json
// @Produce json
// @Param id path string true "Docker Instance ID (UUID)"
// @Param request body PruneVolumesRequest false "Prune filters (optional)"
// @Success 200 {object} map[string]interface{} "Prune result with space reclaimed"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/docker/instances/{id}/volumes/prune [post]
func (h *VolumeHandler) PruneVolumes(c *gin.Context) {
	// Parse instance ID
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		basehandlers.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid instance ID format"))
		return
	}

	// Parse request body (optional filters)
	var reqBody PruneVolumesRequest
	_ = c.ShouldBindJSON(&reqBody) // Ignore error, filters are optional

	// Forward request to agent
	req := &pb.PruneVolumesRequest{
		Filters: reqBody.Filters,
	}
	resp, err := h.agentForwarder.PruneVolumes(instanceID, req)
	if err != nil {
		basehandlers.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to prune volumes: %w", err))
		return
	}

	basehandlers.RespondSuccess(c, resp)
}
