package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// InstanceHandler handles instance management endpoints
type InstanceHandler struct {
	instanceRepo *repository.InstanceRepository
}

// NewInstanceHandler creates a new instance handler
func NewInstanceHandler(instanceRepo *repository.InstanceRepository) *InstanceHandler {
	return &InstanceHandler{
		instanceRepo: instanceRepo,
	}
}

// ListInstancesRequest represents a request to list instances
type ListInstancesRequest struct {
	ServiceType string   `form:"service_type"`
	Status      string   `form:"status"`
	Environment string   `form:"environment"`
	Tags        []string `form:"tags"`
	Search      string   `form:"search"`
	Page        int      `form:"page"`
	PageSize    int      `form:"page_size"`
}

// ListInstances lists all instances with pagination
// @Summary List instances
// @Description Get a paginated list of instances
// @Tags instances
// @Produce json
// @Security BearerAuth
// @Param service_type query string false "Filter by service type"
// @Param status query string false "Filter by status"
// @Param environment query string false "Filter by environment"
// @Param tags query array false "Filter by tags"
// @Param search query string false "Search in name, host, description"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} PaginatedResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/instances [get]
func (h *InstanceHandler) ListInstances(c *gin.Context) {
	var req ListInstancesRequest
	if !BindQuery(c, &req) {
		return
	}

	// Set defaults
	req.Page = defaultInt(req.Page, 1)
	req.PageSize = defaultInt(req.PageSize, 20)
	req.PageSize = clamp(req.PageSize, 1, 100)

	// Build filter
	filter := &repository.ListInstancesFilter{
		ServiceType: req.ServiceType,
		Status:      req.Status,
		Environment: req.Environment,
		Tags:        req.Tags,
		Search:      req.Search,
		Page:        req.Page,
		PageSize:    req.PageSize,
	}

	// Get instances
	instances, total, err := h.instanceRepo.ListInstances(c.Request.Context(), filter)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondPaginated(c, instances, req.Page, req.PageSize, total)
}

// GetInstanceRequest represents a request to get an instance
type GetInstanceRequest struct {
	InstanceID string `uri:"instance_id" binding:"required,uuid"`
}

// GetInstance gets an instance by ID
// @Summary Get instance
// @Description Get instance details by ID
// @Tags instances
// @Produce json
// @Security BearerAuth
// @Param instance_id path string true "Instance ID (UUID)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/instances/{instance_id} [get]
func (h *InstanceHandler) GetInstance(c *gin.Context) {
	var req GetInstanceRequest
	if !BindURI(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		RespondNotFound(c, err)
		return
	}

	RespondSuccess(c, instance)
}

// CreateInstanceRequest represents a request to create an instance
type CreateInstanceRequest struct {
	Name        string                 `json:"name" binding:"required"`
	ServiceType string                 `json:"service_type" binding:"required"`
	Host        string                 `json:"host" binding:"required"`
	Port        int                    `json:"port" binding:"required,min=1,max=65535"`
	Version     string                 `json:"version"`
	Environment string                 `json:"environment" binding:"oneof=dev test staging production"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Tags        []string               `json:"tags"`
}

// CreateInstance creates a new instance
// @Summary Create instance
// @Description Create a new service instance
// @Tags instances
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateInstanceRequest true "Instance creation request"
// @Success 201 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /api/v1/instances [post]
func (h *InstanceHandler) CreateInstance(c *gin.Context) {
	var req CreateInstanceRequest
	if !BindJSON(c, &req) {
		return
	}

	// Check if instance name exists
	exists, err := h.instanceRepo.ExistsName(c.Request.Context(), req.Name, nil)
	if err != nil {
		RespondInternalError(c, err)
		return
	}
	if exists {
		RespondConflict(c, fmt.Errorf("instance name already exists"))
		return
	}

	// Build connection JSON
	connection := map[string]interface{}{
		"host": req.Host,
		"port": req.Port,
	}

	// Create instance
	instance := &models.Instance{
		Name:        req.Name,
		Type:        req.ServiceType,
		Connection:  connection,
		Version:     req.Version,
		Environment: defaultIfEmpty(req.Environment, "production"),
		Description: req.Description,
		Config:      req.Config,
		Tags:        req.Tags,
		Status:      "pending",
		Health:      "unknown",
	}

	if err := h.instanceRepo.Create(c.Request.Context(), instance); err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondCreated(c, instance)
}

// UpdateInstanceRequest represents a request to update an instance
type UpdateInstanceRequest struct {
	InstanceID  string                 `uri:"instance_id" binding:"required,uuid"`
	Name        *string                `json:"name,omitempty"`
	Host        *string                `json:"host,omitempty"`
	Port        *int                   `json:"port,omitempty" binding:"omitempty,min=1,max=65535"`
	Version     *string                `json:"version,omitempty"`
	Environment *string                `json:"environment,omitempty" binding:"omitempty,oneof=dev test staging production"`
	Description *string                `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}

