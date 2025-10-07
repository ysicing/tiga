package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ysicing/tiga/internal/models"
	"gorm.io/gorm"
)

// ServiceFilter represents filtering options for service monitor queries
type ServiceFilter struct {
	Page     int
	PageSize int
	HostID   uint
	Type     string // HTTP/TCP/ICMP
	Enabled  *bool
	Search   string
}

// ServiceRepository defines the interface for service monitor data access
type ServiceRepository interface {
	Create(ctx context.Context, service *models.ServiceMonitor) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.ServiceMonitor, error)
	List(ctx context.Context, filter ServiceFilter) ([]*models.ServiceMonitor, int64, error)
	Update(ctx context.Context, service *models.ServiceMonitor) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Probe results
	SaveProbeResult(ctx context.Context, result *models.ServiceProbeResult) error
	GetProbeHistory(ctx context.Context, serviceID uuid.UUID, start, end time.Time, limit int) ([]*models.ServiceProbeResult, int64, error)
	GetLatestProbeResult(ctx context.Context, serviceID uuid.UUID) (*models.ServiceProbeResult, error)

	// Availability statistics
	SaveAvailability(ctx context.Context, availability *models.ServiceAvailability) error
	GetAvailability(ctx context.Context, serviceID uuid.UUID, period string, start time.Time) (*models.ServiceAvailability, error)
	CalculateAvailability(ctx context.Context, serviceID uuid.UUID, start, end time.Time) (*models.ServiceAvailability, error)
}

// serviceRepository implements ServiceRepository
type serviceRepository struct {
	db *gorm.DB
}

// NewServiceRepository creates a new service repository
func NewServiceRepository(db *gorm.DB) ServiceRepository {
	return &serviceRepository{db: db}
}

// Create creates a new service monitor
func (r *serviceRepository) Create(ctx context.Context, service *models.ServiceMonitor) error {
	return r.db.WithContext(ctx).Create(service).Error
}

// GetByID retrieves a service monitor by ID
func (r *serviceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ServiceMonitor, error) {
	var service models.ServiceMonitor
	err := r.db.WithContext(ctx).First(&service, id).Error
	if err != nil {
		return nil, err
	}
	return &service, nil
}

// List retrieves a list of service monitors with filtering
func (r *serviceRepository) List(ctx context.Context, filter ServiceFilter) ([]*models.ServiceMonitor, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.ServiceMonitor{})

	// Apply filters
	if filter.HostID > 0 {
		query = query.Where("host_node_id = ?", filter.HostID)
	}

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	if filter.Enabled != nil {
		query = query.Where("enabled = ?", *filter.Enabled)
	}

	if filter.Search != "" {
		query = query.Where("name LIKE ? OR target LIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	// Fetch results
	var services []*models.ServiceMonitor
	if err := query.Order("created_at DESC").Find(&services).Error; err != nil {
		return nil, 0, err
	}

	return services, total, nil
}

// Update updates a service monitor
func (r *serviceRepository) Update(ctx context.Context, service *models.ServiceMonitor) error {
	return r.db.WithContext(ctx).Save(service).Error
}

// Delete deletes a service monitor and its associated data
func (r *serviceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete associated probe results
		if err := tx.Where("service_monitor_id = ?", id).Delete(&models.ServiceProbeResult{}).Error; err != nil {
			return err
		}

		// Delete availability statistics
		if err := tx.Where("service_monitor_id = ?", id).Delete(&models.ServiceAvailability{}).Error; err != nil {
			return err
		}

		// Delete the service monitor
		return tx.Delete(&models.ServiceMonitor{}, id).Error
	})
}

// SaveProbeResult saves a probe result
func (r *serviceRepository) SaveProbeResult(ctx context.Context, result *models.ServiceProbeResult) error {
	return r.db.WithContext(ctx).Create(result).Error
}

