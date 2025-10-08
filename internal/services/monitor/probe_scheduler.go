package monitor

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	probing "github.com/prometheus-community/pro-bing"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/alert"
	"github.com/ysicing/tiga/proto"
)

// AgentTaskDispatcher describes the minimal agent manager capabilities required by the scheduler.
type AgentTaskDispatcher interface {
	QueueTask(agentUUID string, task *proto.AgentTask) error
	GetAllAgentUUIDs() []string
}

// ProbeTask represents a scheduled probe task
type ProbeTask struct {
	MonitorID   uuid.UUID
	Monitor     *models.ServiceMonitor
	CronEntryID cron.EntryID
	LastRun     time.Time
	NextRun     time.Time
}

type certificateMetadata struct {
	Subject      string   `json:"subject"`
	Issuer       string   `json:"issuer"`
	NotBefore    string   `json:"not_before"`
	NotAfter     string   `json:"not_after"`
	DaysToExpiry int      `json:"days_to_expiry"`
	DNSNames     []string `json:"dns_names"`
}

// ServiceProbeScheduler schedules and manages service probe tasks
type ServiceProbeScheduler struct {
	serviceRepo  repository.ServiceRepository
	cron         *cron.Cron
	tasks        sync.Map // map[uuid.UUID]*ProbeTask
	mu           sync.RWMutex
	httpClient   *http.Client        // HTTP client for probes
	sentinel     *ServiceSentinel    // ServiceSentinel for data aggregation
	alertEngine  *alert.AlertEngine  // AlertEngine for triggering alerts
	agentManager AgentTaskDispatcher // AgentManager for task dispatch

	// Agent communication (for distributed probing)
	agentStreams sync.Map // map[string]proto.ServiceProbe_ExecuteProbeServer
}

