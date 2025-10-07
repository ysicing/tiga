package base

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector defines the interface for metrics collection
type MetricsCollector interface {
	// CollectMetrics collects metrics for an instance
	CollectMetrics(ctx context.Context, instanceID uuid.UUID) (map[string]float64, error)

	// GetMetricNames returns the list of metric names this collector provides
	GetMetricNames() []string
}

// MetricsConfig represents metrics collection configuration
type MetricsConfig struct {
	Enabled            bool          `json:"enabled"`
	CollectionInterval time.Duration `json:"collection_interval"` // How often to collect
	RetentionDays      int           `json:"retention_days"`      // How long to keep metrics

	// Prometheus configuration
	PrometheusEnabled bool   `json:"prometheus_enabled"`
	PrometheusPrefix  string `json:"prometheus_prefix"`

	// Type-specific configuration
	CustomMetrics map[string]interface{} `json:"custom_metrics,omitempty"`
}

// DefaultMetricsConfig returns default metrics configuration
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:            true,
		CollectionInterval: 30 * time.Second,
		RetentionDays:      90,
		PrometheusEnabled:  true,
		PrometheusPrefix:   "devops",
	}
}

// PrometheusMetrics holds Prometheus metric collectors
type PrometheusMetrics struct {
	// Common metrics
	InstanceUp    *prometheus.GaugeVec
	HealthStatus  *prometheus.GaugeVec
	ResponseTime  *prometheus.HistogramVec
	ErrorsTotal   *prometheus.CounterVec
	RequestsTotal *prometheus.CounterVec

	// Service-specific metrics can be added by concrete implementations
	CustomGauges     map[string]*prometheus.GaugeVec
	CustomCounters   map[string]*prometheus.CounterVec
	CustomHistograms map[string]*prometheus.HistogramVec

	mu sync.RWMutex
}

// NewPrometheusMetrics creates a new PrometheusMetrics instance
func NewPrometheusMetrics(serviceType string) *PrometheusMetrics {
	prefix := "devops"

	return &PrometheusMetrics{
		InstanceUp: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: prefix,
				Subsystem: serviceType,
				Name:      "instance_up",
				Help:      "Instance availability (1 = up, 0 = down)",
			},
			[]string{"instance_id", "instance_name", "environment"},
		),
		HealthStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: prefix,
				Subsystem: serviceType,
				Name:      "health_status",
				Help:      "Instance health status (1 = healthy, 0.5 = degraded, 0 = unhealthy)",
			},
			[]string{"instance_id", "instance_name", "environment"},
		),
		ResponseTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: prefix,
				Subsystem: serviceType,
				Name:      "response_time_seconds",
				Help:      "Response time in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"instance_id", "instance_name", "operation"},
		),
		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: prefix,
				Subsystem: serviceType,
				Name:      "errors_total",
				Help:      "Total number of errors",
			},
			[]string{"instance_id", "instance_name", "error_type"},
		),
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: prefix,
				Subsystem: serviceType,
				Name:      "requests_total",
				Help:      "Total number of requests",
			},
			[]string{"instance_id", "instance_name", "method", "status"},
		),
		CustomGauges:     make(map[string]*prometheus.GaugeVec),
		CustomCounters:   make(map[string]*prometheus.CounterVec),
		CustomHistograms: make(map[string]*prometheus.HistogramVec),
	}
}

// RegisterCustomGauge registers a custom gauge metric
func (pm *PrometheusMetrics) RegisterCustomGauge(name, help string, labels []string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.CustomGauges[name]; exists {
		return fmt.Errorf("gauge %s already registered", name)
	}

	gauge := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: help,
		},
		labels,
	)

	pm.CustomGauges[name] = gauge
	return nil
}

// RegisterCustomCounter registers a custom counter metric
func (pm *PrometheusMetrics) RegisterCustomCounter(name, help string, labels []string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.CustomCounters[name]; exists {
		return fmt.Errorf("counter %s already registered", name)
	}

	counter := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		},
		labels,
	)

	pm.CustomCounters[name] = counter
	return nil
}

// RegisterCustomHistogram registers a custom histogram metric
func (pm *PrometheusMetrics) RegisterCustomHistogram(name, help string, labels []string, buckets []float64) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.CustomHistograms[name]; exists {
		return fmt.Errorf("histogram %s already registered", name)
	}

	if buckets == nil {
		buckets = prometheus.DefBuckets
	}

	histogram := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    name,
			Help:    help,
			Buckets: buckets,
		},
		labels,
	)

	pm.CustomHistograms[name] = histogram
	return nil
}

// SetInstanceUp sets the instance up metric
func (pm *PrometheusMetrics) SetInstanceUp(instanceID, instanceName, environment string, up bool) {
	value := 0.0
	if up {
		value = 1.0
	}
	pm.InstanceUp.WithLabelValues(instanceID, instanceName, environment).Set(value)
}

// SetHealthStatus sets the health status metric
func (pm *PrometheusMetrics) SetHealthStatus(instanceID, instanceName, environment, status string) {
	value := 0.0
	switch status {
	case HealthHealthy:
		value = 1.0
	case HealthDegraded:
		value = 0.5
	case HealthUnhealthy:
		value = 0.0
	}
	pm.HealthStatus.WithLabelValues(instanceID, instanceName, environment).Set(value)
}

