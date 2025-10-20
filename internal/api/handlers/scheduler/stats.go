package scheduler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/api/handlers"
	schedulerrepo "github.com/ysicing/tiga/internal/repository/scheduler"
	schedulerservice "github.com/ysicing/tiga/internal/services/scheduler"
)

// StatsHandler handles scheduler statistics endpoints
// T022: Scheduler API handlers implementation
//
// Reference: .claude/specs/006-gitness-tiga/tasks.md T022
//           .claude/specs/006-gitness-tiga/contracts/scheduler_api.yaml
type StatsHandler struct {
	statsCalculator *schedulerservice.StatsCalculator
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(statsCalculator *schedulerservice.StatsCalculator) *StatsHandler {
	return &StatsHandler{
		statsCalculator: statsCalculator,
	}
}

// GetStats godoc
// @Summary Get statistics
// @Description Get statistics for all tasks including success rate, average execution time, and failure counts
// @Tags scheduler
// @Accept json
// @Produce json
// @Success 200 {object} object{data=object{total_tasks=int,enabled_tasks=int,total_executions=int,success_rate=float64,average_duration_ms=int64,task_stats=[]object}}
// @Failure 401 {object} handlers.ErrorResponse
// @Failure 500 {object} handlers.ErrorResponse
// @Router /scheduler/stats [get]
// @Security BearerAuth
func (h *StatsHandler) GetStats(c *gin.Context) {
	// Get global statistics
	stats, err := h.statsCalculator.GetGlobalStats(c.Request.Context())
	if err != nil {
		logrus.Errorf("Failed to get global stats: %v", err)
		handlers.RespondErrorWithMessage(c, http.StatusInternalServerError, err, "Failed to get statistics")
		return
	}

	// Build response matching OpenAPI schema
	response := gin.H{
		"total_tasks":         stats.TotalTasks,
		"enabled_tasks":       stats.EnabledTasks,
		"total_executions":    stats.TotalExecutions,
		"success_rate":        stats.SuccessRate,
		"average_duration_ms": stats.AverageDurationMs,
		"task_stats":          buildTaskStatsArray(stats.TaskStats),
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

// buildTaskStatsArray builds task stats array matching OpenAPI schema
func buildTaskStatsArray(taskStats []*schedulerrepo.TaskStats) []gin.H {
	result := make([]gin.H, 0, len(taskStats))

	// Convert array of TaskStats to array of objects
	for _, stats := range taskStats {
		result = append(result, gin.H{
			"task_name":           stats.TaskName,
			"total_executions":    stats.TotalExecutions,
			"success_executions":  stats.SuccessExecutions,
			"failure_executions":  stats.FailureExecutions,
			"average_duration_ms": stats.AverageDurationMs,
			"last_executed_at":    stats.LastExecutedAt,
		})
	}

	return result
}
