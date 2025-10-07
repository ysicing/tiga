package handlers

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// MetricsHandler handles metrics endpoints
type MetricsHandler struct {
	metricsRepo *repository.MetricsRepository
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(metricsRepo *repository.MetricsRepository) *MetricsHandler {
	return &MetricsHandler{
		metricsRepo: metricsRepo,
	}
}

// QueryMetricsRequest represents a request to query metrics
type QueryMetricsRequest struct {
	InstanceID string `form:"instance_id" binding:"omitempty,uuid"`
	MetricName string `form:"metric_name"`
	MetricType string `form:"metric_type"`
	StartTime  string `form:"start_time"` // RFC3339 format
	EndTime    string `form:"end_time"`   // RFC3339 format
	Limit      int    `form:"limit" binding:"min=0,max=10000"`
	OrderBy    string `form:"order_by" binding:"omitempty,oneof=asc desc"`
}

// QueryMetrics queries metrics with filters
// @Summary Query metrics
// @Description Query metrics with time range and filters
// @Tags metrics
// @Produce json
// @Security BearerAuth
// @Param instance_id query string false "Filter by instance ID (UUID)"
// @Param metric_name query string false "Filter by metric name"
// @Param metric_type query string false "Filter by metric type"
// @Param start_time query string false "Start time (RFC3339)"
// @Param end_time query string false "End time (RFC3339)"
// @Param limit query int false "Limit results" default(1000)
// @Param order_by query string false "Order by time (asc, desc)" default(desc)
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/metrics [get]
func (h *MetricsHandler) QueryMetrics(c *gin.Context) {
	var req QueryMetricsRequest
	if !BindQuery(c, &req) {
		return
	}

	// Build filter
	filter := &repository.MetricsQueryFilter{
		MetricName:  req.MetricName,
		MetricType:  req.MetricType,
		Limit:       defaultInt(req.Limit, 1000),
		OrderByTime: defaultIfEmpty(req.OrderBy, "DESC"),
	}

	// Parse instance ID if provided
	if req.InstanceID != "" {
		instanceID, err := ParseUUID(req.InstanceID)
		if err != nil {
			RespondBadRequest(c, err)
			return
		}
		filter.InstanceID = &instanceID
	}

	// Parse time range
	if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			RespondBadRequest(c, err)
			return
		}
		filter.StartTime = startTime
	}

	if req.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			RespondBadRequest(c, err)
			return
		}
		filter.EndTime = endTime
	}

	// Query metrics
	metrics, err := h.metricsRepo.QueryMetrics(c.Request.Context(), filter)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, metrics)
}

// CreateMetricRequest represents a request to create a metric
type CreateMetricRequest struct {
	InstanceID string                 `json:"instance_id" binding:"required,uuid"`
	MetricName string                 `json:"metric_name" binding:"required"`
	MetricType string                 `json:"metric_type" binding:"required"`
	Value      float64                `json:"value" binding:"required"`
	Labels     map[string]interface{} `json:"labels,omitempty"`
	Timestamp  *string                `json:"timestamp,omitempty"` // RFC3339 format, defaults to now
}

