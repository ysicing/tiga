package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
)

// TimeoutController handles task execution timeout control
// T015: Timeout control mechanism with Context and grace period
//
// Reference: .claude/specs/006-gitness-tiga/tasks.md T015
//
//	.claude/specs/006-gitness-tiga/research.md Section 5 (Timeout Control)
type TimeoutController struct {
	defaultTimeout time.Duration // Default timeout duration
	gracePeriod    time.Duration // Grace period for cleanup (30 seconds)
}

// NewTimeoutController creates a new timeout controller
func NewTimeoutController(defaultTimeout time.Duration) *TimeoutController {
	return &TimeoutController{
		defaultTimeout: defaultTimeout,
		gracePeriod:    30 * time.Second, // Fixed 30-second grace period
	}
}

// ExecuteWithTimeout executes a task with timeout control
// T015: Implements Context timeout with 30-second grace period
//
// Flow:
//  1. Create timeout context with max_duration_seconds
//  2. Execute task in goroutine
//  3. If timeout:
//     a. Cancel context (task should stop)
//     b. Wait gracePeriod (30s) for cleanup
//     c. If still running after grace period, mark as timed out
func (tc *TimeoutController) ExecuteWithTimeout(
	ctx context.Context,
	task Task,
	timeout time.Duration,
	execution *models.TaskExecution,
	recorder *ExecutionRecorder,
) error {
	// Use schedule timeout or default
	if timeout <= 0 {
		timeout = tc.defaultTimeout
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Channel to receive task completion
	done := make(chan error, 1)
	timedOut := false

	// Execute task in goroutine
	go func() {
		defer func() {
			// Recover from panic
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic: %v", r)
			}
		}()

		// Update to running state
		_ = recorder.UpdateToRunning(ctx, execution)

		// Execute task
		err := task.Run(timeoutCtx)
		done <- err
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		// Task completed normally
		if err != nil {
			// Task failed
			return recorder.CompleteFailure(ctx, execution, err, "")
		}

		// Task succeeded - check if it provides a result
		result := ""
		if resultProvider, ok := task.(ResultProvider); ok {
			result = resultProvider.GetResult()
		}
		return recorder.CompleteSuccess(ctx, execution, result)

	case <-timeoutCtx.Done():
		// Timeout occurred
		logrus.Warnf("Task %s timed out after %v, waiting %v grace period",
			task.Name(), timeout, tc.gracePeriod)

		timedOut = true

		// Wait for grace period
		graceTimer := time.NewTimer(tc.gracePeriod)
		defer graceTimer.Stop()

		select {
		case err := <-done:
			// Task completed during grace period
			graceTimer.Stop()
			logrus.Infof("Task %s completed during grace period", task.Name())

			if err != nil {
				return recorder.CompleteFailure(ctx, execution, err, "")
			}

			// Check if task provides a result
			result := ""
			if resultProvider, ok := task.(ResultProvider); ok {
				result = resultProvider.GetResult()
			}
			return recorder.CompleteSuccess(ctx, execution, result)

		case <-graceTimer.C:
			// Grace period expired, task still running
			logrus.Errorf("Task %s did not complete after %v grace period", task.Name(), tc.gracePeriod)
			return recorder.CompleteTimeout(ctx, execution, timeout+tc.gracePeriod)
		}
	}

	if timedOut {
		// This should not be reached, but handle it just in case
		return recorder.CompleteTimeout(ctx, execution, timeout)
	}

	return nil
}

// GetGracePeriod returns the configured grace period
func (tc *TimeoutController) GetGracePeriod() time.Duration {
	return tc.gracePeriod
}

// GetDefaultTimeout returns the configured default timeout
func (tc *TimeoutController) GetDefaultTimeout() time.Duration {
	return tc.defaultTimeout
}

// CalculateActualTimeout calculates the actual timeout including grace period
func (tc *TimeoutController) CalculateActualTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		timeout = tc.defaultTimeout
	}
	return timeout + tc.gracePeriod
}
