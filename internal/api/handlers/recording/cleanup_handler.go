package recording

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/handlers"
	"github.com/ysicing/tiga/internal/services/recording"
)

// CleanupHandler handles recording cleanup API requests
type CleanupHandler struct {
	cleanupService *recording.CleanupService

	// Track cleanup task status
	mu                sync.RWMutex
	lastRunAt         *time.Time
	lastRunDuration   time.Duration
	lastRunStats      *CleanupStats
	isRunning         bool
	currentTaskID     *uuid.UUID
}

// CleanupStats contains statistics from the last cleanup run
type CleanupStats struct {
	InvalidDeleted int `json:"invalid_deleted"`
	ExpiredDeleted int `json:"expired_deleted"`
	OrphanDeleted  int `json:"orphan_deleted"`
	TotalDeleted   int `json:"total_deleted"`
}

// CleanupStatusResponse defines the response for cleanup status
type CleanupStatusResponse struct {
	IsRunning       bool          `json:"is_running"`
	LastRunAt       *time.Time    `json:"last_run_at,omitempty"`
	LastRunDuration string        `json:"last_run_duration,omitempty"`
	LastRunStats    *CleanupStats `json:"last_run_stats,omitempty"`
	CurrentTaskID   *uuid.UUID    `json:"current_task_id,omitempty"`
}

// TriggerCleanupResponse defines the response for cleanup trigger
type TriggerCleanupResponse struct {
	TaskID  uuid.UUID `json:"task_id"`
	Message string    `json:"message"`
}

// NewCleanupHandler creates a new cleanup handler instance
func NewCleanupHandler(cleanupService *recording.CleanupService) *CleanupHandler {
	return &CleanupHandler{
		cleanupService: cleanupService,
	}
}

// TriggerCleanup manually triggers the cleanup task
// @Summary Trigger cleanup task
// @Description Manually trigger the recording cleanup task (admin only)
// @Tags recordings
// @Accept json
// @Produce json
// @Success 202 {object} TriggerCleanupResponse "Cleanup task started"
// @Failure 403 {object} handlers.ErrorResponse "Forbidden - admin only"
// @Failure 409 {object} handlers.ErrorResponse "Cleanup already running"
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/recordings/cleanup/trigger [post]
// @Security BearerAuth
func (h *CleanupHandler) TriggerCleanup(c *gin.Context) {
	// TODO: RBAC check - admin only
	// For now, allow all authenticated users
	// userRole := c.GetString("user_role")
	// if userRole != "admin" {
	// 	handlers.SendError(c, http.StatusForbidden, "Admin permission required", nil)
	// 	return
	// }

	// Check if cleanup is already running
	h.mu.RLock()
	if h.isRunning {
		h.mu.RUnlock()
		c.JSON(http.StatusConflict, gin.H{"error": "Cleanup task is already running"})
		return
	}
	h.mu.RUnlock()

	// Generate task ID
	taskID := uuid.New()

	// Mark as running
	h.mu.Lock()
	h.isRunning = true
	h.currentTaskID = &taskID
	h.mu.Unlock()

	// Start cleanup in background goroutine
	go h.runCleanup(taskID)

	// Return immediately with task ID
	c.JSON(http.StatusAccepted, TriggerCleanupResponse{
		TaskID:  taskID,
		Message: "Cleanup task started",
	})
}

// runCleanup executes the cleanup task in background
func (h *CleanupHandler) runCleanup(taskID uuid.UUID) {
	defer func() {
		h.mu.Lock()
		h.isRunning = false
		h.currentTaskID = nil
		h.mu.Unlock()
	}()

	ctx := context.Background()
	startTime := time.Now()

	logrus.Infof("[CleanupHandler] Starting manual cleanup task %s", taskID)

	// Execute cleanup
	stats := &CleanupStats{}
	var err error

	stats.InvalidDeleted, err = h.cleanupService.CleanupInvalidRecordings(ctx)
	if err != nil {
		logrus.Errorf("[CleanupHandler] Invalid recordings cleanup failed: %v", err)
	}

	stats.ExpiredDeleted, err = h.cleanupService.CleanupExpiredRecordings(ctx)
	if err != nil {
		logrus.Errorf("[CleanupHandler] Expired recordings cleanup failed: %v", err)
	}

	stats.OrphanDeleted, err = h.cleanupService.CleanupOrphanFiles(ctx)
	if err != nil {
		logrus.Errorf("[CleanupHandler] Orphan files cleanup failed: %v", err)
	}

	stats.TotalDeleted = stats.InvalidDeleted + stats.ExpiredDeleted + stats.OrphanDeleted

	// Update status
	duration := time.Since(startTime)
	now := time.Now()

	h.mu.Lock()
	h.lastRunAt = &now
	h.lastRunDuration = duration
	h.lastRunStats = stats
	h.mu.Unlock()

	logrus.Infof("[CleanupHandler] Manual cleanup task %s completed in %v: invalid=%d, expired=%d, orphan=%d, total=%d",
		taskID, duration, stats.InvalidDeleted, stats.ExpiredDeleted, stats.OrphanDeleted, stats.TotalDeleted)
}

// GetCleanupStatus retrieves the current cleanup task status
// @Summary Get cleanup status
// @Description Get the current status of the cleanup task
// @Tags recordings
// @Accept json
// @Produce json
// @Success 200 {object} CleanupStatusResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /api/v1/recordings/cleanup/status [get]
// @Security BearerAuth
func (h *CleanupHandler) GetCleanupStatus(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	response := CleanupStatusResponse{
		IsRunning:     h.isRunning,
		LastRunAt:     h.lastRunAt,
		LastRunStats:  h.lastRunStats,
		CurrentTaskID: h.currentTaskID,
	}

	if h.lastRunDuration > 0 {
		response.LastRunDuration = h.lastRunDuration.String()
	}

	handlers.RespondSuccess(c, response)
}

// UpdateLastRun updates the last run statistics (called by scheduler)
func (h *CleanupHandler) UpdateLastRun(startTime time.Time, stats *CleanupStats) {
	duration := time.Since(startTime)
	now := time.Now()

	h.mu.Lock()
	h.lastRunAt = &now
	h.lastRunDuration = duration
	h.lastRunStats = stats
	h.mu.Unlock()
}
