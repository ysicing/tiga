package scheduler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/models"

	schedulerrepo "github.com/ysicing/tiga/internal/repository/scheduler"
)

// ExecutionHandler handles scheduler execution endpoints
// T022: Scheduler API handlers implementation
//
// Reference: .claude/specs/006-gitness-tiga/tasks.md T022
//
//	.claude/specs/006-gitness-tiga/contracts/scheduler_api.yaml
type ExecutionHandler struct {
	execRepo schedulerrepo.ExecutionRepository
}

// NewExecutionHandler creates a new execution handler
func NewExecutionHandler(execRepo schedulerrepo.ExecutionRepository) *ExecutionHandler {
	return &ExecutionHandler{
		execRepo: execRepo,
	}
}

// ListExecutions godoc
// @Summary Get execution history
// @Description Get paginated list of task execution history with filtering by task name, state, and time range
// @Tags scheduler
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param page_size query int false "Page size (default: 20)" minimum(1) maximum(100)
// @Param task_name query string false "Filter by task name"
// @Param state query string false "Filter by execution state" Enums(pending, running, success, failure, timeout, cancelled, interrupted)
// @Param start_time query int false "Start time (Unix milliseconds)"
// @Param end_time query int false "End time (Unix milliseconds)"
// @Success 200 {object} object{data=[]models.TaskExecution,pagination=handlers.Pagination}
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /scheduler/executions [get]
// @Security BearerAuth
func (h *ExecutionHandler) ListExecutions(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Build filters
	filters := make(map[string]interface{})

	if taskName := c.Query("task_name"); taskName != "" {
		filters["task_name"] = taskName
	}

	if state := c.Query("state"); state != "" {
		// Validate state enum
		validStates := []models.ExecutionState{
			models.ExecutionStatePending,
			models.ExecutionStateRunning,
			models.ExecutionStateSuccess,
			models.ExecutionStateFailure,
			models.ExecutionStateTimeout,
			models.ExecutionStateCancelled,
			models.ExecutionStateInterrupted,
		}

		isValid := false
		for _, validState := range validStates {
			if string(validState) == state {
				filters["state"] = models.ExecutionState(state)
				isValid = true
				break
			}
		}

		if !isValid {
			handlers.RespondErrorWithMessage(c, http.StatusBadRequest, nil, "Invalid state value")
			return
		}
	}

	// Parse time range filters
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTimeMs, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			handlers.RespondErrorWithMessage(c, http.StatusBadRequest, err, "Invalid start_time format")
			return
		}
		filters["start_time"] = time.UnixMilli(startTimeMs)
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTimeMs, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			handlers.RespondErrorWithMessage(c, http.StatusBadRequest, err, "Invalid end_time format")
			return
		}
		filters["end_time"] = time.UnixMilli(endTimeMs)
	}

	// Query executions
	executions, err := h.execRepo.List(c.Request.Context(), filters, pageSize, offset)
	if err != nil {
		logrus.Errorf("Failed to list executions: %v", err)
		handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, err, "Failed to list executions")
		return
	}

	// Get total count
	total, err := h.execRepo.Count(c.Request.Context(), filters)
	if err != nil {
		logrus.Errorf("Failed to count executions: %v", err)
		handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, err, "Failed to count executions")
		return
	}

	// Build pagination metadata
	totalPages := (int(total) + pageSize - 1) / pageSize
	pagination := handlers.Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       executions,
		"pagination": pagination,
	})
}

// GetExecution godoc
// @Summary Get execution details
// @Description Get details of a specific task execution by ID
// @Tags scheduler
// @Accept json
// @Produce json
// @Param id path int true "Execution ID"
// @Success 200 {object} object{data=models.TaskExecution}
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Router /scheduler/executions/{id} [get]
// @Security BearerAuth
func (h *ExecutionHandler) GetExecution(c *gin.Context) {
	// Parse execution ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		handlers.RespondErrorWithMessage(c, http.StatusBadRequest, err, "Invalid execution ID")
		return
	}

	// Get execution
	execution, err := h.execRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		logrus.Errorf("Failed to get execution %d: %v", id, err)
		handlers.RespondErrorWithMessage(c, http.StatusNotFound, err, "Execution not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": execution,
	})
}
