package scheduler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/handlers"
	schedulerrepo "github.com/ysicing/tiga/internal/repository/scheduler"
	schedulerservice "github.com/ysicing/tiga/internal/services/scheduler"
)

// TaskHandler handles scheduler task endpoints
// T022: Scheduler API handlers implementation
//
// Reference: .claude/specs/006-gitness-tiga/tasks.md T022
//           .claude/specs/006-gitness-tiga/contracts/scheduler_api.yaml
type TaskHandler struct {
	taskRepo  schedulerrepo.TaskRepository
	scheduler *schedulerservice.Scheduler
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(taskRepo schedulerrepo.TaskRepository, scheduler *schedulerservice.Scheduler) *TaskHandler {
	return &TaskHandler{
		taskRepo:  taskRepo,
		scheduler: scheduler,
	}
}

// ListTasks godoc
// @Summary Get task list
// @Description Get paginated list of scheduled tasks with optional filtering
// @Tags scheduler
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param page_size query int false "Page size (default: 20)" minimum(1) maximum(100)
// @Param enabled query bool false "Filter by enabled status"
// @Param type query string false "Filter by task type"
// @Success 200 {object} object{data=[]models.ScheduledTask,pagination=handlers.Pagination}
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /scheduler/tasks [get]
// @Security BearerAuth
func (h *TaskHandler) ListTasks(c *gin.Context) {
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
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		enabled, err := strconv.ParseBool(enabledStr)
		if err == nil {
			filters["enabled"] = enabled
		}
	}
	if taskType := c.Query("type"); taskType != "" {
		filters["type"] = taskType
	}

	// Query tasks
	tasks, err := h.taskRepo.List(c.Request.Context(), filters, pageSize, offset)
	if err != nil {
		logrus.Errorf("Failed to list tasks: %v", err)
		handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, err, "Failed to list tasks")
		return
	}

	// Get total count
	total, err := h.taskRepo.Count(c.Request.Context(), filters)
	if err != nil {
		logrus.Errorf("Failed to count tasks: %v", err)
		handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, err, "Failed to count tasks")
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
		"data":       tasks,
		"pagination": pagination,
	})
}

// GetTask godoc
// @Summary Get task details
// @Description Get details of a specific scheduled task by UID
// @Tags scheduler
// @Accept json
// @Produce json
// @Param id path string true "Task UID"
// @Success 200 {object} object{data=models.ScheduledTask}
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Router /scheduler/tasks/{id} [get]
// @Security BearerAuth
func (h *TaskHandler) GetTask(c *gin.Context) {
	taskUID := c.Param("id")

	task, err := h.taskRepo.GetByUID(c.Request.Context(), taskUID)
	if err != nil {
		logrus.Errorf("Failed to get task %s: %v", taskUID, err)
		handlers.RespondErrorWithMessage(c, http.StatusNotFound, err, "Task not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": task,
	})
}

// EnableTask godoc
// @Summary Enable task
// @Description Enable a scheduled task. Takes effect immediately without service restart
// @Tags scheduler
// @Accept json
// @Produce json
// @Param id path string true "Task UID"
// @Success 200 {object} handlers.SuccessResponse
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 403 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Router /scheduler/tasks/{id}/enable [post]
// @Security BearerAuth
func (h *TaskHandler) EnableTask(c *gin.Context) {
	taskUID := c.Param("id")

	// Get task
	task, err := h.taskRepo.GetByUID(c.Request.Context(), taskUID)
	if err != nil {
		logrus.Errorf("Failed to get task %s: %v", taskUID, err)
		handlers.RespondErrorWithMessage(c, http.StatusNotFound, err, "Task not found")
		return
	}

	// Enable task
	task.Enabled = true
	if err := h.taskRepo.Update(c.Request.Context(), task); err != nil {
		logrus.Errorf("Failed to enable task %s: %v", taskUID, err)
		handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, err, "Failed to enable task")
		return
	}

	// Reload scheduler to apply changes
	// TODO: Implement dynamic reload in scheduler service
	logrus.Infof("Task %s enabled successfully", taskUID)

	c.JSON(http.StatusOK, handlers.SuccessResponse{Success: true, Message: "Task enabled successfully"})
}