// ObserveResponseTime records a response time observation
func (pm *PrometheusMetrics) ObserveResponseTime(instanceID, instanceName, operation string, duration time.Duration) {
	pm.ResponseTime.WithLabelValues(instanceID, instanceName, operation).Observe(duration.Seconds())
}

// IncrementErrors increments the error counter
func (pm *PrometheusMetrics) IncrementErrors(instanceID, instanceName, errorType string) {
	pm.ErrorsTotal.WithLabelValues(instanceID, instanceName, errorType).Inc()
}

// IncrementRequests increments the request counter
func (pm *PrometheusMetrics) IncrementRequests(instanceID, instanceName, method, status string) {
	pm.RequestsTotal.WithLabelValues(instanceID, instanceName, method, status).Inc()
}

// MetricsCollectionScheduler manages scheduled metrics collection
type MetricsCollectionScheduler struct {
	manager   ServiceManager
	collector MetricsCollector
	interval  time.Duration
	stopCh    chan struct{}
	logger    Logger
}

// NewMetricsCollectionScheduler creates a new metrics collection scheduler
func NewMetricsCollectionScheduler(manager ServiceManager, collector MetricsCollector, interval time.Duration) *MetricsCollectionScheduler {
	return &MetricsCollectionScheduler{
		manager:   manager,
		collector: collector,
		interval:  interval,
		stopCh:    make(chan struct{}),
		logger:    &defaultLogger{},
	}
}

// SetLogger sets a custom logger
func (s *MetricsCollectionScheduler) SetLogger(logger Logger) {
	s.logger = logger
}

// Start starts the metrics collection scheduler
func (s *MetricsCollectionScheduler) Start(ctx context.Context, instanceID uuid.UUID) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.logger.Info("Starting metrics collection scheduler for instance %s (interval: %s)", instanceID, s.interval)

	// Collect initial metrics immediately
	s.collectMetrics(ctx, instanceID)

	for {
		select {
		case <-ticker.C:
			s.collectMetrics(ctx, instanceID)
		case <-s.stopCh:
			s.logger.Info("Stopping metrics collection scheduler for instance %s", instanceID)
			return
		case <-ctx.Done():
			s.logger.Info("Context canceled, stopping metrics collection scheduler for instance %s", instanceID)
			return
		}
	}
}

// Stop stops the metrics collection scheduler
func (s *MetricsCollectionScheduler) Stop() {
	close(s.stopCh)
}

// collectMetrics collects metrics for an instance
func (s *MetricsCollectionScheduler) collectMetrics(ctx context.Context, instanceID uuid.UUID) {
	start := time.Now()
	metrics, err := s.collector.CollectMetrics(ctx, instanceID)
	duration := time.Since(start)

	if err != nil {
		s.logger.Error("Failed to collect metrics for instance %s: %v (duration: %s)", instanceID, err, duration)
		return
	}

	s.logger.Debug("Collected %d metrics for instance %s (duration: %s)", len(metrics), instanceID, duration)

	// Note: Metrics saving should be handled by the concrete service manager
	// through its own mechanisms or by exposing a SaveMetric method in the interface
	// The scheduler is primarily for triggering metrics collection
}

// ParseMetricsConfig parses metrics configuration from a map
func ParseMetricsConfig(configMap map[string]interface{}) (*MetricsConfig, error) {
	config := DefaultMetricsConfig()

	if enabled, ok := configMap["enabled"].(bool); ok {
		config.Enabled = enabled
	}

	if interval, ok := configMap["collection_interval"].(float64); ok {
		config.CollectionInterval = time.Duration(interval) * time.Second
	}

	if retentionDays, ok := configMap["retention_days"].(float64); ok {
		config.RetentionDays = int(retentionDays)
	}

	if prometheusEnabled, ok := configMap["prometheus_enabled"].(bool); ok {
		config.PrometheusEnabled = prometheusEnabled
	}

	if prometheusPrefix, ok := configMap["prometheus_prefix"].(string); ok {
		config.PrometheusPrefix = prometheusPrefix
	}

	return config, nil
}

// AggregateMetrics aggregates metrics over a time range
func AggregateMetrics(points []DataPoint, aggregationType string) float64 {
	if len(points) == 0 {
		return 0
	}

	switch aggregationType {
	case "avg", "average":
		sum := 0.0
		for _, p := range points {
			sum += p.Value
		}
		return sum / float64(len(points))

	case "min", "minimum":
		min := points[0].Value
		for _, p := range points {
			if p.Value < min {
				min = p.Value
			}
		}
		return min

	case "max", "maximum":
		max := points[0].Value
		for _, p := range points {
			if p.Value > max {
				max = p.Value
			}
		}
		return max

	case "sum", "total":
		sum := 0.0
		for _, p := range points {
			sum += p.Value
		}
		return sum

	case "last", "latest":
		return points[len(points)-1].Value

	case "first":
		return points[0].Value

	default:
		return 0
	}
}