// NewServiceProbeScheduler creates a new scheduler
func NewServiceProbeScheduler(serviceRepo repository.ServiceRepository, alertEngine *alert.AlertEngine) *ServiceProbeScheduler {
	sentinel := NewServiceSentinel(serviceRepo, alertEngine)

	return &ServiceProbeScheduler{
		serviceRepo: serviceRepo,
		cron:        cron.New(cron.WithSeconds()), // Support second-level precision
		sentinel:    sentinel,
		alertEngine: alertEngine,
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
	// Start ServiceSentinel
	if err := s.sentinel.Start(); err != nil {
		logrus.Errorf("Failed to start ServiceSentinel: %v", err)
	}
	// Load existing monitors and schedule them
	go s.loadAndScheduleMonitors()
}

// Stop stops the scheduler
func (s *ServiceProbeScheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	// Stop ServiceSentinel
	s.sentinel.Stop()
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

	// Log task information
	logrus.Debugf("[ServiceProbe] 服务端收到监控任务 - 类型: %s, 目标: %s, 监控ID: %s, 名称: %s, 策略: %s",
		monitor.Type, monitor.Target, monitor.ID, monitor.Name, monitor.ProbeStrategy)

	// Update last run time
	if task, ok := s.tasks.Load(monitor.ID); ok {
		probeTask := task.(*ProbeTask)
		probeTask.LastRun = time.Now()
		probeTask.NextRun = s.cron.Entry(probeTask.CronEntryID).Next
	}

	// Decide execution strategy
	if monitor.ProbeStrategy == models.ProbeStrategyServer || monitor.ProbeStrategy == "" {
		// Server-side execution
		s.executeServerProbe(ctx, monitor)
	} else {
		// Agent-side execution
		s.dispatchAgentProbe(ctx, monitor)
	}
}

// executeServerProbe executes probe on server side
func (s *ServiceProbeScheduler) executeServerProbe(ctx context.Context, monitor *models.ServiceMonitor) {
	// Execute probe based on type
	var (
		result   *models.ServiceProbeResult
		certInfo *certificateMetadata
	)

	switch monitor.Type {
	case models.ProbeTypeHTTP:
		logrus.Debugf("[ServiceProbe] 服务端执行HTTP探测 - 方法: %s, URL: %s", monitor.HTTPMethod, monitor.Target)
		result, certInfo = s.executeHTTPProbe(ctx, monitor)
	case models.ProbeTypeTCP:
		logrus.Debugf("[ServiceProbe] 服务端执行TCP探测 - 地址: %s", monitor.Target)
		result = s.executeTCPProbe(ctx, monitor)
	case models.ProbeTypeICMP:
		logrus.Debugf("[ServiceProbe] 服务端执行ICMP探测 - 主机: %s", monitor.Target)
		result = s.executeICMPProbe(ctx, monitor)
	default:
		logrus.Errorf("[ServiceProbe] Unknown probe type: %s", monitor.Type)
		return
	}

	// Save result
	if err := s.serviceRepo.SaveProbeResult(ctx, result); err != nil {
		logrus.Errorf("[ServiceProbe] Error saving probe result: %v", err)
	}

	// Log probe result summary
	if result != nil {
		if result.Success {
			logrus.Debugf("[ServiceProbe] 服务端探测成功 - 监控: %s, 延迟: %dms, 状态码: %d",
				monitor.Name, result.Latency, result.HTTPStatusCode)
		} else {
			logrus.Debugf("[ServiceProbe] 服务端探测失败 - 监控: %s, 错误: %s",
				monitor.Name, result.ErrorMessage)
		}
	}

	// Report result to ServiceSentinel for aggregation
	var metadataJSON string
	if certInfo != nil {
		if payload, err := json.Marshal(certInfo); err == nil {
			metadataJSON = string(payload)
		} else {
			logrus.Warnf("Failed to encode certificate metadata for monitor %s: %v", monitor.ID, err)
		}
	}

	report := &ProbeReport{
		ServiceMonitorID: result.ServiceMonitorID,
		HostNodeID:       uuid.Nil, // Server-side probe (use uuid.Nil for server probes)
		Success:          result.Success,
		Latency:          float32(result.Latency),
		Timestamp:        result.Timestamp,
		ErrorMessage:     result.ErrorMessage,
		Data:             metadataJSON, // Contains cert info for HTTPS probes
	}
	s.sentinel.ReportProbeResult(report)

	// Check for failures and trigger alerts if needed
	s.checkProbeFailures(ctx, monitor, result)

	// Check SSL certificate expiry for HTTPS monitors
	s.checkCertificateExpiry(ctx, monitor, result, certInfo)
}

// executeHTTPProbe executes an HTTP/HTTPS probe
func (s *ServiceProbeScheduler) executeHTTPProbe(ctx context.Context, monitor *models.ServiceMonitor) (*models.ServiceProbeResult, *certificateMetadata) {
	start := time.Now()
	result := &models.ServiceProbeResult{
		ServiceMonitorID: monitor.ID,
		Timestamp:        start,
		Success:          false,
	}
	var certInfo *certificateMetadata

	// Parse target URL
	targetURL := monitor.Target
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "http://" + targetURL
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Invalid URL: %v", err)
		logrus.Errorf("HTTP probe failed for %s: %v", monitor.Name, err)
		return result, nil
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
		return result, nil
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
		return result, nil
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

		// Store certificate information
		certInfo = &certificateMetadata{
			Subject:      cert.Subject.CommonName,
			Issuer:       cert.Issuer.CommonName,
			NotBefore:    cert.NotBefore.Format(time.RFC3339),
			NotAfter:     cert.NotAfter.Format(time.RFC3339),
			DaysToExpiry: daysUntilExpiry,
			DNSNames:     cert.DNSNames,
		}

		// Warn if certificate expiring soon (30 days)
		if daysUntilExpiry < 30 && result.Success {
			// Don't fail the probe, but add a warning
			if result.ErrorMessage == "" {
				result.ErrorMessage = fmt.Sprintf("Warning: SSL certificate expiring in %d days", daysUntilExpiry)
			}
		}

		// Critical warning if certificate expiring very soon (7 days)
		if daysUntilExpiry < 7 {
			if result.ErrorMessage == "" {
				result.ErrorMessage = fmt.Sprintf("Critical: SSL certificate expiring in %d days!", daysUntilExpiry)
			}
			logrus.Warnf("[SSL Alert] Certificate for %s expiring in %d days", monitor.Name, daysUntilExpiry)
		}
	}

	logrus.Debugf("HTTP probe %s completed: success=%v, status=%d, latency=%dms",
		monitor.Name, result.Success, result.HTTPStatusCode, result.Latency)

	return result, certInfo
}

