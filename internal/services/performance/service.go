package performance

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Service integrates all performance optimization components
type Service struct {
	monitor        *Monitor
	poolOptimizer  *PoolOptimizer
	queryOptimizer *QueryOptimizer
	config         *Config
	mu             sync.RWMutex
}

// Config holds performance service configuration
type Config struct {
	// Monitoring
	SlowRequestThreshold time.Duration

	// Timeouts
	DefaultTimeout   time.Duration
	EndpointTimeouts map[string]time.Duration

	// Compression
	CompressionConfig *CompressionConfig

	// Pool
	PoolConfig *PoolConfig

	// Query
	SlowQueryThreshold time.Duration

	// Auto-tuning
	EnableAutoTune   bool
	AutoTuneInterval time.Duration
}

// DefaultConfig returns default performance configuration
func DefaultConfig() *Config {
	return &Config{
		SlowRequestThreshold: 1 * time.Second,
		DefaultTimeout:       30 * time.Second,
		EndpointTimeouts:     make(map[string]time.Duration),
		CompressionConfig:    DefaultCompressionConfig(),
		PoolConfig:           DefaultPoolConfig(),
		SlowQueryThreshold:   500 * time.Millisecond,
		EnableAutoTune:       true,
		AutoTuneInterval:     5 * time.Minute,
	}
}

// NewService creates a new performance service
func NewService(db *sql.DB, config *Config) *Service {
	if config == nil {
		config = DefaultConfig()
	}

	service := &Service{
		monitor:        NewMonitor(config.SlowRequestThreshold),
		poolOptimizer:  NewPoolOptimizer(db, config.PoolConfig),
		queryOptimizer: NewQueryOptimizer(db, config.SlowQueryThreshold),
		config:         config,
	}

	return service
}

// Middleware returns a combined middleware for all performance optimizations
func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Apply performance monitoring
		s.monitor.Middleware()(c)
	}
}

// FullMiddleware returns all performance middlewares chained
func (s *Service) FullMiddleware() []gin.HandlerFunc {
	middlewares := []gin.HandlerFunc{
		// Performance monitoring (must be first to capture full duration)
		s.monitor.Middleware(),

		// Timeout protection
		TimeoutMiddleware(s.config.DefaultTimeout),

		// Response compression
		CompressionMiddleware(s.config.CompressionConfig),

		// Pool monitoring
		PoolMonitorMiddleware(s.poolOptimizer),
	}

	return middlewares
}

// StartAutoTune starts automatic performance tuning
func (s *Service) StartAutoTune(ctx context.Context) {
	if !s.config.EnableAutoTune {
		return
	}

	go s.poolOptimizer.StartAutoTune(ctx, s.config.AutoTuneInterval)
}

// GetPerformanceReport generates a comprehensive performance report
func (s *Service) GetPerformanceReport() *PerformanceReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &PerformanceReport{
		Timestamp:       time.Now(),
		Summary:         s.monitor.GetSummary(),
		PoolStats:       s.poolOptimizer.GetStats(),
		SlowRequests:    s.monitor.GetSlowRequests(10),
		SlowQueries:     s.queryOptimizer.GetSlowQueries(10),
		Recommendations: s.getRecommendations(),
	}
}

// PerformanceReport represents a comprehensive performance report
type PerformanceReport struct {
	Timestamp       time.Time           `json:"timestamp"`
	Summary         *PerformanceSummary `json:"summary"`
	PoolStats       *PoolStats          `json:"pool_stats"`
	SlowRequests    []SlowRequest       `json:"slow_requests"`
	SlowQueries     []SlowQuery         `json:"slow_queries"`
	Recommendations []string            `json:"recommendations"`
}

// getRecommendations generates performance recommendations
func (s *Service) getRecommendations() []string {
	recommendations := make([]string, 0)

	// Pool recommendations
	poolRecs := s.poolOptimizer.GetRecommendations()
	recommendations = append(recommendations, poolRecs...)

	// Request performance recommendations
	summary := s.monitor.GetSummary()
	if summary.AverageDuration > 500*time.Millisecond {
		recommendations = append(recommendations,
			fmt.Sprintf("Average response time (%.2fs) exceeds 500ms - consider optimizing slow endpoints",
				summary.AverageDuration.Seconds()))
	}

	if summary.ErrorRate > 0.05 {
		recommendations = append(recommendations,
			fmt.Sprintf("Error rate (%.2f%%) exceeds 5%% - investigate failing endpoints",
				summary.ErrorRate*100))
	}

	// Slow query recommendations
	slowQueries := s.queryOptimizer.GetSlowQueries(0)
	if len(slowQueries) > 10 {
		recommendations = append(recommendations,
			fmt.Sprintf("Detected %d slow queries - review query optimization", len(slowQueries)))
	}

	return recommendations
}

