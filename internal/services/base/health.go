package base

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// HealthChecker defines the interface for health checking
type HealthChecker interface {
	// CheckHealth performs a health check on the instance
	CheckHealth(ctx context.Context, instanceID uuid.UUID) (*HealthStatus, error)
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Enabled          bool          `json:"enabled"`
	Interval         time.Duration `json:"interval"`          // How often to check
	Timeout          time.Duration `json:"timeout"`           // Timeout for each check
	FailureThreshold int           `json:"failure_threshold"` // Consecutive failures before unhealthy
	SuccessThreshold int           `json:"success_threshold"` // Consecutive successes before healthy

	// Type-specific configuration
	HTTPEndpoint   string                 `json:"http_endpoint,omitempty"`   // For HTTP-based checks
	TCPPort        int                    `json:"tcp_port,omitempty"`        // For TCP-based checks
	Command        []string               `json:"command,omitempty"`         // For command-based checks
	ExpectedStatus int                    `json:"expected_status,omitempty"` // Expected HTTP status code
	Custom         map[string]interface{} `json:"custom,omitempty"`          // Custom check parameters
}

// DefaultHealthCheckConfig returns default health check configuration
func DefaultHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		Enabled:          true,
		Interval:         60 * time.Second,
		Timeout:          10 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 1,
		ExpectedStatus:   200,
	}
}

// HealthCheckResult represents a single health check result
type HealthCheckResult struct {
	InstanceID uuid.UUID              `json:"instance_id"`
	Timestamp  time.Time              `json:"timestamp"`
	Duration   time.Duration          `json:"duration"`
	Status     *HealthStatus          `json:"status"`
	Error      error                  `json:"error,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

// HealthCheckScheduler manages scheduled health checks
type HealthCheckScheduler struct {
	manager  ServiceManager
	interval time.Duration
	timeout  time.Duration
	stopCh   chan struct{}
	logger   Logger
}

// NewHealthCheckScheduler creates a new health check scheduler
func NewHealthCheckScheduler(manager ServiceManager, interval, timeout time.Duration) *HealthCheckScheduler {
	return &HealthCheckScheduler{
		manager:  manager,
		interval: interval,
		timeout:  timeout,
		stopCh:   make(chan struct{}),
		logger:   &defaultLogger{},
	}
}

// SetLogger sets a custom logger
func (s *HealthCheckScheduler) SetLogger(logger Logger) {
	s.logger = logger
}

// Start starts the health check scheduler
func (s *HealthCheckScheduler) Start(ctx context.Context, instanceID uuid.UUID) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.logger.Info("Starting health check scheduler for instance %s (interval: %s)", instanceID, s.interval)

	// Perform initial health check immediately
	s.performHealthCheck(ctx, instanceID)

	for {
		select {
		case <-ticker.C:
			s.performHealthCheck(ctx, instanceID)
		case <-s.stopCh:
			s.logger.Info("Stopping health check scheduler for instance %s", instanceID)
			return
		case <-ctx.Done():
			s.logger.Info("Context canceled, stopping health check scheduler for instance %s", instanceID)
			return
		}
	}
}

// Stop stops the health check scheduler
func (s *HealthCheckScheduler) Stop() {
	close(s.stopCh)
}

// performHealthCheck performs a single health check
func (s *HealthCheckScheduler) performHealthCheck(ctx context.Context, instanceID uuid.UUID) {
	checkCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	start := time.Now()
	health, err := s.manager.HealthCheck(checkCtx, instanceID)
	duration := time.Since(start)

	if err != nil {
		s.logger.Error("Health check failed for instance %s: %v (duration: %s)", instanceID, err, duration)

		// Update to unhealthy status
		health = &HealthStatus{
			Healthy:   false,
			Status:    HealthUnhealthy,
			Message:   fmt.Sprintf("Health check failed: %v", err),
			CheckedAt: time.Now(),
			Latency:   duration,
		}
	} else {
		health.Latency = duration
		s.logger.Debug("Health check succeeded for instance %s: %s (duration: %s)", instanceID, health.Status, duration)
	}

	// Note: Health status update should be handled by the concrete service manager
	// through its own mechanisms or by calling UpdateHealthStatus directly
	// The scheduler is primarily for triggering health checks
}

// PerformHealthCheck performs a one-time health check
func PerformHealthCheck(ctx context.Context, manager ServiceManager, instanceID uuid.UUID, timeout time.Duration) (*HealthCheckResult, error) {
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	health, err := manager.HealthCheck(checkCtx, instanceID)
	duration := time.Since(start)

	result := &HealthCheckResult{
		InstanceID: instanceID,
		Timestamp:  start,
		Duration:   duration,
		Status:     health,
		Error:      err,
	}

	if err != nil {
		result.Status = &HealthStatus{
			Healthy:   false,
			Status:    HealthUnhealthy,
			Message:   fmt.Sprintf("Health check error: %v", err),
			CheckedAt: time.Now(),
			Latency:   duration,
		}
	}

	return result, err
}

// ParseHealthCheckConfig parses health check configuration from a map
func ParseHealthCheckConfig(configMap map[string]interface{}) (*HealthCheckConfig, error) {
	config := DefaultHealthCheckConfig()

	if enabled, ok := configMap["enabled"].(bool); ok {
		config.Enabled = enabled
	}

	if interval, ok := configMap["interval"].(float64); ok {
		config.Interval = time.Duration(interval) * time.Second
	}

	if timeout, ok := configMap["timeout"].(float64); ok {
		config.Timeout = time.Duration(timeout) * time.Second
	}

	if failureThreshold, ok := configMap["failure_threshold"].(float64); ok {
		config.FailureThreshold = int(failureThreshold)
	}

	if successThreshold, ok := configMap["success_threshold"].(float64); ok {
		config.SuccessThreshold = int(successThreshold)
	}

	if httpEndpoint, ok := configMap["http_endpoint"].(string); ok {
		config.HTTPEndpoint = httpEndpoint
	}

	if tcpPort, ok := configMap["tcp_port"].(float64); ok {
		config.TCPPort = int(tcpPort)
	}

	if expectedStatus, ok := configMap["expected_status"].(float64); ok {
		config.ExpectedStatus = int(expectedStatus)
	}

	return config, nil
}

// ValidateHealthStatus validates a health status string
func ValidateHealthStatus(status string) error {
	validStatuses := []string{HealthHealthy, HealthUnhealthy, HealthDegraded, HealthUnknown}
	for _, valid := range validStatuses {
		if status == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid health status: %s (valid: %v)", status, validStatuses)
}