// executeTCPProbe executes a TCP port probe
func (s *ServiceProbeScheduler) executeTCPProbe(ctx context.Context, monitor *models.ServiceMonitor) *models.ServiceProbeResult {
	start := time.Now()
	result := &models.ServiceProbeResult{
		ServiceMonitorID: monitor.ID,
		Timestamp:        start,
		Success:          false,
	}

	// Parse target (should be in format "host:port")
	target := monitor.Target

	// Set timeout from monitor config or default to 10 seconds
	timeout := time.Duration(monitor.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	// Attempt TCP connection
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("TCP connection failed: %v", err)
		logrus.Errorf("TCP probe failed for %s (%s): %v", monitor.Name, target, err)
		return result
	}
	defer conn.Close()

	// Calculate latency
	responseTime := time.Since(start)
	result.Latency = int(responseTime.Milliseconds())
	result.Success = true

	// Optional: Send and receive data if configured
	if monitor.TCPSend != "" {
		// Write data to connection
		_, err := conn.Write([]byte(monitor.TCPSend))
		if err != nil {
			result.Success = false
			result.ErrorMessage = fmt.Sprintf("Failed to send data: %v", err)
			logrus.Errorf("TCP send failed for %s: %v", monitor.Name, err)
			return result
		}

		// If expect response is configured, read and validate
		if monitor.TCPExpect != "" {
			// Set read deadline
			conn.SetReadDeadline(time.Now().Add(timeout))

			// Read response (up to 1KB)
			buffer := make([]byte, 1024)
			n, err := conn.Read(buffer)
			if err != nil {
				result.Success = false
				result.ErrorMessage = fmt.Sprintf("Failed to read response: %v", err)
				logrus.Errorf("TCP read failed for %s: %v", monitor.Name, err)
				return result
			}

			// Validate response
			response := string(buffer[:n])
			if !strings.Contains(response, monitor.TCPExpect) {
				result.Success = false
				result.ErrorMessage = "Response validation failed: expected content not found"
				logrus.Errorf("TCP response validation failed for %s", monitor.Name)
				return result
			}
		}
	}

	logrus.Debugf("TCP probe %s completed: success=%v, target=%s, latency=%dms",
		monitor.Name, result.Success, target, result.Latency)

	return result
}

// executeICMPProbe executes an ICMP ping probe
func (s *ServiceProbeScheduler) executeICMPProbe(ctx context.Context, monitor *models.ServiceMonitor) *models.ServiceProbeResult {
	start := time.Now()
	result := &models.ServiceProbeResult{
		ServiceMonitorID: monitor.ID,
		Timestamp:        start,
		Success:          false,
	}

	// Parse target (can be hostname or IP)
	target := monitor.Target

	// Create pinger
	pinger, err := probing.NewPinger(target)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create pinger: %v", err)
		logrus.Errorf("ICMP probe failed for %s (%s): %v", monitor.Name, target, err)
		return result
	}

	// Configure pinger
	// Set privileged mode to true to use raw sockets (requires root/capabilities on Linux)
	// If this fails, it will fall back to unprivileged UDP ping
	pinger.SetPrivileged(true)

	// Send 5 packets (same as Nezha)
	pinger.Count = 5

	// Set timeout from monitor config or default to 20 seconds
	timeout := time.Duration(monitor.Timeout) * time.Second
	if timeout == 0 {
		timeout = 20 * time.Second
	}
	pinger.Timeout = timeout

	// Run the ping
	err = pinger.Run()
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Ping execution failed: %v", err)
		logrus.Errorf("ICMP probe execution failed for %s: %v", monitor.Name, err)
		return result
	}

	// Get statistics
	stats := pinger.Statistics()

	// Check if we received any packets
	if stats.PacketsRecv > 0 {
		result.Success = true
		result.Latency = int(stats.AvgRtt.Milliseconds())
	} else {
		result.ErrorMessage = fmt.Sprintf("No packets received (sent: %d, received: %d, loss: %.1f%%)",
			stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
	}

	logrus.Debugf("ICMP probe %s completed: success=%v, target=%s, latency=%dms, loss=%.1f%%",
		monitor.Name, result.Success, target, result.Latency, stats.PacketLoss)

	return result
}

