package scheduler

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository/scheduler"
)

// Task represents a scheduled task
type Task interface {
	// Run executes the task
	Run(ctx context.Context) error

	// Name returns the task name
	Name() string
}

// Schedule represents a task schedule
type Schedule struct {
	Task       Task
	Interval   time.Duration // For interval-based tasks
	CronExpr   string        // For cron-based tasks
	CronID     cron.EntryID  // Cron entry ID
	Enabled    bool
	TaskUID    string        // Unique task identifier
	MaxTimeout time.Duration // Maximum execution timeout
}

// Scheduler manages scheduled tasks
// Enhanced in T013 to support execution history and cron expressions
type Scheduler struct {
	schedules   map[string]*Schedule
	mu          sync.RWMutex
	stopCh      chan struct{}
	wg          sync.WaitGroup
	execRepo    scheduler.ExecutionRepository // T013: Execution history repository
	cron        *cron.Cron                     // T013: Cron scheduler
	instanceID  string                         // T013: Scheduler instance ID
	manualTasks chan string                    // T013: Channel for manual task triggers
}

// NewScheduler creates a new scheduler
// Enhanced in T013 to accept ExecutionRepository
func NewScheduler(execRepo scheduler.ExecutionRepository) *Scheduler {
	instanceID := fmt.Sprintf("scheduler-%s", uuid.New().String()[:8])

	return &Scheduler{
		schedules:   make(map[string]*Schedule),
		stopCh:      make(chan struct{}),
		execRepo:    execRepo,
		cron:        cron.New(),
		instanceID:  instanceID,
		manualTasks: make(chan string, 100), // Buffer for manual triggers
	}
}

// AddTask adds a task to the scheduler
func (s *Scheduler) AddTask(name string, task Task, interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	taskUID := uuid.New().String()

	s.schedules[name] = &Schedule{
		Task:       task,
		Interval:   interval,
		Enabled:    true,
		TaskUID:    taskUID,
		MaxTimeout: 30 * time.Minute, // Default timeout
	}

	logrus.Infof("Added task %s (UID=%s) with interval %v", name, taskUID, interval)
}

// AddCron adds a cron-based task to the scheduler
// T013: New method for cron expression support
func (s *Scheduler) AddCron(name string, cronExpr string, task Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	taskUID := uuid.New().String()

	// Validate cron expression by adding to cron scheduler
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		// Execute with panic recovery
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("Panic in cron task %s: %v\nStack: %s", name, r, debug.Stack())
			}
		}()

		s.triggerTask(context.Background(), name)
	})
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	s.schedules[name] = &Schedule{
		Task:       task,
		CronExpr:   cronExpr,
		CronID:     entryID,
		Enabled:    true,
		TaskUID:    taskUID,
		MaxTimeout: 30 * time.Minute, // Default timeout
	}

	logrus.Infof("Added cron task %s (UID=%s) with expression: %s", name, taskUID, cronExpr)
	return nil
}

// Trigger manually triggers a task execution
// T013: New method for manual task triggering
func (s *Scheduler) Trigger(name string) error {
	s.mu.RLock()
	schedule, ok := s.schedules[name]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("task %s not found", name)
	}

	if !schedule.Enabled {
		return fmt.Errorf("task %s is disabled", name)
	}

	// Send to manual trigger channel (non-blocking)
	select {
	case s.manualTasks <- name:
		logrus.Infof("Manually triggered task %s", name)
		return nil
	default:
		return fmt.Errorf("manual trigger queue full, try again later")
	}
}

// RemoveTask removes a task from the scheduler
func (s *Scheduler) RemoveTask(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.schedules, name)
	logrus.Infof("Removed task %s", name)
}

// EnableTask enables a task
func (s *Scheduler) EnableTask(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	schedule, ok := s.schedules[name]
	if !ok {
		return fmt.Errorf("task %s not found", name)
	}

	schedule.Enabled = true
	logrus.Infof("Enabled task %s", name)
	return nil
}

// DisableTask disables a task
func (s *Scheduler) DisableTask(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	schedule, ok := s.schedules[name]
	if !ok {
		return fmt.Errorf("task %s not found", name)
	}

	schedule.Enabled = false
	logrus.Infof("Disabled task %s", name)
	return nil
}

// Start starts the scheduler
// Enhanced in T013 to start cron scheduler and manual trigger handler
func (s *Scheduler) Start(ctx context.Context) {
	// Start cron scheduler
	s.cron.Start()
	logrus.Info("Cron scheduler started")

	// Start manual trigger handler
	s.wg.Add(1)
	go s.handleManualTriggers(ctx)

	// Start interval-based tasks
	s.mu.RLock()
	defer s.mu.RUnlock()

	for name, schedule := range s.schedules {
		// Only start interval-based tasks (not cron tasks)
		if schedule.Interval > 0 {
			s.wg.Add(1)
			go s.runTask(ctx, name, schedule)
		}
	}

	logrus.Info("Scheduler started")
}

