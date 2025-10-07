package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// ProbeTask represents a scheduled probe task
type ProbeTask struct {
	MonitorID   uuid.UUID
	Monitor     *models.ServiceMonitor
	CronEntryID cron.EntryID
	LastRun     time.Time
	NextRun     time.Time
}

// ServiceProbeScheduler schedules and manages service probe tasks
type ServiceProbeScheduler struct {
	serviceRepo repository.ServiceRepository
	cron        *cron.Cron
	tasks       sync.Map // map[uuid.UUID]*ProbeTask
	mu          sync.RWMutex

	// Agent communication (for distributed probing)
	agentStreams sync.Map // map[string]proto.ServiceProbe_ExecuteProbeServer
}

// NewServiceProbeScheduler creates a new scheduler
func NewServiceProbeScheduler(serviceRepo repository.ServiceRepository) *ServiceProbeScheduler {
	return &ServiceProbeScheduler{
		serviceRepo: serviceRepo,
		cron:        cron.New(cron.WithSeconds()), // Support second-level precision
	}
}

// Start starts the scheduler
func (s *ServiceProbeScheduler) Start() {
	s.cron.Start()
	// Load existing monitors and schedule them
	go s.loadAndScheduleMonitors()
}

// Stop stops the scheduler
func (s *ServiceProbeScheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
}

// loadAndScheduleMonitors loads all enabled monitors and schedules them
func (s *ServiceProbeScheduler) loadAndScheduleMonitors() {
	ctx := context.Background()

	// Get all enabled monitors
	enabled := true
	monitors, _, err := s.serviceRepo.List(ctx, repository.ServiceFilter{
		Enabled:  &enabled,
		PageSize: 1000, // Load all
	})
	if err != nil {
		fmt.Printf("Error loading monitors: %v\n", err)
		return
	}

	// Schedule each monitor
	for _, monitor := range monitors {
		if err := s.ScheduleMonitor(monitor); err != nil {
			fmt.Printf("Error scheduling monitor %d: %v\n", monitor.ID, err)
		}
	}
}

// ScheduleMonitor schedules a service monitor for periodic probing
func (s *ServiceProbeScheduler) ScheduleMonitor(monitor *models.ServiceMonitor) error {
	// Remove existing schedule if any
	s.UnscheduleMonitor(monitor.ID)

	// Build cron expression from interval
	cronExpr := s.buildCronExpression(monitor.Interval)

	// Create the probe function
	probeFunc := func() {
		s.executeProbe(monitor)
	}

	// Schedule the task
	entryID, err := s.cron.AddFunc(cronExpr, probeFunc)
	if err != nil {
		return fmt.Errorf("failed to schedule: %w", err)
	}

	// Store task info
	task := &ProbeTask{
		MonitorID:   monitor.ID,
		Monitor:     monitor,
		CronEntryID: entryID,
		NextRun:     s.cron.Entry(entryID).Next,
	}
	s.tasks.Store(monitor.ID, task)

	return nil
}

// UnscheduleMonitor removes a monitor from the schedule
func (s *ServiceProbeScheduler) UnscheduleMonitor(monitorID uuid.UUID) {
	if task, ok := s.tasks.LoadAndDelete(monitorID); ok {
		probeTask := task.(*ProbeTask)
		s.cron.Remove(probeTask.CronEntryID)
	}
}

// buildCronExpression builds a cron expression from interval in seconds
func (s *ServiceProbeScheduler) buildCronExpression(intervalSeconds int) string {
	if intervalSeconds < 60 {
		// Every N seconds
		return fmt.Sprintf("*/%d * * * * *", intervalSeconds)
	} else if intervalSeconds < 3600 {
		// Every N minutes
		minutes := intervalSeconds / 60
		return fmt.Sprintf("0 */%d * * * *", minutes)
	} else {
		// Every N hours
		hours := intervalSeconds / 3600
		return fmt.Sprintf("0 0 */%d * * *", hours)
	}
}

// executeProbe executes a single probe task
func (s *ServiceProbeScheduler) executeProbe(monitor *models.ServiceMonitor) {
	ctx := context.Background()

	// Update last run time
	if task, ok := s.tasks.Load(monitor.ID); ok {
		probeTask := task.(*ProbeTask)
		probeTask.LastRun = time.Now()
		probeTask.NextRun = s.cron.Entry(probeTask.CronEntryID).Next
	}

	// Execute probe based on type
	var result *models.ServiceProbeResult

	switch monitor.Type {
	case models.ProbeTypeHTTP:
		result = s.executeHTTPProbe(ctx, monitor)
	case models.ProbeTypeTCP:
		result = s.executeTCPProbe(ctx, monitor)
	case models.ProbeTypeICMP:
		result = s.executeICMPProbe(ctx, monitor)
	default:
		fmt.Printf("Unknown probe type: %s\n", monitor.Type)
		return
	}

	// Save result
	if err := s.serviceRepo.SaveProbeResult(ctx, result); err != nil {
		fmt.Printf("Error saving probe result: %v\n", err)
	}

	// Check for failures and trigger alerts if needed
	s.checkProbeFailures(ctx, monitor, result)
}