// UpdateInstance updates an instance
// @Summary Update instance
// @Description Update instance details
// @Tags instances
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param instance_id path string true "Instance ID (UUID)"
// @Param request body UpdateInstanceRequest true "Instance update request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/instances/{instance_id} [patch]
func (h *InstanceHandler) UpdateInstance(c *gin.Context) {
	var req UpdateInstanceRequest
	if !BindURI(c, &req) {
		return
	}
	if !BindJSON(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	// Build updates
	updates := make(map[string]interface{})
	if req.Name != nil {
		// Check name uniqueness
		exists, err := h.instanceRepo.ExistsName(c.Request.Context(), *req.Name, &instanceID)
		if err != nil {
			RespondInternalError(c, err)
			return
		}
		if exists {
			RespondConflict(c, fmt.Errorf("instance name already exists"))
			return
		}
		updates["name"] = *req.Name
	}
	if req.Host != nil {
		updates["host"] = *req.Host
	}
	if req.Port != nil {
		updates["port"] = *req.Port
	}
	if req.Version != nil {
		updates["version"] = *req.Version
	}
	if req.Environment != nil {
		updates["environment"] = *req.Environment
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Config != nil {
		updates["config"] = req.Config
	}
	if req.Tags != nil {
		updates["tags"] = req.Tags
	}

	// Update instance
	if len(updates) > 0 {
		if err := h.instanceRepo.UpdateFields(c.Request.Context(), instanceID, updates); err != nil {
			RespondInternalError(c, err)
			return
		}
	}

	// Get updated instance
	instance, err := h.instanceRepo.GetByID(c.Request.Context(), instanceID)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, instance)
}

// DeleteInstance deletes an instance
// @Summary Delete instance
// @Description Soft delete an instance
// @Tags instances
// @Produce json
// @Security BearerAuth
// @Param instance_id path string true "Instance ID (UUID)"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/instances/{instance_id} [delete]
func (h *InstanceHandler) DeleteInstance(c *gin.Context) {
	var req GetInstanceRequest
	if !BindURI(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	if err := h.instanceRepo.Delete(c.Request.Context(), instanceID); err != nil {
		RespondNotFound(c, err)
		return
	}

	RespondNoContent(c)
}

// UpdateInstanceStatusRequest represents a request to update instance status
type UpdateInstanceStatusRequest struct {
	InstanceID string `uri:"instance_id" binding:"required,uuid"`
	Status     string `json:"status" binding:"required,oneof=pending running stopped failed"`
}

// UpdateInstanceStatus updates an instance status
// @Summary Update instance status
// @Description Update instance operational status
// @Tags instances
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param instance_id path string true "Instance ID (UUID)"
// @Param request body UpdateInstanceStatusRequest true "Status update request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/instances/{instance_id}/status [patch]
func (h *InstanceHandler) UpdateInstanceStatus(c *gin.Context) {
	var req UpdateInstanceStatusRequest
	if !BindURI(c, &req) {
		return
	}
	if !BindJSON(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	if err := h.instanceRepo.UpdateStatus(c.Request.Context(), instanceID, req.Status); err != nil {
		RespondNotFound(c, err)
		return
	}

	RespondSuccessWithMessage(c, nil, "instance status updated")
}

// UpdateInstanceHealthRequest represents a request to update instance health
type UpdateInstanceHealthRequest struct {
	InstanceID    string  `uri:"instance_id" binding:"required,uuid"`
	Health        string  `json:"health" binding:"required,oneof=healthy unhealthy degraded unknown"`
	HealthMessage *string `json:"health_message,omitempty"`
}

// UpdateInstanceHealth updates an instance health status
// @Summary Update instance health
// @Description Update instance health status
// @Tags instances
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param instance_id path string true "Instance ID (UUID)"
// @Param request body UpdateInstanceHealthRequest true "Health update request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/instances/{instance_id}/health [patch]
func (h *InstanceHandler) UpdateInstanceHealth(c *gin.Context) {
	var req UpdateInstanceHealthRequest
	if !BindURI(c, &req) {
		return
	}
	if !BindJSON(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	if err := h.instanceRepo.UpdateHealth(c.Request.Context(), instanceID, req.Health, req.HealthMessage); err != nil {
		RespondNotFound(c, err)
		return
	}

	RespondSuccessWithMessage(c, nil, "instance health updated")
}

// GetInstanceStatistics gets instance statistics
// @Summary Get instance statistics
// @Description Get overall instance statistics
// @Tags instances
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/instances/statistics [get]
func (h *InstanceHandler) GetInstanceStatistics(c *gin.Context) {
	stats, err := h.instanceRepo.GetStatistics(c.Request.Context())
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, stats)
}

// ManageTagsRequest represents a request to manage instance tags
type ManageTagsRequest struct {
	InstanceID string   `uri:"instance_id" binding:"required,uuid"`
	Tags       []string `json:"tags" binding:"required"`
}

// AddTags adds tags to an instance
// @Summary Add tags
// @Description Add tags to an instance
// @Tags instances
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param instance_id path string true "Instance ID (UUID)"
// @Param request body ManageTagsRequest true "Tags to add"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/instances/{instance_id}/tags [post]
func (h *InstanceHandler) AddTags(c *gin.Context) {
	var req ManageTagsRequest
	if !BindURI(c, &req) {
		return
	}
	if !BindJSON(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	if err := h.instanceRepo.AddTags(c.Request.Context(), instanceID, req.Tags); err != nil {
		RespondBadRequest(c, err)
		return
	}

	RespondSuccessWithMessage(c, nil, "tags added successfully")
}

// RemoveTags removes tags from an instance
// @Summary Remove tags
// @Description Remove tags from an instance
// @Tags instances
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param instance_id path string true "Instance ID (UUID)"
// @Param request body ManageTagsRequest true "Tags to remove"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/instances/{instance_id}/tags [delete]
func (h *InstanceHandler) RemoveTags(c *gin.Context) {
	var req ManageTagsRequest
	if !BindURI(c, &req) {
		return
	}
	if !BindJSON(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	if err := h.instanceRepo.RemoveTags(c.Request.Context(), instanceID, req.Tags); err != nil {
		RespondBadRequest(c, err)
		return
	}

	RespondSuccessWithMessage(c, nil, "tags removed successfully")
}
