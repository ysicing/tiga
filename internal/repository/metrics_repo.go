package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// MetricsRepository handles metrics data operations (TimescaleDB optimized)
type MetricsRepository struct {
	db *gorm.DB
}

// NewMetricsRepository creates a new MetricsRepository
func NewMetricsRepository(db *gorm.DB) *MetricsRepository {
	return &MetricsRepository{db: db}
}

// Create creates a new metric entry
func (r *MetricsRepository) Create(ctx context.Context, metric *models.Metric) error {
	if err := r.db.WithContext(ctx).Create(metric).Error; err != nil {
		return fmt.Errorf("failed to create metric: %w", err)
	}
	return nil
}

// CreateBatch creates multiple metric entries in batch
func (r *MetricsRepository) CreateBatch(ctx context.Context, metrics []*models.Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).CreateInBatches(metrics, 100).Error; err != nil {
		return fmt.Errorf("failed to create metrics batch: %w", err)
	}
	return nil
}

// GetByID retrieves a metric by ID
func (r *MetricsRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Metric, error) {
	var metric models.Metric
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&metric).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("metric not found")
		}
		return nil, fmt.Errorf("failed to get metric: %w", err)
	}

	return &metric, nil
}

// MetricsQueryFilter represents metrics query filters
type MetricsQueryFilter struct {
	InstanceID  *uuid.UUID // Filter by instance ID
	MetricName  string     // Filter by metric name
	MetricType  string     // Filter by metric type
	StartTime   time.Time  // Start time (inclusive)
	EndTime     time.Time  // End time (inclusive)
	Limit       int        // Limit results
	OrderByTime string     // "ASC" or "DESC", defaults to DESC
}

// QueryMetrics retrieves metrics with filters
func (r *MetricsRepository) QueryMetrics(ctx context.Context, filter *MetricsQueryFilter) ([]*models.Metric, error) {
	query := r.db.WithContext(ctx).Model(&models.Metric{})

	// Apply filters
	if filter.InstanceID != nil {
		query = query.Where("instance_id = ?", *filter.InstanceID)
	}

	if filter.MetricName != "" {
		query = query.Where("metric_name = ?", filter.MetricName)
	}

	if filter.MetricType != "" {
		query = query.Where("metric_type = ?", filter.MetricType)
	}

	if !filter.StartTime.IsZero() {
		query = query.Where("timestamp >= ?", filter.StartTime)
	}

	if !filter.EndTime.IsZero() {
		query = query.Where("timestamp <= ?", filter.EndTime)
	}

	// Ordering
	orderBy := "timestamp DESC"
	if filter.OrderByTime == "ASC" {
		orderBy = "timestamp ASC"
	}
	query = query.Order(orderBy)

	// Limit
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	// Fetch results
	var metrics []*models.Metric
	if err := query.Find(&metrics).Error; err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}

	return metrics, nil
}

// AggregatedMetric represents an aggregated metric result
type AggregatedMetric struct {
	Timestamp  time.Time              `json:"timestamp"`
	MetricName string                 `json:"metric_name"`
	MetricType string                 `json:"metric_type"`
	AvgValue   float64                `json:"avg_value"`
	MinValue   float64                `json:"min_value"`
	MaxValue   float64                `json:"max_value"`
	SumValue   float64                `json:"sum_value"`
	Count      int64                  `json:"count"`
	Labels     map[string]interface{} `json:"labels,omitempty"`
}

// AggregateMetrics aggregates metrics over time buckets
func (r *MetricsRepository) AggregateMetrics(ctx context.Context, instanceID uuid.UUID, metricName string, startTime, endTime time.Time, interval string) ([]*AggregatedMetric, error) {
	// Use TimescaleDB time_bucket function for time-series aggregation
	query := `
		SELECT
			time_bucket($1::interval, timestamp) AS timestamp,
			metric_name,
			metric_type,
			AVG(value) AS avg_value,
			MIN(value) AS min_value,
			MAX(value) AS max_value,
			SUM(value) AS sum_value,
			COUNT(*) AS count
		FROM metrics
		WHERE instance_id = $2
			AND metric_name = $3
			AND timestamp >= $4
			AND timestamp <= $5
		GROUP BY time_bucket($1::interval, timestamp), metric_name, metric_type
		ORDER BY timestamp DESC
	`

	var results []*AggregatedMetric
	err := r.db.WithContext(ctx).Raw(query, interval, instanceID, metricName, startTime, endTime).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate metrics: %w", err)
	}

	return results, nil
}