// executeHTTPProbe executes an HTTP/HTTPS probe
func (s *ServiceProbeScheduler) executeHTTPProbe(ctx context.Context, monitor *models.ServiceMonitor) *models.ServiceProbeResult {
	// TODO: Implement HTTP probe using net/http
	// For now, return a mock result
	result := &models.ServiceProbeResult{
		ServiceMonitorID: monitor.ID,
		Timestamp:        time.Now(),
		Success:          true,
		Latency:          50, // milliseconds
		HTTPStatusCode:   200,
	}
	return result
}

// executeTCPProbe executes a TCP port probe
func (s *ServiceProbeScheduler) executeTCPProbe(ctx context.Context, monitor *models.ServiceMonitor) *models.ServiceProbeResult {
	// TODO: Implement TCP probe using net.Dial
	result := &models.ServiceProbeResult{
		ServiceMonitorID: monitor.ID,
		Timestamp:        time.Now(),
		Success:          true,
		Latency:          10,
	}
	return result
}

// executeICMPProbe executes an ICMP ping probe
func (s *ServiceProbeScheduler) executeICMPProbe(ctx context.Context, monitor *models.ServiceMonitor) *models.ServiceProbeResult {
	// TODO: Implement ICMP probe using golang.org/x/net/icmp
	result := &models.ServiceProbeResult{
		ServiceMonitorID: monitor.ID,
		Timestamp:        time.Now(),
		Success:          true,
		Latency:          5,
	}
	return result
}

// checkProbeFailures checks for consecutive failures and triggers alerts
func (s *ServiceProbeScheduler) checkProbeFailures(ctx context.Context, monitor *models.ServiceMonitor, latestResult *models.ServiceProbeResult) {
	if !monitor.NotifyOnFailure {
		return
	}

	// Get recent probe results
	results, _, err := s.serviceRepo.GetProbeHistory(ctx, monitor.ID, time.Time{}, time.Time{}, monitor.FailureThreshold)
	if err != nil {
		return
	}

	// Count consecutive failures
	consecutiveFailures := 0
	for i := len(results) - 1; i >= 0 && i >= len(results)-monitor.FailureThreshold; i-- {
		if !results[i].Success {
			consecutiveFailures++
		} else {
			break
		}
	}

	// Trigger alert if threshold exceeded
	if consecutiveFailures >= monitor.FailureThreshold {
		// TODO: Integrate with AlertEngine to create an alert event
		fmt.Printf("Alert: Service %s has failed %d consecutive times\n", monitor.Name, consecutiveFailures)
	}
}

// TriggerManualProbe triggers an immediate probe for a monitor
func (s *ServiceProbeScheduler) TriggerManualProbe(ctx context.Context, monitorID uuid.UUID) (*models.ServiceProbeResult, error) {
	monitor, err := s.serviceRepo.GetByID(ctx, monitorID)
	if err != nil {
		return nil, err
	}

	// Execute the probe
	var result *models.ServiceProbeResult
	switch monitor.Type {
	case models.ProbeTypeHTTP:
		result = s.executeHTTPProbe(ctx, monitor)
	case models.ProbeTypeTCP:
		result = s.executeTCPProbe(ctx, monitor)
	case models.ProbeTypeICMP:
		result = s.executeICMPProbe(ctx, monitor)
	default:
		return nil, fmt.Errorf("unknown probe type: %s", monitor.Type)
	}

	// Save result
	if err := s.serviceRepo.SaveProbeResult(ctx, result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetScheduledTasks returns all currently scheduled tasks
func (s *ServiceProbeScheduler) GetScheduledTasks() []*ProbeTask {
	var tasks []*ProbeTask
	s.tasks.Range(func(key, value interface{}) bool {
		tasks = append(tasks, value.(*ProbeTask))
		return true
	})
	return tasks
}

// UpdateMonitorSchedule updates the schedule for a monitor
func (s *ServiceProbeScheduler) UpdateMonitorSchedule(monitor *models.ServiceMonitor) error {
	return s.ScheduleMonitor(monitor)
}

// GetTaskStatus returns the status of a specific probe task
func (s *ServiceProbeScheduler) GetTaskStatus(monitorID uuid.UUID) (*ProbeTask, bool) {
	if task, ok := s.tasks.Load(monitorID); ok {
		return task.(*ProbeTask), true
	}
	return nil, false
}