// checkProbeFailures checks for consecutive failures and triggers alerts
func (s *ServiceProbeScheduler) checkProbeFailures(ctx context.Context, monitor *models.ServiceMonitor, latestResult *models.ServiceProbeResult) {
	if !monitor.NotifyOnFailure || s.alertEngine == nil {
		return
	}

	// Get recent probe results (larger window to calculate availability)
	lookbackDuration := time.Hour // Get 1 hour of data for better availability calculation
	startTime := time.Now().Add(-lookbackDuration)
	results, _, err := s.serviceRepo.GetProbeHistory(ctx, monitor.ID, startTime, time.Time{}, 0)
	if err != nil {
		logrus.Warnf("Failed to get probe history for alert evaluation: %v", err)
		return
	}

	if len(results) == 0 {
		return
	}

	// Calculate availability metrics
	totalChecks := len(results)
	successfulChecks := 0
	failedChecks := 0
	var totalLatency int64
	minLatency := int(^uint(0) >> 1) // Max int
	maxLatency := 0

	for _, result := range results {
		if result.Success {
			successfulChecks++
			totalLatency += int64(result.Latency)
			if result.Latency < minLatency {
				minLatency = result.Latency
			}
			if result.Latency > maxLatency {
				maxLatency = result.Latency
			}
		} else {
			failedChecks++
		}
	}

	// Calculate uptime percentage
	var uptimePercentage float64
	if totalChecks > 0 {
		uptimePercentage = float64(successfulChecks) / float64(totalChecks) * 100
	}

	// Calculate average latency
	var avgLatency float64
	if successfulChecks > 0 {
		avgLatency = float64(totalLatency) / float64(successfulChecks)
	}

	// Create availability data for alert evaluation
	availability := &models.ServiceAvailability{
		ServiceMonitorID: monitor.ID,
		Period:           "1h",
		StartTime:        startTime,
		EndTime:          time.Now(),
		TotalChecks:      totalChecks,
		SuccessfulChecks: successfulChecks,
		FailedChecks:     failedChecks,
		UptimePercentage: uptimePercentage,
		AvgLatency:       avgLatency,
		MinLatency:       minLatency,
		MaxLatency:       maxLatency,
	}

	// Evaluate alert rules via AlertEngine
	if err := s.alertEngine.EvaluateServiceRules(ctx, monitor.ID, availability); err != nil {
		logrus.Warnf("Failed to evaluate alert rules for service %s: %v", monitor.Name, err)
	}
}