// CheckHealth performs comprehensive health check
func (s *Service) CheckHealth(ctx context.Context) *HealthReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	report := &HealthReport{
		Timestamp:  time.Now(),
		Healthy:    true,
		Components: make(map[string]ComponentHealth),
	}

	// Check pool health
	poolHealth := CheckPoolHealth(s.poolOptimizer, ctx)
	report.Components["database_pool"] = ComponentHealth{
		Healthy: poolHealth.Healthy,
		Issues:  poolHealth.Issues,
	}
	if !poolHealth.Healthy {
		report.Healthy = false
	}

	// Check API performance health
	apiHealth := s.monitor.CheckHealth(500 * time.Millisecond)
	report.Components["api_performance"] = ComponentHealth{
		Healthy: apiHealth.Healthy,
		Issues:  apiHealth.Issues,
	}
	if !apiHealth.Healthy {
		report.Healthy = false
	}

	return report
}

// HealthReport represents overall health status
type HealthReport struct {
	Timestamp  time.Time                  `json:"timestamp"`
	Healthy    bool                       `json:"healthy"`
	Components map[string]ComponentHealth `json:"components"`
}

// ComponentHealth represents health of a single component
type ComponentHealth struct {
	Healthy bool     `json:"healthy"`
	Issues  []string `json:"issues,omitempty"`
}

// GetMonitor returns the performance monitor
func (s *Service) GetMonitor() *Monitor {
	return s.monitor
}

// GetPoolOptimizer returns the pool optimizer
func (s *Service) GetPoolOptimizer() *PoolOptimizer {
	return s.poolOptimizer
}

// GetQueryOptimizer returns the query optimizer
func (s *Service) GetQueryOptimizer() *QueryOptimizer {
	return s.queryOptimizer
}

// Reset resets all performance metrics
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.monitor.Reset()
	s.queryOptimizer.ClearSlowQueries()
}

// OptimizationSuggestions provides specific optimization suggestions
type OptimizationSuggestions struct {
	Critical []string `json:"critical"`
	Warning  []string `json:"warning"`
	Info     []string `json:"info"`
}

// GetOptimizationSuggestions analyzes performance and provides categorized suggestions
func (s *Service) GetOptimizationSuggestions() *OptimizationSuggestions {
	suggestions := &OptimizationSuggestions{
		Critical: make([]string, 0),
		Warning:  make([]string, 0),
		Info:     make([]string, 0),
	}

	// Analyze API performance
	summary := s.monitor.GetSummary()

	// Critical issues
	if summary.ErrorRate > 0.10 {
		suggestions.Critical = append(suggestions.Critical,
			fmt.Sprintf("CRITICAL: Error rate %.2f%% exceeds 10%%", summary.ErrorRate*100))
	}

	if summary.AverageDuration > 2*time.Second {
		suggestions.Critical = append(suggestions.Critical,
			fmt.Sprintf("CRITICAL: Average response time %.2fs exceeds 2s", summary.AverageDuration.Seconds()))
	}

	// Warnings
	if summary.ErrorRate > 0.05 {
		suggestions.Warning = append(suggestions.Warning,
			fmt.Sprintf("WARNING: Error rate %.2f%% exceeds 5%%", summary.ErrorRate*100))
	}

	if summary.AverageDuration > 500*time.Millisecond {
		suggestions.Warning = append(suggestions.Warning,
			fmt.Sprintf("WARNING: Average response time %.2fs exceeds 500ms", summary.AverageDuration.Seconds()))
	}

	// Pool warnings
	poolStats := s.poolOptimizer.GetStats()
	if poolStats.WaitCount > 50 {
		suggestions.Warning = append(suggestions.Warning,
			fmt.Sprintf("WARNING: Database connection wait count is %d", poolStats.WaitCount))
	}

	// Informational
	slowQueries := s.queryOptimizer.GetSlowQueries(0)
	if len(slowQueries) > 0 {
		suggestions.Info = append(suggestions.Info,
			fmt.Sprintf("INFO: %d slow queries detected", len(slowQueries)))
	}

	slowRequests := s.monitor.GetSlowRequests(0)
	if len(slowRequests) > 0 {
		suggestions.Info = append(suggestions.Info,
			fmt.Sprintf("INFO: %d slow requests detected", len(slowRequests)))
	}

	return suggestions
}