// GetLatestMetric retrieves the latest metric for an instance and metric name
func (r *MetricsRepository) GetLatestMetric(ctx context.Context, instanceID uuid.UUID, metricName string) (*models.Metric, error) {
	var metric models.Metric
	err := r.db.WithContext(ctx).
		Where("instance_id = ? AND metric_name = ?", instanceID, metricName).
		Order("timestamp DESC").
		First(&metric).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("metric not found")
		}
		return nil, fmt.Errorf("failed to get latest metric: %w", err)
	}

	return &metric, nil
}

// GetLatestMetrics retrieves the latest metrics for an instance (all metric names)
func (r *MetricsRepository) GetLatestMetrics(ctx context.Context, instanceID uuid.UUID) ([]*models.Metric, error) {
	// Use DISTINCT ON to get latest metric for each metric_name
	query := `
		SELECT DISTINCT ON (metric_name) *
		FROM metrics
		WHERE instance_id = $1
		ORDER BY metric_name, timestamp DESC
	`

	var metrics []*models.Metric
	err := r.db.WithContext(ctx).Raw(query, instanceID).Scan(&metrics).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get latest metrics: %w", err)
	}

	return metrics, nil
}

// DeleteOldMetrics deletes metrics older than a specified time
func (r *MetricsRepository) DeleteOldMetrics(ctx context.Context, olderThan time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("timestamp < ?", olderThan).
		Delete(&models.Metric{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old metrics: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// DeleteMetricsByInstance deletes all metrics for a specific instance
func (r *MetricsRepository) DeleteMetricsByInstance(ctx context.Context, instanceID uuid.UUID) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Delete(&models.Metric{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete instance metrics: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// CountMetrics counts metrics matching the filter
func (r *MetricsRepository) CountMetrics(ctx context.Context, filter *MetricsQueryFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Metric{})

	if filter.InstanceID != nil {
		query = query.Where("instance_id = ?", *filter.InstanceID)
	}

	if filter.MetricName != "" {
		query = query.Where("metric_name = ?", filter.MetricName)
	}

	if filter.MetricType != "" {
		query = query.Where("metric_type = ?", filter.MetricType)
	}

	if !filter.StartTime.IsZero() {
		query = query.Where("timestamp >= ?", filter.StartTime)
	}

	if !filter.EndTime.IsZero() {
		query = query.Where("timestamp <= ?", filter.EndTime)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count metrics: %w", err)
	}

	return count, nil
}

// GetMetricNames retrieves all distinct metric names for an instance
func (r *MetricsRepository) GetMetricNames(ctx context.Context, instanceID uuid.UUID) ([]string, error) {
	var names []string
	err := r.db.WithContext(ctx).
		Model(&models.Metric{}).
		Where("instance_id = ?", instanceID).
		Distinct("metric_name").
		Pluck("metric_name", &names).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get metric names: %w", err)
	}

	return names, nil
}

// GetMetricTypes retrieves all distinct metric types
func (r *MetricsRepository) GetMetricTypes(ctx context.Context) ([]string, error) {
	var types []string
	err := r.db.WithContext(ctx).
		Model(&models.Metric{}).
		Distinct("metric_type").
		Pluck("metric_type", &types).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get metric types: %w", err)
	}

	return types, nil
}

// CalculateAverage calculates average value for a metric over time range
func (r *MetricsRepository) CalculateAverage(ctx context.Context, instanceID uuid.UUID, metricName string, startTime, endTime time.Time) (float64, error) {
	var avg float64
	err := r.db.WithContext(ctx).
		Model(&models.Metric{}).
		Where("instance_id = ? AND metric_name = ? AND timestamp >= ? AND timestamp <= ?",
			instanceID, metricName, startTime, endTime).
		Select("AVG(value)").
		Scan(&avg).Error

	if err != nil {
		return 0, fmt.Errorf("failed to calculate average: %w", err)
	}

	return avg, nil
}

// CalculatePercentile calculates percentile value for a metric (e.g., p95, p99)
func (r *MetricsRepository) CalculatePercentile(ctx context.Context, instanceID uuid.UUID, metricName string, startTime, endTime time.Time, percentile float64) (float64, error) {
	// Use percentile_cont for continuous percentile calculation
	query := `
		SELECT percentile_cont($1) WITHIN GROUP (ORDER BY value)
		FROM metrics
		WHERE instance_id = $2
			AND metric_name = $3
			AND timestamp >= $4
			AND timestamp <= $5
	`

	var result float64
	err := r.db.WithContext(ctx).Raw(query, percentile, instanceID, metricName, startTime, endTime).Scan(&result).Error
	if err != nil {
		return 0, fmt.Errorf("failed to calculate percentile: %w", err)
	}

	return result, nil
}

// GetTimeSeries retrieves time-series data for visualization
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

func (r *MetricsRepository) GetTimeSeries(ctx context.Context, instanceID uuid.UUID, metricName string, startTime, endTime time.Time, interval string) ([]*TimeSeriesPoint, error) {
	// Use TimescaleDB time_bucket for downsampling
	query := `
		SELECT
			time_bucket($1::interval, timestamp) AS timestamp,
			AVG(value) AS value
		FROM metrics
		WHERE instance_id = $2
			AND metric_name = $3
			AND timestamp >= $4
			AND timestamp <= $5
		GROUP BY time_bucket($1::interval, timestamp)
		ORDER BY timestamp ASC
	`

	var points []*TimeSeriesPoint
	err := r.db.WithContext(ctx).Raw(query, interval, instanceID, metricName, startTime, endTime).Scan(&points).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get time series: %w", err)
	}

	return points, nil
}

// MetricsStatistics represents metrics statistics
type MetricsStatistics struct {
	TotalMetrics      int64            `json:"total_metrics"`
	ByMetricType      map[string]int64 `json:"by_metric_type"`
	OldestTimestamp   *time.Time       `json:"oldest_timestamp"`
	NewestTimestamp   *time.Time       `json:"newest_timestamp"`
	UniqueInstances   int64            `json:"unique_instances"`
	UniqueMetricNames int64            `json:"unique_metric_names"`
}

// GetStatistics retrieves metrics statistics
func (r *MetricsRepository) GetStatistics(ctx context.Context) (*MetricsStatistics, error) {
	stats := &MetricsStatistics{
		ByMetricType: make(map[string]int64),
	}

	// Total count
	if err := r.db.WithContext(ctx).Model(&models.Metric{}).Count(&stats.TotalMetrics).Error; err != nil {
		return nil, fmt.Errorf("failed to count total metrics: %w", err)
	}

	// Group by metric type
	type TypeCount struct {
		MetricType string
		Count      int64
	}
	var typeCounts []TypeCount
	if err := r.db.WithContext(ctx).
		Model(&models.Metric{}).
		Select("metric_type, COUNT(*) as count").
		Group("metric_type").
		Scan(&typeCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to count by metric type: %w", err)
	}
	for _, tc := range typeCounts {
		stats.ByMetricType[tc.MetricType] = tc.Count
	}

	// Oldest timestamp
	var oldest time.Time
	if err := r.db.WithContext(ctx).
		Model(&models.Metric{}).
		Select("MIN(timestamp)").
		Scan(&oldest).Error; err == nil && !oldest.IsZero() {
		stats.OldestTimestamp = &oldest
	}

	// Newest timestamp
	var newest time.Time
	if err := r.db.WithContext(ctx).
		Model(&models.Metric{}).
		Select("MAX(timestamp)").
		Scan(&newest).Error; err == nil && !newest.IsZero() {
		stats.NewestTimestamp = &newest
	}

	// Unique instances
	if err := r.db.WithContext(ctx).
		Model(&models.Metric{}).
		Distinct("instance_id").
		Count(&stats.UniqueInstances).Error; err != nil {
		return nil, fmt.Errorf("failed to count unique instances: %w", err)
	}

	// Unique metric names
	if err := r.db.WithContext(ctx).
		Model(&models.Metric{}).
		Distinct("metric_name").
		Count(&stats.UniqueMetricNames).Error; err != nil {
		return nil, fmt.Errorf("failed to count unique metric names: %w", err)
	}

	return stats, nil
}
