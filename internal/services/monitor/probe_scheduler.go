package monitor

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
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
	httpClient  *http.Client // HTTP client for probes

	// Agent communication (for distributed probing)
	agentStreams sync.Map // map[string]proto.ServiceProbe_ExecuteProbeServer
}

// NewServiceProbeScheduler creates a new scheduler
func NewServiceProbeScheduler(serviceRepo repository.ServiceRepository) *ServiceProbeScheduler {
	return &ServiceProbeScheduler{
		serviceRepo: serviceRepo,
		cron:        cron.New(cron.WithSeconds()), // Support second-level precision
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // TODO: Make configurable
				},
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
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
			fmt.Printf("Error scheduling monitor %s: %v\n", monitor.ID.String(), err)
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
	start := time.Now()
	result := &models.ServiceProbeResult{
		ServiceMonitorID: monitor.ID,
		Timestamp:        start,
		Success:          false,
	}

	// Parse target URL
	targetURL := monitor.Target
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "http://" + targetURL
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Invalid URL: %v", err)
		logrus.Errorf("HTTP probe failed for %s: %v", monitor.Name, err)
		return result
	}

	// Create HTTP request
	method := monitor.HTTPMethod
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, parsedURL.String(), nil)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		logrus.Errorf("HTTP probe failed for %s: %v", monitor.Name, err)
		return result
	}

	// Parse and add custom headers if configured
	if monitor.HTTPHeaders != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(monitor.HTTPHeaders), &headers); err == nil {
			for key, value := range headers {
				req.Header.Set(key, value)
			}
		}
	}

	// Add default headers
	req.Header.Set("User-Agent", "Tiga/1.0 ServiceProbe")

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Request failed: %v", err)
		logrus.Errorf("HTTP probe failed for %s: %v", monitor.Name, err)
		return result
	}
	defer resp.Body.Close()

	// Calculate response time
	responseTime := time.Since(start)
	result.Latency = int(responseTime.Milliseconds())
	result.HTTPStatusCode = resp.StatusCode

	// Check status code
	expectedStatus := monitor.ExpectStatus
	if expectedStatus == 0 {
		expectedStatus = 200
	}

	if resp.StatusCode != expectedStatus {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Unexpected status code: %d (expected %d)", resp.StatusCode, expectedStatus)
	} else {
		result.Success = true
	}

	// Check response content if configured
	if result.Success && monitor.ExpectBody != "" {
		// Read response body (limited to 10KB)
		bodyBytes := make([]byte, 1024*10)
		n, _ := io.ReadFull(resp.Body, bodyBytes)
		body := string(bodyBytes[:n])

		// Store first 1KB in result
		if len(body) > 1024 {
			result.HTTPResponseBody = body[:1024]
		} else {
			result.HTTPResponseBody = body
		}

		// Check for expected content (substring match)
		if !strings.Contains(body, monitor.ExpectBody) {
			result.Success = false
			result.ErrorMessage = "Content validation failed: expected content not found"
		}
	}

	// SSL certificate validation for HTTPS
	if parsedURL.Scheme == "https" && resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)

		// Warn if certificate expiring soon
		if daysUntilExpiry < 30 && result.Success {
			// Don't fail the probe, but add a warning
			if result.ErrorMessage == "" {
				result.ErrorMessage = fmt.Sprintf("Warning: SSL certificate expiring in %d days", daysUntilExpiry)
			}
		}
	}

	logrus.Debugf("HTTP probe %s completed: success=%v, status=%d, latency=%dms",
		monitor.Name, result.Success, result.HTTPStatusCode, result.Latency)

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