// PerformanceMetrics provides detailed metrics for monitoring systems
type PerformanceMetrics struct {
	API struct {
		TotalRequests   int64         `json:"total_requests"`
		ErrorRate       float64       `json:"error_rate"`
		AverageDuration time.Duration `json:"average_duration"`
		P50Duration     time.Duration `json:"p50_duration"`
		P95Duration     time.Duration `json:"p95_duration"`
		P99Duration     time.Duration `json:"p99_duration"`
	} `json:"api"`

	Database struct {
		OpenConnections int   `json:"open_connections"`
		InUse           int   `json:"in_use"`
		Idle            int   `json:"idle"`
		WaitCount       int64 `json:"wait_count"`
	} `json:"database"`

	Queries struct {
		TotalSlowQueries int `json:"total_slow_queries"`
	} `json:"queries"`
}

// GetMetrics returns detailed performance metrics
func (s *Service) GetMetrics() *PerformanceMetrics {
	metrics := &PerformanceMetrics{}

	// API metrics
	summary := s.monitor.GetSummary()
	metrics.API.TotalRequests = summary.TotalRequests
	metrics.API.ErrorRate = summary.ErrorRate
	metrics.API.AverageDuration = summary.AverageDuration

	// Get endpoint metrics for percentiles
	allMetrics := s.monitor.GetMetrics()
	for _, em := range allMetrics {
		if em.Count > 0 {
			// Use first endpoint's percentiles as representative
			metrics.API.P50Duration = em.P50Duration
			metrics.API.P95Duration = em.P95Duration
			metrics.API.P99Duration = em.P99Duration
			break
		}
	}

	// Database metrics
	poolStats := s.poolOptimizer.GetStats()
	metrics.Database.OpenConnections = poolStats.OpenConnections
	metrics.Database.InUse = poolStats.InUse
	metrics.Database.Idle = poolStats.Idle
	metrics.Database.WaitCount = poolStats.WaitCount

	// Query metrics
	slowQueries := s.queryOptimizer.GetSlowQueries(0)
	metrics.Queries.TotalSlowQueries = len(slowQueries)

	return metrics
}

// PerformanceAlert represents a performance alert
type PerformanceAlertLevel string

const (
	AlertLevelInfo     PerformanceAlertLevel = "info"
	AlertLevelWarning  PerformanceAlertLevel = "warning"
	AlertLevelCritical PerformanceAlertLevel = "critical"
)

// Alert represents a performance alert
type Alert struct {
	Level     PerformanceAlertLevel `json:"level"`
	Component string                `json:"component"`
	Message   string                `json:"message"`
	Timestamp time.Time             `json:"timestamp"`
	Value     interface{}           `json:"value,omitempty"`
}

// GetAlerts returns current performance alerts
func (s *Service) GetAlerts() []Alert {
	alerts := make([]Alert, 0)
	now := time.Now()

	// API alerts
	apiAlerts := s.monitor.CheckAlerts(500, 0.05)
	for _, a := range apiAlerts {
		level := AlertLevelWarning
		if a.Level == "critical" {
			level = AlertLevelCritical
		}

		alerts = append(alerts, Alert{
			Level:     level,
			Component: "api",
			Message:   fmt.Sprintf("%s: %s", a.Endpoint, a.Message),
			Timestamp: now,
			Value:     a.Value,
		})
	}

	// Pool alerts
	poolStats := s.poolOptimizer.GetStats()
	if poolStats.WaitCount > 100 {
		alerts = append(alerts, Alert{
			Level:     AlertLevelCritical,
			Component: "database_pool",
			Message:   "High connection wait count",
			Timestamp: now,
			Value:     poolStats.WaitCount,
		})
	}

	return alerts
}
