package scheduler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	schedulerrepo "github.com/ysicing/tiga/internal/repository/scheduler"
)

// StatsCalculator handles task statistics calculation
// T016: Task statistics calculation service
//
// Reference: .claude/specs/006-gitness-tiga/tasks.md T016
//           .claude/specs/006-gitness-tiga/contracts/scheduler_api.yaml /stats endpoint
type StatsCalculator struct {
	repo schedulerrepo.ExecutionRepository
}

// NewStatsCalculator creates a new stats calculator
func NewStatsCalculator(repo schedulerrepo.ExecutionRepository) *StatsCalculator {
	return &StatsCalculator{
		repo: repo,
	}
}

// GetTaskStats retrieves statistics for a specific task
// T016: Single task statistics (success rate, average execution time, etc.)
//
// Returns:
// - TaskUID: Unique task identifier
// - TaskName: Task name
// - TotalExecutions: Total number of executions
// - SuccessExecutions: Number of successful executions
// - FailureExecutions: Number of failed executions
// - TimeoutExecutions: Number of timed out executions
// - AverageDurationMs: Average execution duration in milliseconds
// - LastExecutedAt: Last execution timestamp
// - LastExecutionState: Last execution state
func (sc *StatsCalculator) GetTaskStats(ctx context.Context, taskUID string) (*schedulerrepo.TaskStats, error) {
	stats, err := sc.repo.GetStats(ctx, taskUID)
	if err != nil {
		logrus.Errorf("Failed to get task stats for %s: %v", taskUID, err)
		return nil, fmt.Errorf("failed to get task stats: %w", err)
	}

	logrus.Debugf("Task %s stats: total=%d, success=%d, failure=%d, avg_duration=%dms",
		taskUID, stats.TotalExecutions, stats.SuccessExecutions, stats.FailureExecutions, stats.AverageDurationMs)

	return stats, nil
}

// GetGlobalStats retrieves global statistics across all tasks
// T016: Global statistics aggregation
//
// Returns:
// - TotalTasks: Total number of unique tasks
// - EnabledTasks: Number of enabled tasks
// - TotalExecutions: Total number of executions across all tasks
// - SuccessExecutions: Total successful executions
// - FailureExecutions: Total failed executions (includes timeout)
// - SuccessRate: Overall success rate percentage
// - AverageDurationMs: Average execution duration across all successful tasks
// - TaskStats: Detailed stats for each task
func (sc *StatsCalculator) GetGlobalStats(ctx context.Context) (*schedulerrepo.GlobalStats, error) {
	stats, err := sc.repo.GetGlobalStats(ctx)
	if err != nil {
		logrus.Errorf("Failed to get global stats: %v", err)
		return nil, fmt.Errorf("failed to get global stats: %w", err)
	}

	logrus.Debugf("Global stats: tasks=%d, executions=%d, success_rate=%.2f%%, avg_duration=%dms",
		stats.TotalTasks, stats.TotalExecutions, stats.SuccessRate, stats.AverageDurationMs)

	return stats, nil
}

// CalculateSuccessRate calculates success rate for a task
// T016: Helper method for success rate calculation
func (sc *StatsCalculator) CalculateSuccessRate(successCount, totalCount int64) float64 {
	if totalCount == 0 {
		return 0.0
	}
	return float64(successCount) / float64(totalCount) * 100.0
}

// GetRecentExecutions retrieves recent executions for a task
// T016: Helper method to get recent execution history
func (sc *StatsCalculator) GetRecentExecutions(ctx context.Context, taskUID string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 10 // Default to 10 recent executions
	}

	executions, err := sc.repo.ListByTaskUID(ctx, taskUID, limit, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent executions: %w", err)
	}

	// Convert to simplified map format for API response
	result := make([]map[string]interface{}, 0, len(executions))
	for _, exec := range executions {
		result = append(result, map[string]interface{}{
			"execution_uid": exec.ExecutionUID,
			"started_at":    exec.StartedAt,
			"finished_at":   exec.FinishedAt,
			"state":         exec.State,
			"duration_ms":   exec.DurationMs,
			"trigger_type":  exec.TriggerType,
		})
	}

	return result, nil
}

// GetTaskHealthStatus determines the health status of a task based on recent executions
// T016: Helper method for task health assessment
//
// Health Status:
// - "healthy": Last 5 executions all successful
// - "warning": 1-2 failures in last 5 executions
// - "critical": 3+ failures in last 5 executions
// - "unknown": No execution history
func (sc *StatsCalculator) GetTaskHealthStatus(ctx context.Context, taskUID string) (string, error) {
	// Get last 5 executions
	executions, err := sc.repo.ListByTaskUID(ctx, taskUID, 5, 0)
	if err != nil {
		return "unknown", fmt.Errorf("failed to get executions: %w", err)
	}

	if len(executions) == 0 {
		return "unknown", nil
	}

	// Count failures in recent executions
	failureCount := 0
	for _, exec := range executions {
		if exec.State.IsFailure() {
			failureCount++
		}
	}

	// Determine health status
	switch {
	case failureCount == 0:
		return "healthy", nil
	case failureCount <= 2:
		return "warning", nil
	default:
		return "critical", nil
	}
}
