package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/ysicing/tiga/internal/repository"
)

// PrometheusService manages Prometheus metrics collection
type PrometheusService struct {
	instanceRepo *repository.InstanceRepository
	alertRepo    *repository.AlertRepository
	auditRepo    *repository.AuditLogRepository

	// Instance metrics
	instancesTotal      *prometheus.GaugeVec
	instancesHealthy    *prometheus.GaugeVec
	instancesUnhealthy  *prometheus.GaugeVec
	instanceConnections *prometheus.GaugeVec

	// Alert metrics
	alertsTotal        prometheus.Gauge
	alertsFiring       prometheus.Gauge
	alertEventsTotal   *prometheus.CounterVec
	alertNotifications *prometheus.CounterVec

	// Audit metrics
	auditLogsTotal *prometheus.CounterVec
	auditFailures  prometheus.Counter

	// API metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec

	// Database metrics
	dbConnectionsActive prometheus.Gauge
	dbConnectionsIdle   prometheus.Gauge
	dbQueryDuration     *prometheus.HistogramVec

	// System metrics
	uptimeSeconds prometheus.Gauge
}

// NewPrometheusService creates a new Prometheus service
func NewPrometheusService(
	instanceRepo *repository.InstanceRepository,
	alertRepo *repository.AlertRepository,
	auditRepo *repository.AuditLogRepository,
) *PrometheusService {
	s := &PrometheusService{
		instanceRepo: instanceRepo,
		alertRepo:    alertRepo,
		auditRepo:    auditRepo,
	}

	s.registerMetrics()
	return s
}

// registerMetrics registers all Prometheus metrics
func (s *PrometheusService) registerMetrics() {
	// Instance metrics
	s.instancesTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tiga_instances_total",
			Help: "Total number of managed instances",
		},
		[]string{"type", "status"},
	)

	s.instancesHealthy = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tiga_instances_healthy",
			Help: "Number of healthy instances",
		},
		[]string{"type"},
	)

	s.instancesUnhealthy = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tiga_instances_unhealthy",
			Help: "Number of unhealthy instances",
		},
		[]string{"type"},
	)

	s.instanceConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tiga_instance_connections",
			Help: "Number of active connections to instances",
		},
		[]string{"instance_id", "type"},
	)

	// Alert metrics
	s.alertsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tiga_alerts_total",
			Help: "Total number of configured alerts",
		},
	)

	s.alertsFiring = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tiga_alerts_firing",
			Help: "Number of currently firing alerts",
		},
	)

	s.alertEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tiga_alert_events_total",
			Help: "Total number of alert events",
		},
		[]string{"alert_id", "status"},
	)

	s.alertNotifications = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tiga_alert_notifications_total",
			Help: "Total number of alert notifications sent",
		},
		[]string{"channel", "status"},
	)

	// Audit metrics
	s.auditLogsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tiga_audit_logs_total",
			Help: "Total number of audit log entries",
		},
		[]string{"action", "status"},
	)

	s.auditFailures = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tiga_audit_failures_total",
			Help: "Total number of audit log failures",
		},
	)

	// API metrics
	s.httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tiga_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	s.httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tiga_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// Database metrics
	s.dbConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tiga_db_connections_active",
			Help: "Number of active database connections",
		},
	)

	s.dbConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tiga_db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	s.dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tiga_db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// System metrics
	s.uptimeSeconds = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tiga_uptime_seconds",
			Help: "Application uptime in seconds",
		},
	)
}

// CollectMetrics collects all metrics from repositories
func (s *PrometheusService) CollectMetrics(ctx context.Context) error {
	// Collect instance metrics
	if err := s.collectInstanceMetrics(ctx); err != nil {
		return err
	}

	// Collect alert metrics
	if err := s.collectAlertMetrics(ctx); err != nil {
		return err
	}

	// Collect audit metrics
	if err := s.collectAuditMetrics(ctx); err != nil {
		return err
	}

	return nil
}

// collectInstanceMetrics collects instance-related metrics
func (s *PrometheusService) collectInstanceMetrics(ctx context.Context) error {
	// Get instance statistics
	stats, err := s.instanceRepo.GetStatistics(ctx)
	if err != nil {
		return err
	}

	// Reset gauges
	s.instancesTotal.Reset()
	s.instancesHealthy.Reset()
	s.instancesUnhealthy.Reset()

	// Set metrics from statistics
	for serviceType, count := range stats.ByServiceType {
		s.instancesTotal.WithLabelValues(serviceType, "all").Set(float64(count))
	}

	for status, count := range stats.ByStatus {
		if status == "healthy" || status == "active" {
			s.instancesHealthy.WithLabelValues("all").Set(float64(count))
		} else {
			s.instancesUnhealthy.WithLabelValues("all").Set(float64(count))
		}
	}

	return nil
}

// collectAlertMetrics collects alert-related metrics
func (s *PrometheusService) collectAlertMetrics(ctx context.Context) error {
	// Get alert statistics
	stats, err := s.alertRepo.GetStatistics(ctx)
	if err != nil {
		return err
	}

	s.alertsTotal.Set(float64(stats.TotalRules))
	s.alertsFiring.Set(float64(stats.ActiveEvents))

	return nil
}

// collectAuditMetrics collects audit log metrics
func (s *PrometheusService) collectAuditMetrics(ctx context.Context) error {
	// Get audit statistics
	stats, err := s.auditRepo.GetStatistics(ctx)
	if err != nil {
		return err
	}

	// Update counters based on statistics
	// Note: Counters can only increase, so we track deltas
	for action, count := range stats.ByAction {
		s.auditLogsTotal.WithLabelValues(action, "success").Add(float64(count))
	}

	return nil
}

// RecordHTTPRequest records an HTTP request metric
func (s *PrometheusService) RecordHTTPRequest(method, endpoint, status string, duration time.Duration) {
	s.httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	s.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordAlertEvent records an alert event
func (s *PrometheusService) RecordAlertEvent(alertID, status string) {
	s.alertEventsTotal.WithLabelValues(alertID, status).Inc()
}

// RecordAlertNotification records an alert notification
func (s *PrometheusService) RecordAlertNotification(channel, status string) {
	s.alertNotifications.WithLabelValues(channel, status).Inc()
}

// RecordAuditLog records an audit log entry
func (s *PrometheusService) RecordAuditLog(action, status string) {
	s.auditLogsTotal.WithLabelValues(action, status).Inc()
	if status == "failure" {
		s.auditFailures.Inc()
	}
}

// RecordDBQuery records a database query
func (s *PrometheusService) RecordDBQuery(operation string, duration time.Duration) {
	s.dbQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// UpdateDBConnectionStats updates database connection statistics
func (s *PrometheusService) UpdateDBConnectionStats(active, idle int) {
	s.dbConnectionsActive.Set(float64(active))
	s.dbConnectionsIdle.Set(float64(idle))
}

// UpdateUptime updates the application uptime
func (s *PrometheusService) UpdateUptime(startTime time.Time) {
	s.uptimeSeconds.Set(time.Since(startTime).Seconds())
}

// StartBackgroundCollector starts a background goroutine to collect metrics periodically
func (s *PrometheusService) StartBackgroundCollector(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.CollectMetrics(ctx); err != nil {
				// Log error but continue
				continue
			}
		}
	}
}
