package performance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Monitor tracks API performance metrics
type Monitor struct {
	metrics       map[string]*EndpointMetrics
	mu            sync.RWMutex
	slowThreshold time.Duration
	slowRequests  []SlowRequest
	maxSlowLogs   int
}

// EndpointMetrics holds performance metrics for an endpoint
type EndpointMetrics struct {
	Endpoint      string        `json:"endpoint"`
	Method        string        `json:"method"`
	Count         int64         `json:"count"`
	TotalDuration time.Duration `json:"total_duration"`
	MinDuration   time.Duration `json:"min_duration"`
	MaxDuration   time.Duration `json:"max_duration"`
	AvgDuration   time.Duration `json:"avg_duration"`
	P50Duration   time.Duration `json:"p50_duration"`
	P95Duration   time.Duration `json:"p95_duration"`
	P99Duration   time.Duration `json:"p99_duration"`
	ErrorCount    int64         `json:"error_count"`
	durations     []time.Duration
}

// SlowRequest represents a slow request log
type SlowRequest struct {
	Timestamp time.Time     `json:"timestamp"`
	Method    string        `json:"method"`
	Endpoint  string        `json:"endpoint"`
	Duration  time.Duration `json:"duration"`
	Status    int           `json:"status"`
	ClientIP  string        `json:"client_ip"`
	UserAgent string        `json:"user_agent"`
}

// NewMonitor creates a new performance monitor
func NewMonitor(slowThreshold time.Duration) *Monitor {
	return &Monitor{
		metrics:       make(map[string]*EndpointMetrics),
		slowThreshold: slowThreshold,
		slowRequests:  make([]SlowRequest, 0),
		maxSlowLogs:   1000,
	}
}

// Middleware creates a Gin middleware for performance monitoring
func (m *Monitor) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}
		method := c.Request.Method
		status := c.Writer.Status()

		// Record metrics
		m.recordMetrics(method, endpoint, duration, status >= 400)

		// Log slow requests
		if duration > m.slowThreshold {
			m.logSlowRequest(SlowRequest{
				Timestamp: start,
				Method:    method,
				Endpoint:  endpoint,
				Duration:  duration,
				Status:    status,
				ClientIP:  c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
			})
		}

		// Add performance headers
		c.Header("X-Response-Time", duration.String())
	}
}

// recordMetrics records performance metrics for an endpoint
func (m *Monitor) recordMetrics(method, endpoint string, duration time.Duration, isError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := method + ":" + endpoint
	metrics, exists := m.metrics[key]

	if !exists {
		metrics = &EndpointMetrics{
			Endpoint:    endpoint,
			Method:      method,
			MinDuration: duration,
			MaxDuration: duration,
			durations:   make([]time.Duration, 0, 1000),
		}
		m.metrics[key] = metrics
	}

	// Update metrics
	metrics.Count++
	metrics.TotalDuration += duration
	metrics.AvgDuration = metrics.TotalDuration / time.Duration(metrics.Count)

	if duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}

	if isError {
		metrics.ErrorCount++
	}

	// Store duration for percentile calculation (limit to prevent memory growth)
	if len(metrics.durations) < 1000 {
		metrics.durations = append(metrics.durations, duration)
		m.calculatePercentiles(metrics)
	}
}

// calculatePercentiles calculates percentile metrics
func (m *Monitor) calculatePercentiles(metrics *EndpointMetrics) {
	if len(metrics.durations) == 0 {
		return
	}

	// Simple percentile calculation (not perfectly accurate but fast)
	sorted := make([]time.Duration, len(metrics.durations))
	copy(sorted, metrics.durations)

	// Bubble sort for small datasets
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate percentiles
	p50Index := int(float64(len(sorted)) * 0.50)
	p95Index := int(float64(len(sorted)) * 0.95)
	p99Index := int(float64(len(sorted)) * 0.99)

	if p50Index < len(sorted) {
		metrics.P50Duration = sorted[p50Index]
	}
	if p95Index < len(sorted) {
		metrics.P95Duration = sorted[p95Index]
	}
	if p99Index < len(sorted) {
		metrics.P99Duration = sorted[p99Index]
	}
}

// logSlowRequest logs a slow request
func (m *Monitor) logSlowRequest(req SlowRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add to slow requests log
	m.slowRequests = append(m.slowRequests, req)

	// Keep only the most recent slow requests
	if len(m.slowRequests) > m.maxSlowLogs {
		m.slowRequests = m.slowRequests[1:]
	}
}

// GetMetrics retrieves all performance metrics
func (m *Monitor) GetMetrics() map[string]*EndpointMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	result := make(map[string]*EndpointMetrics)
	for k, v := range m.metrics {
		metricsCopy := *v
		result[k] = &metricsCopy
	}

	return result
}

// GetEndpointMetrics retrieves metrics for a specific endpoint
func (m *Monitor) GetEndpointMetrics(method, endpoint string) *EndpointMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := method + ":" + endpoint
	if metrics, exists := m.metrics[key]; exists {
		metricsCopy := *metrics
		return &metricsCopy
	}

	return nil
}