// Stop stops the scheduler
// Enhanced in T013 to stop cron scheduler
func (s *Scheduler) Stop() {
	// Stop cron scheduler
	ctx := s.cron.Stop()
	<-ctx.Done() // Wait for cron to finish

	// Stop other goroutines
	close(s.stopCh)
	close(s.manualTasks)
	s.wg.Wait()
	logrus.Info("Scheduler stopped")
}

// runTask runs a task on a schedule
func (s *Scheduler) runTask(ctx context.Context, name string, schedule *Schedule) {
	defer s.wg.Done()

	ticker := time.NewTicker(schedule.Interval)
	defer ticker.Stop()

	// Run immediately on start
	s.executeTask(ctx, name, schedule)

	for {
		select {
		case <-ticker.C:
			s.executeTask(ctx, name, schedule)
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// executeTask executes a single task with execution history recording
// T027: Updated to use ExecutionRecorder and TimeoutController
func (s *Scheduler) executeTask(ctx context.Context, name string, schedule *Schedule) {
	if !schedule.Enabled {
		logrus.Debugf("Task %s is disabled, skipping", name)
		return
	}

	// Skip if execRepo is not available
	if s.execRepo == nil {
		logrus.Warnf("ExecutionRepository not available, running task without history")
		// Fallback to simple execution
		logrus.Debugf("Executing task %s (no history)", name)
		start := time.Now()
		if err := schedule.Task.Run(ctx); err != nil {
			logrus.Errorf("Task %s failed: %v", name, err)
		} else {
			logrus.Debugf("Task %s completed in %v", name, time.Since(start))
		}
		return
	}

	// Create execution record
	execution := &models.TaskExecution{
		TaskUID:      schedule.TaskUID,
		TaskName:     name,
		TaskType:     name, // Use task name as type
		ExecutionUID: uuid.New().String(),
		RunBy:        s.instanceID,
		ScheduledAt:  time.Now(),
		StartedAt:    time.Now(),
		State:        models.ExecutionStatePending,
		Progress:     0,
		TriggerType:  "scheduled", // Default to scheduled
		RetryCount:   0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Create execution record
	if err := s.execRepo.Create(ctx, execution); err != nil {
		logrus.Errorf("Failed to create execution record for task %s: %v", name, err)
		// Continue with execution anyway
	}

	// Create execution recorder
	recorder := NewExecutionRecorder(s.execRepo, s.instanceID)

	// Create timeout controller
	timeoutController := NewTimeoutController(schedule.MaxTimeout)

	// Execute with timeout control
	err := timeoutController.ExecuteWithTimeout(
		ctx,
		schedule.Task,
		schedule.MaxTimeout,
		execution,
		recorder,
	)

	// Log result
	if err != nil {
		logrus.Errorf("Task %s failed: %v", name, err)
	} else {
		logrus.Infof("Task %s completed successfully in %dms", name, execution.DurationMs)
	}
}

// ListTasks returns all scheduled tasks
func (s *Scheduler) ListTasks() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]string, 0, len(s.schedules))
	for name := range s.schedules {
		tasks = append(tasks, name)
	}
	return tasks
}

// GetTaskStatus returns the status of a task
func (s *Scheduler) GetTaskStatus(name string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedule, ok := s.schedules[name]
	if !ok {
		return false, fmt.Errorf("task %s not found", name)
	}

	return schedule.Enabled, nil
}

// handleManualTriggers handles manual task triggers
// T013: New method for handling manual triggers from channel
func (s *Scheduler) handleManualTriggers(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case name, ok := <-s.manualTasks:
			if !ok {
				// Channel closed
				return
			}
			s.triggerTask(ctx, name)
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// triggerTask triggers a task execution (for cron and manual triggers)
// T013: New helper method with panic recovery
func (s *Scheduler) triggerTask(ctx context.Context, name string) {
	s.mu.RLock()
	schedule, ok := s.schedules[name]
	s.mu.RUnlock()

	if !ok {
		logrus.Warnf("Task %s not found in schedules", name)
		return
	}

	// Execute with panic recovery
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Panic in task %s: %v\nStack: %s", name, r, debug.Stack())

			// Record panic in execution history
			if s.execRepo != nil {
				execution := &models.TaskExecution{
					TaskUID:      schedule.TaskUID,
					TaskName:     name,
					TaskType:     "scheduled", // TODO: Get from task metadata
					ExecutionUID: uuid.New().String(),
					RunBy:        s.instanceID,
					ScheduledAt:  time.Now(),
					StartedAt:    time.Now(),
					FinishedAt:   time.Now(),
					State:        models.ExecutionStateFailure,
					ErrorMessage: fmt.Sprintf("Panic: %v", r),
					ErrorStack:   string(debug.Stack()),
					DurationMs:   0,
					Progress:     0,
					TriggerType:  "scheduled",
				}
				_ = s.execRepo.Create(context.Background(), execution)
			}
		}
	}()

	s.executeTask(ctx, name, schedule)
}
