package recording

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/recording"
)

// RecordingHandler handles terminal recording API requests
type RecordingHandler struct {
	managerService *recording.ManagerService
}

// NewRecordingHandler creates a new recording handler instance
func NewRecordingHandler(managerService *recording.ManagerService) *RecordingHandler {
	return &RecordingHandler{
		managerService: managerService,
	}
}

// ListRecordingsRequest defines query parameters for listing recordings
type ListRecordingsRequest struct {
	Page          int        `form:"page" binding:"omitempty,min=1"`
	Limit         int        `form:"limit" binding:"omitempty,min=1,max=100"`
	UserID        *uuid.UUID `form:"user_id" binding:"omitempty,uuid"`
	RecordingType *string    `form:"recording_type" binding:"omitempty,oneof=docker webssh k8s_node k8s_pod"`
	StorageType   *string    `form:"storage_type" binding:"omitempty,oneof=local minio"`
	StartTime     *time.Time `form:"start_time" binding:"omitempty" time_format:"2006-01-02T15:04:05Z07:00"`
	EndTime       *time.Time `form:"end_time" binding:"omitempty" time_format:"2006-01-02T15:04:05Z07:00"`
	SortBy        string     `form:"sort_by" binding:"omitempty,oneof=started_at ended_at file_size duration"`
	SortOrder     string     `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// ListRecordingsResponse defines the response for listing recordings
type ListRecordingsResponse struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
}

// ListRecordings retrieves a paginated list of terminal recordings
// @Summary List terminal recordings
// @Description Get a paginated list of terminal recordings with optional filtering
// @Tags recordings
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Param user_id query string false "Filter by user ID (UUID)"
// @Param recording_type query string false "Filter by recording type (docker, webssh, k8s_node, k8s_pod)"
// @Param storage_type query string false "Filter by storage type (local, minio)"
// @Param start_time query string false "Filter by start time (RFC3339 format)"
// @Param end_time query string false "Filter by end time (RFC3339 format)"
// @Param sort_by query string false "Sort field (started_at, ended_at, file_size, duration)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {object} ListRecordingsResponse
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/recordings [get]
// @Security BearerAuth
func (h *RecordingHandler) ListRecordings(c *gin.Context) {
	var req ListRecordingsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Set defaults
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 20
	}

	// Build filters
	filters := repository.RecordingFilters{
		UserID:        req.UserID,
		RecordingType: req.RecordingType,
		StorageType:   req.StorageType,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		SortBy:        req.SortBy,
		SortOrder:     req.SortOrder,
	}

	// Call service
	recordings, total, err := h.managerService.ListRecordings(c.Request.Context(), filters, req.Page, req.Limit)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	// Calculate total pages
	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	// Convert to metadata
	items := make([]interface{}, 0, len(recordings))
	for _, r := range recordings {
		items = append(items, r.ToMetadata())
	}

	handlers.RespondSuccess(c, ListRecordingsResponse{
		Items:      items,
		Total:      total,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	})
}

// GetRecording retrieves a single terminal recording by ID
// @Summary Get terminal recording
// @Description Get detailed information about a terminal recording
// @Tags recordings
// @Accept json
// @Produce json
// @Param id path string true "Recording ID (UUID)"
// @Success 200 {object} models.RecordingMetadata
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/recordings/{id} [get]
// @Security BearerAuth
func (h *RecordingHandler) GetRecording(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	recording, err := h.managerService.GetRecording(c.Request.Context(), id)
	if err != nil {
		handlers.RespondNotFound(c, err)
		return
	}

	handlers.RespondSuccess(c, recording.ToMetadata())
}

// DeleteRecording deletes a terminal recording and its associated file
// @Summary Delete terminal recording
// @Description Delete a terminal recording and its associated file
// @Tags recordings
// @Accept json
// @Produce json
// @Param id path string true "Recording ID (UUID)"
// @Success 200 {object} handlers.SuccessResponse
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 403 {object} handlers.ErrorResponse "Forbidden - insufficient permissions"
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/recordings/{id} [delete]
// @Security BearerAuth
func (h *RecordingHandler) DeleteRecording(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// TODO: RBAC permission check
	// Check if user has permission to delete this recording
	// For now, assume all authenticated users can delete their own recordings

	if err := h.managerService.DeleteRecording(c.Request.Context(), id); err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, gin.H{
		"message": "Recording deleted successfully",
	})
}

// SearchRecordingsRequest defines query parameters for searching recordings
type SearchRecordingsRequest struct {
	Query string `form:"q" binding:"required,min=1"`
	Page  int    `form:"page" binding:"omitempty,min=1"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

// SearchRecordings performs full-text search on terminal recordings
// @Summary Search terminal recordings
// @Description Search recordings by username, description, or tags
// @Tags recordings
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} ListRecordingsResponse
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/recordings/search [get]
// @Security BearerAuth
func (h *RecordingHandler) SearchRecordings(c *gin.Context) {
	var req SearchRecordingsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		handlers.RespondBadRequest(c, err)
		return
	}

	// Set defaults
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Limit == 0 {
		req.Limit = 20
	}

	// Call service
	recordings, total, err := h.managerService.SearchRecordings(c.Request.Context(), req.Query, req.Page, req.Limit)
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	// Calculate total pages
	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	// Convert to metadata
	items := make([]interface{}, 0, len(recordings))
	for _, r := range recordings {
		items = append(items, r.ToMetadata())
	}

	handlers.RespondSuccess(c, ListRecordingsResponse{
		Items:      items,
		Total:      total,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	})
}

// GetStatistics retrieves aggregated statistics about terminal recordings
// @Summary Get recording statistics
// @Description Get aggregated statistics about terminal recordings (total count, size, by type, top users)
// @Tags recordings
// @Accept json
// @Produce json
// @Success 200 {object} repository.RecordingStatistics
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/recordings/statistics [get]
// @Security BearerAuth
func (h *RecordingHandler) GetStatistics(c *gin.Context) {
	stats, err := h.managerService.GetStatistics(c.Request.Context())
	if err != nil {
		handlers.RespondInternalError(c, err)
		return
	}

	handlers.RespondSuccess(c, stats)
}