// CreateMetric creates a new metric entry
// @Summary Create metric
// @Description Create a new metric data point
// @Tags metrics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateMetricRequest true "Metric creation request"
// @Success 201 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/metrics [post]
func (h *MetricsHandler) CreateMetric(c *gin.Context) {
	var req CreateMetricRequest
	if !BindJSON(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	// Parse timestamp
	var timestamp time.Time
	if req.Timestamp != nil {
		timestamp, err = time.Parse(time.RFC3339, *req.Timestamp)
		if err != nil {
			RespondBadRequest(c, err)
			return
		}
	} else {
		timestamp = time.Now()
	}

	// Create metric
	metric := &models.Metric{
		InstanceID: instanceID,
		MetricName: req.MetricName,
		MetricType: req.MetricType,
		Value:      req.Value,
		Labels:     req.Labels,
		Time:       timestamp,
	}

	if err := h.metricsRepo.Create(c.Request.Context(), metric); err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondCreated(c, metric)
}

// CreateMetricsBatchRequest represents a batch metric creation request
type CreateMetricsBatchRequest struct {
	Metrics []CreateMetricRequest `json:"metrics" binding:"required,min=1,max=1000"`
}

// CreateMetricsBatch creates multiple metrics in batch
// @Summary Create metrics batch
// @Description Create multiple metric data points in batch
// @Tags metrics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateMetricsBatchRequest true "Batch metric creation request"
// @Success 201 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/metrics/batch [post]
func (h *MetricsHandler) CreateMetricsBatch(c *gin.Context) {
	var req CreateMetricsBatchRequest
	if !BindJSON(c, &req) {
		return
	}

	metrics := make([]*models.Metric, 0, len(req.Metrics))
	for _, metricReq := range req.Metrics {
		instanceID, err := ParseUUID(metricReq.InstanceID)
		if err != nil {
			RespondBadRequest(c, err)
			return
		}

		var timestamp time.Time
		if metricReq.Timestamp != nil {
			timestamp, err = time.Parse(time.RFC3339, *metricReq.Timestamp)
			if err != nil {
				RespondBadRequest(c, err)
				return
			}
		} else {
			timestamp = time.Now()
		}

		metric := &models.Metric{
			InstanceID: instanceID,
			MetricName: metricReq.MetricName,
			MetricType: metricReq.MetricType,
			Value:      metricReq.Value,
			Labels:     metricReq.Labels,
			Time:       timestamp,
		}
		metrics = append(metrics, metric)
	}

	if err := h.metricsRepo.CreateBatch(c.Request.Context(), metrics); err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondCreated(c, gin.H{
		"count":   len(metrics),
		"message": "metrics created successfully",
	})
}

// AggregateMetricsRequest represents a request to aggregate metrics
type AggregateMetricsRequest struct {
	InstanceID string `form:"instance_id" binding:"required,uuid"`
	MetricName string `form:"metric_name" binding:"required"`
	StartTime  string `form:"start_time" binding:"required"` // RFC3339
	EndTime    string `form:"end_time" binding:"required"`   // RFC3339
	Interval   string `form:"interval" binding:"required"`   // e.g., "5 minutes", "1 hour"
}

// AggregateMetrics aggregates metrics over time buckets
// @Summary Aggregate metrics
// @Description Aggregate metrics over time buckets using TimescaleDB
// @Tags metrics
// @Produce json
// @Security BearerAuth
// @Param instance_id query string true "Instance ID (UUID)"
// @Param metric_name query string true "Metric name"
// @Param start_time query string true "Start time (RFC3339)"
// @Param end_time query string true "End time (RFC3339)"
// @Param interval query string true "Aggregation interval (e.g., '5 minutes', '1 hour')"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/metrics/aggregate [get]
func (h *MetricsHandler) AggregateMetrics(c *gin.Context) {
	var req AggregateMetricsRequest
	if !BindQuery(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	// Aggregate metrics
	aggregated, err := h.metricsRepo.AggregateMetrics(c.Request.Context(), instanceID, req.MetricName, startTime, endTime, req.Interval)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, aggregated)
}

// GetLatestMetricRequest represents a request to get latest metric
type GetLatestMetricRequest struct {
	InstanceID string `form:"instance_id" binding:"required,uuid"`
	MetricName string `form:"metric_name" binding:"required"`
}

// GetLatestMetric gets the latest metric for an instance and metric name
// @Summary Get latest metric
// @Description Get the most recent metric value
// @Tags metrics
// @Produce json
// @Security BearerAuth
// @Param instance_id query string true "Instance ID (UUID)"
// @Param metric_name query string true "Metric name"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/metrics/latest [get]
func (h *MetricsHandler) GetLatestMetric(c *gin.Context) {
	var req GetLatestMetricRequest
	if !BindQuery(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	metric, err := h.metricsRepo.GetLatestMetric(c.Request.Context(), instanceID, req.MetricName)
	if err != nil {
		RespondNotFound(c, err)
		return
	}

	RespondSuccess(c, metric)
}

// GetLatestMetricsRequest represents a request to get all latest metrics
type GetLatestMetricsRequest struct {
	InstanceID string `uri:"instance_id" binding:"required,uuid"`
}

// GetLatestMetrics gets all latest metrics for an instance
// @Summary Get all latest metrics
// @Description Get the most recent value for all metrics of an instance
// @Tags metrics
// @Produce json
// @Security BearerAuth
// @Param instance_id path string true "Instance ID (UUID)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/instances/{instance_id}/metrics/latest [get]
func (h *MetricsHandler) GetLatestMetrics(c *gin.Context) {
	var req GetLatestMetricsRequest
	if !BindURI(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	metrics, err := h.metricsRepo.GetLatestMetrics(c.Request.Context(), instanceID)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, metrics)
}

// GetTimeSeriesRequest represents a request to get time series data
type GetTimeSeriesRequest struct {
	InstanceID string `form:"instance_id" binding:"required,uuid"`
	MetricName string `form:"metric_name" binding:"required"`
	StartTime  string `form:"start_time" binding:"required"` // RFC3339
	EndTime    string `form:"end_time" binding:"required"`   // RFC3339
	Interval   string `form:"interval" binding:"required"`   // e.g., "5 minutes"
}

// GetTimeSeries gets time series data for visualization
// @Summary Get time series data
// @Description Get time series data points for charting
// @Tags metrics
// @Produce json
// @Security BearerAuth
// @Param instance_id query string true "Instance ID (UUID)"
// @Param metric_name query string true "Metric name"
// @Param start_time query string true "Start time (RFC3339)"
// @Param end_time query string true "End time (RFC3339)"
// @Param interval query string true "Data point interval (e.g., '5 minutes')"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/metrics/timeseries [get]
func (h *MetricsHandler) GetTimeSeries(c *gin.Context) {
	var req GetTimeSeriesRequest
	if !BindQuery(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	timeSeries, err := h.metricsRepo.GetTimeSeries(c.Request.Context(), instanceID, req.MetricName, startTime, endTime, req.Interval)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, timeSeries)
}

// GetMetricNamesRequest represents a request to get metric names
type GetMetricNamesRequest struct {
	InstanceID string `uri:"instance_id" binding:"required,uuid"`
}

// GetMetricNames gets all distinct metric names for an instance
// @Summary Get metric names
// @Description Get all available metric names for an instance
// @Tags metrics
// @Produce json
// @Security BearerAuth
// @Param instance_id path string true "Instance ID (UUID)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/instances/{instance_id}/metrics/names [get]
func (h *MetricsHandler) GetMetricNames(c *gin.Context) {
	var req GetMetricNamesRequest
	if !BindURI(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	names, err := h.metricsRepo.GetMetricNames(c.Request.Context(), instanceID)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, names)
}

// GetMetricsStatistics gets metrics statistics
// @Summary Get metrics statistics
// @Description Get overall metrics statistics
// @Tags metrics
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/metrics/statistics [get]
func (h *MetricsHandler) GetMetricsStatistics(c *gin.Context) {
	stats, err := h.metricsRepo.GetStatistics(c.Request.Context())
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, stats)
}

// CalculateAverageRequest represents a request to calculate average
type CalculateAverageRequest struct {
	InstanceID string `form:"instance_id" binding:"required,uuid"`
	MetricName string `form:"metric_name" binding:"required"`
	StartTime  string `form:"start_time" binding:"required"` // RFC3339
	EndTime    string `form:"end_time" binding:"required"`   // RFC3339
}

// CalculateAverage calculates average value for a metric
// @Summary Calculate average
// @Description Calculate average metric value over time range
// @Tags metrics
// @Produce json
// @Security BearerAuth
// @Param instance_id query string true "Instance ID (UUID)"
// @Param metric_name query string true "Metric name"
// @Param start_time query string true "Start time (RFC3339)"
// @Param end_time query string true "End time (RFC3339)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/metrics/average [get]
func (h *MetricsHandler) CalculateAverage(c *gin.Context) {
	var req CalculateAverageRequest
	if !BindQuery(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	avg, err := h.metricsRepo.CalculateAverage(c.Request.Context(), instanceID, req.MetricName, startTime, endTime)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, gin.H{
		"average":     avg,
		"instance_id": req.InstanceID,
		"metric_name": req.MetricName,
		"start_time":  startTime,
		"end_time":    endTime,
	})
}

// CalculatePercentileRequest represents a request to calculate percentile
type CalculatePercentileRequest struct {
	InstanceID string  `form:"instance_id" binding:"required,uuid"`
	MetricName string  `form:"metric_name" binding:"required"`
	StartTime  string  `form:"start_time" binding:"required"` // RFC3339
	EndTime    string  `form:"end_time" binding:"required"`   // RFC3339
	Percentile float64 `form:"percentile" binding:"required,min=0,max=1"`
}

// CalculatePercentile calculates percentile value for a metric
// @Summary Calculate percentile
// @Description Calculate percentile (p50, p95, p99) for a metric
// @Tags metrics
// @Produce json
// @Security BearerAuth
// @Param instance_id query string true "Instance ID (UUID)"
// @Param metric_name query string true "Metric name"
// @Param start_time query string true "Start time (RFC3339)"
// @Param end_time query string true "End time (RFC3339)"
// @Param percentile query number true "Percentile value (0-1, e.g., 0.95 for p95)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/metrics/percentile [get]
func (h *MetricsHandler) CalculatePercentile(c *gin.Context) {
	var req CalculatePercentileRequest
	if !BindQuery(c, &req) {
		return
	}

	instanceID, err := ParseUUID(req.InstanceID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	percentile, err := h.metricsRepo.CalculatePercentile(c.Request.Context(), instanceID, req.MetricName, startTime, endTime, req.Percentile)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, gin.H{
		"percentile":  req.Percentile,
		"value":       percentile,
		"instance_id": req.InstanceID,
		"metric_name": req.MetricName,
		"start_time":  startTime,
		"end_time":    endTime,
	})
}
