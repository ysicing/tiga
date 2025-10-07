package base

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// BaseManager provides common functionality for all service managers
// Concrete service managers should embed this struct
type BaseManager struct {
	db          *gorm.DB
	serviceType string
	logger      Logger
}

// Logger interface for logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// defaultLogger is a simple logger implementation
type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, args ...interface{}) {
	log.Printf("[DEBUG] "+msg, args...)
}

func (l *defaultLogger) Info(msg string, args ...interface{}) {
	log.Printf("[INFO] "+msg, args...)
}

func (l *defaultLogger) Warn(msg string, args ...interface{}) {
	log.Printf("[WARN] "+msg, args...)
}

func (l *defaultLogger) Error(msg string, args ...interface{}) {
	log.Printf("[ERROR] "+msg, args...)
}

// NewBaseManager creates a new BaseManager
func NewBaseManager(db *gorm.DB, serviceType string) *BaseManager {
	return &BaseManager{
		db:          db,
		serviceType: serviceType,
		logger:      &defaultLogger{},
	}
}

// SetLogger sets a custom logger
func (m *BaseManager) SetLogger(logger Logger) {
	m.logger = logger
}

// Type returns the service type
func (m *BaseManager) Type() string {
	return m.serviceType
}

// DB returns the database connection
func (m *BaseManager) DB() *gorm.DB {
	return m.db
}

// Get retrieves an instance by ID
func (m *BaseManager) Get(ctx context.Context, id uuid.UUID) (*models.Instance, error) {
	var instance models.Instance

	err := m.db.WithContext(ctx).
		Preload("Owner").
		Where("id = ? AND type = ? AND deleted_at IS NULL", id, m.serviceType).
		First(&instance).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("instance %s not found", id)
		}
		m.logger.Error("Failed to get instance: %v", err)
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	return &instance, nil
}