// checkCertificateExpiry checks SSL certificate expiry and creates alerts
func (s *ServiceProbeScheduler) checkCertificateExpiry(ctx context.Context, monitor *models.ServiceMonitor, latestResult *models.ServiceProbeResult, certInfo *certificateMetadata) {
	// Only check HTTPS monitors
	if monitor.Type != models.ProbeTypeHTTP {
		return
	}

	// Check if monitor has certificate alert enabled (via NotifyOnFailure)
	if !monitor.NotifyOnFailure || s.alertEngine == nil {
		return
	}

	if certInfo == nil {
		return
	}

	// Define severity levels
	var severity string
	var shouldAlert bool

	if certInfo.DaysToExpiry < 0 {
		// Certificate has expired
		severity = "critical"
		shouldAlert = true
	} else if certInfo.DaysToExpiry <= 7 {
		// Critical: expiring within 7 days
		severity = "critical"
		shouldAlert = true
	} else if certInfo.DaysToExpiry <= 15 {
		// Warning: expiring within 15 days
		severity = "warning"
		shouldAlert = true
	} else if certInfo.DaysToExpiry <= 30 {
		// Info: expiring within 30 days
		severity = "info"
		shouldAlert = true
	}

	if !shouldAlert {
		return
	}

	// Create certificate alert event
	message := fmt.Sprintf("SSL certificate for %s expiring in %d days", monitor.Name, certInfo.DaysToExpiry)
	if certInfo.DaysToExpiry < 0 {
		message = fmt.Sprintf("SSL certificate for %s has expired!", monitor.Name)
	}

	// Format certificate details
	subject := certInfo.Subject
	if subject == "" {
		subject = "Unknown"
	}
	issuer := certInfo.Issuer
	if issuer == "" {
		issuer = "Unknown"
	}
	notAfter := certInfo.NotAfter
	if notAfter == "" {
		notAfter = "Unknown"
	}

	details := fmt.Sprintf("Certificate Details:\n- Subject: %s\n- Issuer: %s\n- Expires: %s\n- Days Until Expiry: %d",
		subject, issuer, notAfter, certInfo.DaysToExpiry)

	// Trigger certificate expiry alert via AlertEngine
	certAlert := &models.ServiceAvailability{
		ServiceMonitorID: monitor.ID,
		Period:           "cert_check",
		StartTime:        time.Now(),
		EndTime:          time.Now(),
		TotalChecks:      1,
		SuccessfulChecks: 1,
		FailedChecks:     0,
		UptimePercentage: 100,
		AvgLatency:       float64(latestResult.Latency),
	}

	// For cert expiry alerts, we manually create alert event
	// since standard rule evaluation doesn't cover certificate expiry
	// We'll use the existing alert engine but may need to enhance it
	logrus.Infof("[SSL Alert] %s (severity: %s)", message, severity)
	logrus.Debugf("[SSL Alert] %s", details)

	// Try to evaluate with standard rules first
	// Real implementation would create a dedicated cert alert event
	if err := s.alertEngine.EvaluateServiceRules(ctx, monitor.ID, certAlert); err != nil {
		logrus.Warnf("Failed to evaluate cert alert for service %s: %v", monitor.Name, err)
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
		result, _ = s.executeHTTPProbe(ctx, monitor)
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

// ReportAgentProbeResultBatch receives batch probe results from Agents and forwards to ServiceSentinel
// This is a performance optimization to reduce gRPC overhead and database transactions
func (s *ServiceProbeScheduler) ReportAgentProbeResultBatch(reports []*ProbeReport) {
	// Forward each result to ServiceSentinel for aggregation
	for _, report := range reports {
		s.sentinel.ReportProbeResult(report)
	}
}

// GetOverviewStats returns 30-day aggregated statistics for all service monitors
// This method delegates to ServiceSentinel.CopyStats()
func (s *ServiceProbeScheduler) GetOverviewStats(ctx context.Context) (map[string]*ServiceResponseItem, error) {
	if s.sentinel == nil {
		return nil, fmt.Errorf("sentinel not initialized")
	}
	return s.sentinel.CopyStats(ctx)
}

// SetAgentManager sets the agent manager (to avoid circular dependency)
func (s *ServiceProbeScheduler) SetAgentManager(agentManager AgentTaskDispatcher) {
	s.agentManager = agentManager
}

// dispatchAgentProbe dispatches probe task to agent(s)
func (s *ServiceProbeScheduler) dispatchAgentProbe(ctx context.Context, monitor *models.ServiceMonitor) {
	if s.agentManager == nil {
		logrus.Warnf("[ServiceProbe] AgentManager not set, fallback to server probe for monitor %s", monitor.ID)
		s.executeServerProbe(ctx, monitor)
		return
	}

	// Get target agent UUIDs based on strategy
	agentUUIDs, err := s.selectAgents(ctx, monitor)
	if err != nil {
		logrus.Errorf("[ServiceProbe] Failed to select agents for monitor %s: %v", monitor.ID, err)
		return
	}

	if len(agentUUIDs) == 0 {
		logrus.Warnf("[ServiceProbe] No agents available for monitor %s, fallback to server probe", monitor.ID)
		s.executeServerProbe(ctx, monitor)
		return
	}

	// Create agent task
	task := &proto.AgentTask{
		TaskId:   fmt.Sprintf("%d", time.Now().UnixNano()), // Use timestamp as task ID
		TaskType: "probe",
		Params: map[string]string{
			"type":       string(monitor.Type),
			"target":     monitor.Target,
			"monitor_id": monitor.ID.String(),
		},
	}

	// Add type-specific parameters
	if monitor.Type == models.ProbeTypeHTTP {
		task.Params["method"] = monitor.HTTPMethod
		if monitor.HTTPHeaders != "" {
			task.Params["headers"] = monitor.HTTPHeaders
		}
		if monitor.ExpectStatus > 0 {
			task.Params["expected_status"] = fmt.Sprintf("%d", monitor.ExpectStatus)
		}
	}

	// Dispatch task to each agent
	for _, agentUUID := range agentUUIDs {
		if err := s.agentManager.QueueTask(agentUUID, task); err != nil {
			logrus.Errorf("[ServiceProbe] Failed to queue task to agent %s: %v", agentUUID, err)
		} else {
			logrus.Infof("[ServiceProbe] Dispatched probe task to agent %s for monitor %s (target: %s)",
				agentUUID, monitor.Name, monitor.Target)
		}
	}
}

// selectAgents selects agent UUIDs based on probe strategy
func (s *ServiceProbeScheduler) selectAgents(ctx context.Context, monitor *models.ServiceMonitor) ([]string, error) {
	var agentUUIDs []string

	switch monitor.ProbeStrategy {
	case models.ProbeStrategyInclude:
		// Parse included node IDs from JSON array
		if monitor.ProbeNodeIDs != "" {
			var nodeIDs []string
			if err := json.Unmarshal([]byte(monitor.ProbeNodeIDs), &nodeIDs); err != nil {
				return nil, fmt.Errorf("failed to parse probe_node_ids: %w", err)
			}
			agentUUIDs = nodeIDs
		}

	case models.ProbeStrategyExclude:
		// Get all agents and exclude specified ones
		allAgents := s.agentManager.GetAllAgentUUIDs()
		if monitor.ProbeNodeIDs != "" {
			var excludeIDs []string
			if err := json.Unmarshal([]byte(monitor.ProbeNodeIDs), &excludeIDs); err != nil {
				return nil, fmt.Errorf("failed to parse probe_node_ids: %w", err)
			}
			// Filter out excluded agents
			excludeMap := make(map[string]bool)
			for _, id := range excludeIDs {
				excludeMap[id] = true
			}
			for _, uuid := range allAgents {
				if !excludeMap[uuid] {
					agentUUIDs = append(agentUUIDs, uuid)
				}
			}
		} else {
			agentUUIDs = allAgents
		}

	case models.ProbeStrategyGroup:
		// Get agents by group name (this requires HostRepository support)
		// For now, fallback to all agents
		// TODO: Implement group-based agent selection
		logrus.Warnf("[ServiceProbe] Group strategy not fully implemented yet, using all agents")
		agentUUIDs = s.agentManager.GetAllAgentUUIDs()

	default:
		return nil, fmt.Errorf("unknown probe strategy: %s", monitor.ProbeStrategy)
	}

	return agentUUIDs, nil
}
