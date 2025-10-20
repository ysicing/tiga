package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository/scheduler"
)

// ExecutionRecorder handles task execution history recording
// T014: Task execution history recording service
//
// Reference: .claude/specs/006-gitness-tiga/tasks.md T014
type ExecutionRecorder struct {
	repo       scheduler.ExecutionRepository
	instanceID string
}

// NewExecutionRecorder creates a new execution recorder
func NewExecutionRecorder(repo scheduler.ExecutionRepository, instanceID string) *ExecutionRecorder {
	return &ExecutionRecorder{
		repo:       repo,
		instanceID: instanceID,
	}
}

// RecordExecution records a complete task execution with history tracking
// T014: Implements state transition tracking (pending → running → success/failure/timeout)
func (er *ExecutionRecorder) RecordExecution(
	ctx context.Context,
	schedule *Schedule,
	task Task,
	triggerType string, // "scheduled" or "manual"
	triggerBy string, // User ID for manual triggers
) *models.TaskExecution {
	executionUID := uuid.New().String()
	now := time.Now()

	// Create initial execution record (pending state)
	execution := &models.TaskExecution{
		TaskUID:      schedule.TaskUID,
		TaskName:     task.Name(),
		TaskType:     "scheduled", // TODO: Get from task metadata
		ExecutionUID: executionUID,
		RunBy:        er.instanceID,
		ScheduledAt:  now,
		StartedAt:    now,
		State:        models.ExecutionStatePending,
		Progress:     0,
		TriggerType:  triggerType,
		TriggerBy:    triggerBy,
	}

	// Create pending record
	if err := er.repo.Create(ctx, execution); err != nil {
		logrus.Errorf("Failed to create pending execution record: %v", err)
	}

	return execution
}

// UpdateToRunning transitions execution to running state
// T014: State transition: pending → running
func (er *ExecutionRecorder) UpdateToRunning(ctx context.Context, execution *models.TaskExecution) error {
	execution.State = models.ExecutionStateRunning
	execution.StartedAt = time.Now()
	execution.Progress = 10 // Initial progress

	if err := er.repo.Update(ctx, execution); err != nil {
		logrus.Errorf("Failed to update execution to running: %v", err)
		return err
	}

	logrus.Debugf("Execution %s transitioned to running", execution.ExecutionUID)
	return nil
}

// CompleteSuccess marks execution as successful
// T014: State transition: running → success
func (er *ExecutionRecorder) CompleteSuccess(
	ctx context.Context,
	execution *models.TaskExecution,
	result string,
) error {
	execution.State = models.ExecutionStateSuccess
	execution.FinishedAt = time.Now()
	execution.Result = result
	execution.Progress = 100
	execution.UpdateDuration()

	if err := er.repo.Update(ctx, execution); err != nil {
		logrus.Errorf("Failed to complete execution as success: %v", err)
		return err
	}

	logrus.Infof("Execution %s completed successfully in %dms",
		execution.ExecutionUID, execution.DurationMs)
	return nil
}

// CompleteFailure marks execution as failed
// T014: State transition: running → failure
// T014: Implements error stack recording
func (er *ExecutionRecorder) CompleteFailure(
	ctx context.Context,
	execution *models.TaskExecution,
	err error,
	errorStack string,
) error {
	execution.State = models.ExecutionStateFailure
	execution.FinishedAt = time.Now()
	execution.ErrorMessage = err.Error()
	execution.ErrorStack = errorStack
	execution.Progress = execution.Progress // Keep last progress
	execution.UpdateDuration()

	if updateErr := er.repo.Update(ctx, execution); updateErr != nil {
		logrus.Errorf("Failed to complete execution as failure: %v", updateErr)
		return updateErr
	}

	logrus.Errorf("Execution %s failed after %dms: %v",
		execution.ExecutionUID, execution.DurationMs, err)
	return nil
}

// CompleteTimeout marks execution as timed out
// T014: State transition: running → timeout
func (er *ExecutionRecorder) CompleteTimeout(
	ctx context.Context,
	execution *models.TaskExecution,
	timeoutDuration time.Duration,
) error {
	execution.State = models.ExecutionStateTimeout
	execution.FinishedAt = time.Now()
	execution.ErrorMessage = fmt.Sprintf("Execution timed out after %v", timeoutDuration)
	execution.Progress = execution.Progress // Keep last progress
	execution.UpdateDuration()

	if err := er.repo.Update(ctx, execution); err != nil {
		logrus.Errorf("Failed to complete execution as timeout: %v", err)
		return err
	}

	logrus.Warnf("Execution %s timed out after %v",
		execution.ExecutionUID, timeoutDuration)
	return nil
}

// CompleteCancelled marks execution as cancelled
// T014: State transition: running → cancelled
func (er *ExecutionRecorder) CompleteCancelled(
	ctx context.Context,
	execution *models.TaskExecution,
	reason string,
) error {
	execution.State = models.ExecutionStateCancelled
	execution.FinishedAt = time.Now()
	execution.ErrorMessage = fmt.Sprintf("Execution cancelled: %s", reason)
	execution.Progress = execution.Progress // Keep last progress
	execution.UpdateDuration()

	if err := er.repo.Update(ctx, execution); err != nil {
		logrus.Errorf("Failed to complete execution as cancelled: %v", err)
		return err
	}

	logrus.Infof("Execution %s cancelled: %s", execution.ExecutionUID, reason)
	return nil
}

// CompleteInterrupted marks execution as interrupted (system restart)
// T014: State transition: running → interrupted
func (er *ExecutionRecorder) CompleteInterrupted(
	ctx context.Context,
	execution *models.TaskExecution,
) error {
	execution.State = models.ExecutionStateInterrupted
	execution.FinishedAt = time.Now()
	execution.ErrorMessage = "Execution interrupted by system restart"
	execution.Progress = execution.Progress // Keep last progress
	execution.UpdateDuration()

	if err := er.repo.Update(ctx, execution); err != nil {
		logrus.Errorf("Failed to complete execution as interrupted: %v", err)
		return err
	}

	logrus.Warnf("Execution %s interrupted by system restart", execution.ExecutionUID)
	return nil
}

// UpdateProgress updates execution progress
func (er *ExecutionRecorder) UpdateProgress(
	ctx context.Context,
	execution *models.TaskExecution,
	progress int,
) error {
	if progress < 0 || progress > 100 {
		return fmt.Errorf("invalid progress value: %d (must be 0-100)", progress)
	}

	execution.Progress = progress

	if err := er.repo.Update(ctx, execution); err != nil {
		logrus.Errorf("Failed to update execution progress: %v", err)
		return err
	}

	return nil
}

// RecoverInterruptedExecutions marks all running executions as interrupted
// T014: Called on system startup to handle unclean shutdown
func (er *ExecutionRecorder) RecoverInterruptedExecutions(ctx context.Context) (int, error) {
	// Query all running executions
	runningExecutions, err := er.repo.ListByState(ctx, models.ExecutionStateRunning, 1000, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to query running executions: %w", err)
	}

	if len(runningExecutions) == 0 {
		return 0, nil
	}

	logrus.Infof("Found %d interrupted executions, marking as interrupted", len(runningExecutions))

	count := 0
	for _, execution := range runningExecutions {
		if err := er.CompleteInterrupted(ctx, execution); err != nil {
			logrus.Errorf("Failed to mark execution %s as interrupted: %v", execution.ExecutionUID, err)
			continue
		}
		count++
	}

	logrus.Infof("Successfully marked %d executions as interrupted", count)
	return count, nil
}
