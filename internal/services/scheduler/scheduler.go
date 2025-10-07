package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
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
	Task     Task
	Interval time.Duration
	Enabled  bool
}

// Scheduler manages scheduled tasks
type Scheduler struct {
	schedules map[string]*Schedule
	mu        sync.RWMutex
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// NewScheduler creates a new scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		schedules: make(map[string]*Schedule),
		stopCh:    make(chan struct{}),
	}
}

// AddTask adds a task to the scheduler
func (s *Scheduler) AddTask(name string, task Task, interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.schedules[name] = &Schedule{
		Task:     task,
		Interval: interval,
		Enabled:  true,
	}

	logrus.Infof("Added task %s with interval %v", name, interval)
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
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for name, schedule := range s.schedules {
		s.wg.Add(1)
		go s.runTask(ctx, name, schedule)
	}

	logrus.Info("Scheduler started")
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	close(s.stopCh)
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

// executeTask executes a single task
func (s *Scheduler) executeTask(ctx context.Context, name string, schedule *Schedule) {
	if !schedule.Enabled {
		logrus.Debugf("Task %s is disabled, skipping", name)
		return
	}

	logrus.Debugf("Executing task %s", name)
	start := time.Now()

	if err := schedule.Task.Run(ctx); err != nil {
		logrus.Errorf("Task %s failed: %v", name, err)
	} else {
		logrus.Debugf("Task %s completed in %v", name, time.Since(start))
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
