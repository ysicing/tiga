package instances

import (
	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/services"
)

// MetricsHandler handles instance metrics endpoints
type MetricsHandler struct {
	instanceService *services.InstanceService
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(instanceService *services.InstanceService) *MetricsHandler {
	return &MetricsHandler{
		instanceService: instanceService,
	}
}

// GetInstanceMetricsRequest represents a request to get instance metrics
type GetInstanceMetricsRequest struct {
	InstanceID string `uri:"instance_id" binding:"required,uuid"`
}

// GetInstanceMetrics retrieves metrics for an instance
// @Summary Get instance metrics
// @Description Get metrics for a specific instance
// @Tags instances
// @Produce json
// @Security BearerAuth
// @Param instance_id path string true "Instance ID (UUID)"
// @Success 200 {object} handlers.SuccessResponse{data=managers.ServiceMetrics}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/instances/{instance_id}/metrics [get]
func (h *MetricsHandler) GetInstanceMetrics(c *gin.Context) {
	var req GetInstanceMetricsRequest
	if !handlers.BindURI(c, &req) {
		return
	}

	instanceID, err := handlers.ParseUUID(req.InstanceID)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	metrics, err := h.instanceService.GetMetrics(c.Request.Context(), instanceID)
	if err != nil {
		if err.Error() == "failed to get instance: record not found" ||
			err.Error() == "record not found" {
			handlers.RespondNotFound(c, err)
			return
		}
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, metrics)
}