// GetProbeHistory retrieves probe history within a time range
func (r *serviceRepository) GetProbeHistory(ctx context.Context, serviceID uuid.UUID, start, end time.Time, limit int) ([]*models.ServiceProbeResult, int64, error) {
	query := r.db.WithContext(ctx).
		Where("service_monitor_id = ?", serviceID)

	if !start.IsZero() && !end.IsZero() {
		query = query.Where("timestamp >= ? AND timestamp <= ?", start, end)
	}

	// Count total
	var total int64
	if err := query.Model(&models.ServiceProbeResult{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply limit
	if limit <= 0 {
		limit = 100
	}
	query = query.Order("timestamp DESC").Limit(limit)

	var results []*models.ServiceProbeResult
	if err := query.Find(&results).Error; err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// GetLatestProbeResult retrieves the most recent probe result
func (r *serviceRepository) GetLatestProbeResult(ctx context.Context, serviceID uuid.UUID) (*models.ServiceProbeResult, error) {
	var result models.ServiceProbeResult
	err := r.db.WithContext(ctx).
		Where("service_monitor_id = ?", serviceID).
		Order("timestamp DESC").
		First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SaveAvailability saves availability statistics
func (r *serviceRepository) SaveAvailability(ctx context.Context, availability *models.ServiceAvailability) error {
	// Upsert based on service_id + period + start_time
	return r.db.WithContext(ctx).
		Where("service_monitor_id = ? AND period = ? AND start_time = ?",
			availability.ServiceMonitorID, availability.Period, availability.StartTime).
		Assign(availability).
		FirstOrCreate(availability).Error
}

// GetAvailability retrieves availability statistics for a specific period
func (r *serviceRepository) GetAvailability(ctx context.Context, serviceID uuid.UUID, period string, start time.Time) (*models.ServiceAvailability, error) {
	var availability models.ServiceAvailability
	err := r.db.WithContext(ctx).
		Where("service_monitor_id = ? AND period = ? AND start_time = ?", serviceID, period, start).
		First(&availability).Error
	if err != nil {
		return nil, err
	}
	return &availability, nil
}

// CalculateAvailability calculates availability from probe results
func (r *serviceRepository) CalculateAvailability(ctx context.Context, serviceID uuid.UUID, start, end time.Time) (*models.ServiceAvailability, error) {
	var totalChecks int64
	var successfulChecks int64

	// Count total checks
	if err := r.db.WithContext(ctx).
		Model(&models.ServiceProbeResult{}).
		Where("service_monitor_id = ? AND timestamp >= ? AND timestamp <= ?", serviceID, start, end).
		Count(&totalChecks).Error; err != nil {
		return nil, err
	}

	// Count successful checks
	if err := r.db.WithContext(ctx).
		Model(&models.ServiceProbeResult{}).
		Where("service_monitor_id = ? AND timestamp >= ? AND timestamp <= ? AND success = ?", serviceID, start, end, true).
		Count(&successfulChecks).Error; err != nil {
		return nil, err
	}

	// Calculate average latency
	var avgLatency float64
	var minLatency, maxLatency int

	r.db.WithContext(ctx).
		Model(&models.ServiceProbeResult{}).
		Where("service_monitor_id = ? AND timestamp >= ? AND timestamp <= ? AND success = ?", serviceID, start, end, true).
		Select("AVG(latency) as avg, MIN(latency) as min, MAX(latency) as max").
		Row().Scan(&avgLatency, &minLatency, &maxLatency)

	availability := &models.ServiceAvailability{
		ServiceMonitorID: serviceID,
		StartTime:        start,
		EndTime:          end,
		TotalChecks:      int(totalChecks),
		SuccessfulChecks: int(successfulChecks),
		FailedChecks:     int(totalChecks - successfulChecks),
		AvgLatency:       avgLatency,
		MinLatency:       minLatency,
		MaxLatency:       maxLatency,
	}

	availability.CalculateUptime()

	return availability, nil
}
