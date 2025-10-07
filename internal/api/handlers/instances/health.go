package instances

import (
	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/services"
)

// HealthHandler handles instance health check endpoints
type HealthHandler struct {
	instanceService *services.InstanceService
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(instanceService *services.InstanceService) *HealthHandler {
	return &HealthHandler{
		instanceService: instanceService,
	}
}

// GetInstanceHealthRequest represents a request to get instance health
type GetInstanceHealthRequest struct {
	InstanceID string `uri:"instance_id" binding:"required,uuid"`
}

// GetInstanceHealth retrieves health status for an instance
// @Summary Get instance health
// @Description Get health status for a specific instance
// @Tags instances
// @Produce json
// @Security BearerAuth
// @Param instance_id path string true "Instance ID (UUID)"
// @Success 200 {object} handlers.SuccessResponse{data=managers.HealthStatus}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/instances/{instance_id}/health [get]
func (h *HealthHandler) GetInstanceHealth(c *gin.Context) {
	var req GetInstanceHealthRequest
	if !handlers.BindURI(c, &req) {
		return
	}

	instanceID, err := handlers.ParseUUID(req.InstanceID)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	health, err := h.instanceService.GetHealthStatus(c.Request.Context(), instanceID)
	if err != nil {
		if err.Error() == "failed to get instance: record not found" ||
			err.Error() == "record not found" {
			handlers.RespondNotFound(c, err)
			return
		}
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, health)
}