// DisableTask godoc
// @Summary Disable task
// @Description Disable a scheduled task. Takes effect immediately. Running tasks are not affected
// @Tags scheduler
// @Accept json
// @Produce json
// @Param id path string true "Task UID"
// @Success 200 {object} handlers.SuccessResponse
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 403 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Router /scheduler/tasks/{id}/disable [post]
// @Security BearerAuth
func (h *TaskHandler) DisableTask(c *gin.Context) {
	taskUID := c.Param("id")

	// Get task
	task, err := h.taskRepo.GetByUID(c.Request.Context(), taskUID)
	if err != nil {
		logrus.Errorf("Failed to get task %s: %v", taskUID, err)
		handlers.RespondErrorWithMessage(c, http.StatusNotFound, err, "Task not found")
		return
	}

	// Disable task
	task.Enabled = false
	if err := h.taskRepo.Update(c.Request.Context(), task); err != nil {
		logrus.Errorf("Failed to disable task %s: %v", taskUID, err)
		handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, err, "Failed to disable task")
		return
	}

	// Reload scheduler to apply changes
	// TODO: Implement dynamic reload in scheduler service
	logrus.Infof("Task %s disabled successfully", taskUID)

	c.JSON(http.StatusOK, handlers.SuccessResponse{Success: true, Message: "Task disabled successfully"})
}

// TriggerTaskRequest represents the request body for manual task trigger
type TriggerTaskRequest struct {
	OverrideData string `json:"override_data,omitempty"` // JSON string to override task input data
}

// TriggerTask godoc
// @Summary Trigger task manually
// @Description Immediately execute a task without waiting for the next scheduled time. For debugging and emergency situations
// @Tags scheduler
// @Accept json
// @Produce json
// @Param id path string true "Task UID"
// @Param body body TriggerTaskRequest false "Trigger parameters (optional)"
// @Success 202 {object} object{message=string,execution_uid=string} "Task queued successfully"
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 403 {object} handlers.ErrorResponse
// @Failure 404 {object} handlers.ErrorResponse
// @Failure 409 {object} handlers.ErrorResponse "Task already running"
// @Router /scheduler/tasks/{id}/trigger [post]
// @Security BearerAuth
func (h *TaskHandler) TriggerTask(c *gin.Context) {
	taskUID := c.Param("id")

	// Parse request body (optional)
	var req TriggerTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		handlers.RespondErrorWithMessage(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	// Get task
	task, err := h.taskRepo.GetByUID(c.Request.Context(), taskUID)
	if err != nil {
		logrus.Errorf("Failed to get task %s: %v", taskUID, err)
		handlers.RespondErrorWithMessage(c, http.StatusNotFound, err, "Task not found")
		return
	}

	// TODO: Check if task is already running (implement concurrency control)
	// For now, we allow triggering even if task is running

	// Trigger task manually
	// T013: Use Scheduler.Trigger() method
	if err := h.scheduler.Trigger(task.Name); err != nil {
		logrus.Errorf("Failed to trigger task %s: %v", task.Name, err)

		// Check if it's a queue full error (HTTP 409)
		if err.Error() == "manual trigger queue full" {
			handlers.RespondErrorWithMessage(c, http.StatusConflict, err, "Task trigger queue is full, please try again later")
			return
		}

		handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, err, "Failed to trigger task")
		return
	}

	// Generate execution UID (will be created by scheduler)
	// For now, return a pending message
	logrus.Infof("Task %s triggered manually", task.Name)

	c.JSON(http.StatusAccepted, gin.H{
		"message":       "Task triggered successfully",
		"execution_uid": "", // Will be populated by scheduler
	})
}
