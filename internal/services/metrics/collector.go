package metrics

import (
	"context"
	"database/sql"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// SystemCollector collects system-level metrics
type SystemCollector struct {
	// Go runtime metrics
	goroutines      prometheus.Gauge
	memoryAllocated prometheus.Gauge
	memoryInUse     prometheus.Gauge
	gcPauseSeconds  prometheus.Histogram

	// Application metrics
	startTime time.Time
}

// NewSystemCollector creates a new system metrics collector
func NewSystemCollector() *SystemCollector {
	sc := &SystemCollector{
		startTime: time.Now(),
	}

	sc.registerMetrics()
	return sc
}

// registerMetrics registers system metrics
func (sc *SystemCollector) registerMetrics() {
	sc.goroutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tiga_goroutines",
			Help: "Number of goroutines",
		},
	)

	sc.memoryAllocated = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tiga_memory_allocated_bytes",
			Help: "Bytes of allocated memory",
		},
	)

	sc.memoryInUse = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tiga_memory_in_use_bytes",
			Help: "Bytes of memory in use",
		},
	)

	sc.gcPauseSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "tiga_gc_pause_seconds",
			Help:    "GC pause duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)
}

// Collect collects system metrics
func (sc *SystemCollector) Collect() {
	// Goroutines
	sc.goroutines.Set(float64(runtime.NumGoroutine()))

	// Memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	sc.memoryAllocated.Set(float64(memStats.Alloc))
	sc.memoryInUse.Set(float64(memStats.Sys))

	// GC stats
	if memStats.NumGC > 0 {
		// Get last GC pause duration
		lastPause := memStats.PauseNs[(memStats.NumGC+255)%256]
		sc.gcPauseSeconds.Observe(float64(lastPause) / 1e9)
	}
}

// GetUptime returns application uptime
func (sc *SystemCollector) GetUptime() time.Duration {
	return time.Since(sc.startTime)
}

// DBStatsCollector collects database connection pool metrics
type DBStatsCollector struct {
	db *sql.DB

	maxOpenConnections prometheus.Gauge
	openConnections    prometheus.Gauge
	inUseConnections   prometheus.Gauge
	idleConnections    prometheus.Gauge
	waitCount          prometheus.Counter
	waitDuration       prometheus.Counter
	maxIdleClosed      prometheus.Counter
	maxLifetimeClosed  prometheus.Counter
}

// NewDBStatsCollector creates a new database stats collector
func NewDBStatsCollector(db *sql.DB, dbName string) *DBStatsCollector {
	dc := &DBStatsCollector{
		db: db,
	}

	dc.registerMetrics(dbName)
	return dc
}

// registerMetrics registers database metrics
func (dc *DBStatsCollector) registerMetrics(dbName string) {
	labels := prometheus.Labels{"database": dbName}

	dc.maxOpenConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name:        "tiga_db_max_open_connections",
			Help:        "Maximum number of open connections to the database",
			ConstLabels: labels,
		},
	)

	dc.openConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name:        "tiga_db_open_connections",
			Help:        "Number of established connections to the database",
			ConstLabels: labels,
		},
	)

	dc.inUseConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name:        "tiga_db_in_use_connections",
			Help:        "Number of connections currently in use",
			ConstLabels: labels,
		},
	)

	dc.idleConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name:        "tiga_db_idle_connections",
			Help:        "Number of idle connections",
			ConstLabels: labels,
		},
	)

	dc.waitCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name:        "tiga_db_wait_count_total",
			Help:        "Total number of connections waited for",
			ConstLabels: labels,
		},
	)

	dc.waitDuration = promauto.NewCounter(
		prometheus.CounterOpts{
			Name:        "tiga_db_wait_duration_seconds_total",
			Help:        "Total time blocked waiting for a new connection",
			ConstLabels: labels,
		},
	)

	dc.maxIdleClosed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name:        "tiga_db_max_idle_closed_total",
			Help:        "Total number of connections closed due to max idle",
			ConstLabels: labels,
		},
	)

	dc.maxLifetimeClosed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name:        "tiga_db_max_lifetime_closed_total",
			Help:        "Total number of connections closed due to max lifetime",
			ConstLabels: labels,
		},
	)
}

// Collect collects database statistics
func (dc *DBStatsCollector) Collect() {
	stats := dc.db.Stats()

	dc.maxOpenConnections.Set(float64(stats.MaxOpenConnections))
	dc.openConnections.Set(float64(stats.OpenConnections))
	dc.inUseConnections.Set(float64(stats.InUse))
	dc.idleConnections.Set(float64(stats.Idle))
	dc.waitCount.Add(float64(stats.WaitCount))
	dc.waitDuration.Add(stats.WaitDuration.Seconds())
	dc.maxIdleClosed.Add(float64(stats.MaxIdleClosed))
	dc.maxLifetimeClosed.Add(float64(stats.MaxLifetimeClosed))
}

// MetricsAggregator aggregates all metric collectors
type MetricsAggregator struct {
	prometheusService *PrometheusService
	systemCollector   *SystemCollector
	dbCollectors      []*DBStatsCollector
}

// NewMetricsAggregator creates a new metrics aggregator
func NewMetricsAggregator(prometheusService *PrometheusService, systemCollector *SystemCollector) *MetricsAggregator {
	return &MetricsAggregator{
		prometheusService: prometheusService,
		systemCollector:   systemCollector,
		dbCollectors:      make([]*DBStatsCollector, 0),
	}
}

// AddDBCollector adds a database stats collector
func (ma *MetricsAggregator) AddDBCollector(collector *DBStatsCollector) {
	ma.dbCollectors = append(ma.dbCollectors, collector)
}

// CollectAll collects all metrics
func (ma *MetricsAggregator) CollectAll(ctx context.Context) error {
	// Collect Prometheus service metrics
	if err := ma.prometheusService.CollectMetrics(ctx); err != nil {
		return err
	}

	// Collect system metrics
	ma.systemCollector.Collect()

	// Update uptime
	ma.prometheusService.UpdateUptime(ma.systemCollector.startTime)

	// Collect database metrics
	for _, dbCollector := range ma.dbCollectors {
		dbCollector.Collect()
	}

	return nil
}

// StartPeriodicCollection starts periodic metric collection
func (ma *MetricsAggregator) StartPeriodicCollection(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := ma.CollectAll(ctx); err != nil {
				// Log error but continue
				continue
			}
		}
	}
}