// List retrieves instances based on filter
func (m *BaseManager) List(ctx context.Context, filter *Filter) ([]*models.Instance, error) {
	var instances []*models.Instance

	query := m.db.WithContext(ctx).
		Preload("Owner").
		Where("type = ? AND deleted_at IS NULL", m.serviceType)

	// Apply filters
	if filter != nil {
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
		if filter.Health != "" {
			query = query.Where("health = ?", filter.Health)
		}
		if filter.Environment != "" {
			query = query.Where("environment = ?", filter.Environment)
		}
		if filter.Team != "" {
			query = query.Where("team = ?", filter.Team)
		}
		if filter.OwnerID != nil {
			query = query.Where("owner_id = ?", *filter.OwnerID)
		}
		if filter.Search != "" {
			query = query.Where(
				"to_tsvector('simple', COALESCE(name, '') || ' ' || COALESCE(display_name, '') || ' ' || COALESCE(description, '')) @@ plainto_tsquery('simple', ?)",
				filter.Search,
			)
		}

		// Sorting
		sortBy := "created_at"
		if filter.SortBy != "" {
			sortBy = filter.SortBy
		}
		sortOrder := "DESC"
		if filter.SortOrder == "asc" {
			sortOrder = "ASC"
		}
		query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

		// Pagination
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
	}

	err := query.Find(&instances).Error
	if err != nil {
		m.logger.Error("Failed to list instances: %v", err)
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	return instances, nil
}

// Delete soft-deletes an instance
func (m *BaseManager) Delete(ctx context.Context, id uuid.UUID) error {
	// Get instance first to verify it exists and belongs to this service type
	instance, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	// Soft delete
	err = m.db.WithContext(ctx).Delete(instance).Error
	if err != nil {
		m.logger.Error("Failed to delete instance %s: %v", id, err)
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	m.logger.Info("Deleted instance %s (%s)", instance.Name, id)
	return nil
}

// CreateSnapshot creates a snapshot of the instance configuration
func (m *BaseManager) CreateSnapshot(ctx context.Context, instance *models.Instance, changeType string, changedFields []string, reason string, userID *uuid.UUID) error {
	// Convert instance to JSONB for snapshot
	snapshotData := models.JSONB{
		"id":          instance.ID,
		"name":        instance.Name,
		"type":        instance.Type,
		"connection":  instance.Connection,
		"config":      instance.Config,
		"status":      instance.Status,
		"health":      instance.Health,
		"version":     instance.Version,
		"tags":        instance.Tags,
		"labels":      instance.Labels,
		"environment": instance.Environment,
		"owner_id":    instance.OwnerID,
		"team":        instance.Team,
	}

	snapshot := &models.InstanceSnapshot{
		InstanceID:    instance.ID,
		Snapshot:      snapshotData,
		ChangeType:    changeType,
		ChangedFields: models.StringArray(changedFields),
		ChangeReason:  reason,
		CreatedBy:     userID,
	}

	err := m.db.WithContext(ctx).Create(snapshot).Error
	if err != nil {
		m.logger.Error("Failed to create snapshot for instance %s: %v", instance.ID, err)
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	m.logger.Debug("Created snapshot for instance %s, change type: %s", instance.ID, changeType)
	return nil
}

// UpdateHealthStatus updates the health status of an instance
func (m *BaseManager) UpdateHealthStatus(ctx context.Context, id uuid.UUID, health *HealthStatus) error {
	now := time.Now()
	updates := map[string]interface{}{
		"health":            health.Status,
		"health_message":    health.Message,
		"last_health_check": now,
	}

	err := m.db.WithContext(ctx).
		Model(&models.Instance{}).
		Where("id = ? AND type = ?", id, m.serviceType).
		Updates(updates).Error

	if err != nil {
		m.logger.Error("Failed to update health status for instance %s: %v", id, err)
		return fmt.Errorf("failed to update health status: %w", err)
	}

	m.logger.Debug("Updated health status for instance %s: %s", id, health.Status)
	return nil
}

// SaveMetric saves a metric to the database
func (m *BaseManager) SaveMetric(ctx context.Context, instanceID uuid.UUID, metricName string, value float64, labels map[string]interface{}) error {
	metric := &models.Metric{
		Time:       time.Now(),
		InstanceID: instanceID,
		MetricName: metricName,
		MetricType: MetricTypeGauge,
		Value:      value,
		Labels:     models.JSONB(labels),
	}

	err := m.db.WithContext(ctx).Create(metric).Error
	if err != nil {
		m.logger.Error("Failed to save metric %s for instance %s: %v", metricName, instanceID, err)
		return fmt.Errorf("failed to save metric: %w", err)
	}

	return nil
}

// GetMetricsFromDB retrieves metrics from the database
func (m *BaseManager) GetMetricsFromDB(ctx context.Context, instanceID uuid.UUID, timeRange TimeRange, metricNames []string) (map[string]MetricData, error) {
	var metrics []models.Metric

	query := m.db.WithContext(ctx).
		Where("instance_id = ? AND time >= ? AND time <= ?", instanceID, timeRange.Start, timeRange.End)

	if len(metricNames) > 0 {
		query = query.Where("metric_name IN ?", metricNames)
	}

	err := query.Order("time ASC").Find(&metrics).Error
	if err != nil {
		m.logger.Error("Failed to get metrics for instance %s: %v", instanceID, err)
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	// Group metrics by name
	result := make(map[string]MetricData)
	for _, metric := range metrics {
		if _, exists := result[metric.MetricName]; !exists {
			result[metric.MetricName] = MetricData{
				Name:   metric.MetricName,
				Type:   metric.MetricType,
				Points: []DataPoint{},
			}
		}

		data := result[metric.MetricName]
		data.Points = append(data.Points, DataPoint{
			Timestamp: metric.Time,
			Value:     metric.Value,
		})
		result[metric.MetricName] = data
	}

	return result, nil
}

// ValidateInstanceConfig validates basic instance configuration
func (m *BaseManager) ValidateInstanceConfig(config *InstanceConfig) error {
	if config == nil {
		return fmt.Errorf("instance config is required")
	}
	if config.Name == "" {
		return fmt.Errorf("instance name is required")
	}
	if config.Type != m.serviceType {
		return fmt.Errorf("instance type mismatch: expected %s, got %s", m.serviceType, config.Type)
	}
	if config.Connection.Host == "" {
		return fmt.Errorf("connection host is required")
	}
	if config.OwnerID == uuid.Nil {
		return fmt.Errorf("owner ID is required")
	}
	return nil
}