// GetSlowRequests retrieves slow request logs
func (m *Monitor) GetSlowRequests(limit int) []SlowRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.slowRequests) {
		limit = len(m.slowRequests)
	}

	// Return most recent slow requests
	start := len(m.slowRequests) - limit
	result := make([]SlowRequest, limit)
	copy(result, m.slowRequests[start:])

	return result
}

// Reset resets all metrics
func (m *Monitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics = make(map[string]*EndpointMetrics)
	m.slowRequests = make([]SlowRequest, 0)
}

// GetSummary returns a performance summary
func (m *Monitor) GetSummary() *PerformanceSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary := &PerformanceSummary{
		TotalRequests:      0,
		TotalErrors:        0,
		AverageDuration:    0,
		SlowestEndpoint:    "",
		SlowestDuration:    0,
		FastestEndpoint:    "",
		FastestDuration:    time.Hour,
		SlowRequestCount:   int64(len(m.slowRequests)),
		EndpointsMonitored: len(m.metrics),
	}

	var totalDuration time.Duration

	for _, metrics := range m.metrics {
		summary.TotalRequests += metrics.Count
		summary.TotalErrors += metrics.ErrorCount
		totalDuration += metrics.TotalDuration

		if metrics.MaxDuration > summary.SlowestDuration {
			summary.SlowestDuration = metrics.MaxDuration
			summary.SlowestEndpoint = metrics.Method + " " + metrics.Endpoint
		}

		if metrics.MinDuration < summary.FastestDuration {
			summary.FastestDuration = metrics.MinDuration
			summary.FastestEndpoint = metrics.Method + " " + metrics.Endpoint
		}
	}

	if summary.TotalRequests > 0 {
		summary.AverageDuration = totalDuration / time.Duration(summary.TotalRequests)
		summary.ErrorRate = float64(summary.TotalErrors) / float64(summary.TotalRequests)
	}

	return summary
}

// PerformanceSummary represents overall performance summary
type PerformanceSummary struct {
	TotalRequests      int64         `json:"total_requests"`
	TotalErrors        int64         `json:"total_errors"`
	ErrorRate          float64       `json:"error_rate"`
	AverageDuration    time.Duration `json:"average_duration"`
	SlowestEndpoint    string        `json:"slowest_endpoint"`
	SlowestDuration    time.Duration `json:"slowest_duration"`
	FastestEndpoint    string        `json:"fastest_endpoint"`
	FastestDuration    time.Duration `json:"fastest_duration"`
	SlowRequestCount   int64         `json:"slow_request_count"`
	EndpointsMonitored int           `json:"endpoints_monitored"`
}

// CheckHealth checks if performance is within acceptable limits
func (m *Monitor) CheckHealth(maxP95 time.Duration) HealthCheck {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := HealthCheck{
		Healthy:   true,
		Timestamp: time.Now(),
		Issues:    make([]string, 0),
	}

	for key, metrics := range m.metrics {
		if metrics.P95Duration > maxP95 {
			health.Healthy = false
			health.Issues = append(health.Issues,
				key+" P95 duration "+metrics.P95Duration.String()+" exceeds threshold "+maxP95.String())
		}

		errorRate := float64(metrics.ErrorCount) / float64(metrics.Count)
		if errorRate > 0.05 { // 5% error rate
			health.Healthy = false
			health.Issues = append(health.Issues,
				fmt.Sprintf("%s error rate %.2f%% exceeds 5%%", key, errorRate*100))
		}
	}

	return health
}

// HealthCheck represents performance health status
type HealthCheck struct {
	Healthy   bool      `json:"healthy"`
	Timestamp time.Time `json:"timestamp"`
	Issues    []string  `json:"issues,omitempty"`
}

// PerformanceAlert represents a performance alert
type PerformanceAlert struct {
	Level     string    `json:"level"` // warning, critical
	Endpoint  string    `json:"endpoint"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
}

// CheckAlerts checks for performance alerts
func (m *Monitor) CheckAlerts(p95Threshold, errorRateThreshold float64) []PerformanceAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	alerts := make([]PerformanceAlert, 0)
	now := time.Now()

	for key, metrics := range m.metrics {
		// Check P95 latency
		p95Ms := float64(metrics.P95Duration.Milliseconds())
		if p95Ms > p95Threshold {
			level := "warning"
			if p95Ms > p95Threshold*2 {
				level = "critical"
			}

			alerts = append(alerts, PerformanceAlert{
				Level:     level,
				Endpoint:  key,
				Message:   "High P95 latency",
				Timestamp: now,
				Value:     p95Ms,
				Threshold: p95Threshold,
			})
		}

		// Check error rate
		if metrics.Count > 0 {
			errorRate := float64(metrics.ErrorCount) / float64(metrics.Count)
			if errorRate > errorRateThreshold {
				level := "warning"
				if errorRate > errorRateThreshold*2 {
					level = "critical"
				}

				alerts = append(alerts, PerformanceAlert{
					Level:     level,
					Endpoint:  key,
					Message:   "High error rate",
					Timestamp: now,
					Value:     errorRate * 100,
					Threshold: errorRateThreshold * 100,
				})
			}
		}
	}

	return alerts
}

// StartAutoReset starts automatic metric reset
func (m *Monitor) StartAutoReset(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.Reset()
		}
	}
}
