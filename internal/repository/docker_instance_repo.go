package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// DockerInstanceRepository implements DockerInstanceRepositoryInterface
type DockerInstanceRepository struct {
	db *gorm.DB
}

// NewDockerInstanceRepository creates a new DockerInstanceRepository
func NewDockerInstanceRepository(db *gorm.DB) DockerInstanceRepositoryInterface {
	return &DockerInstanceRepository{db: db}
}

// Create creates a new Docker instance
func (r *DockerInstanceRepository) Create(ctx context.Context, instance *models.DockerInstance) error {
	return r.db.WithContext(ctx).Create(instance).Error
}

// GetByID retrieves a Docker instance by ID
func (r *DockerInstanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DockerInstance, error) {
	var instance models.DockerInstance
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// GetByName retrieves a Docker instance by name
func (r *DockerInstanceRepository) GetByName(ctx context.Context, name string) (*models.DockerInstance, error) {
	var instance models.DockerInstance
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// GetByAgentID retrieves a Docker instance by agent ID
func (r *DockerInstanceRepository) GetByAgentID(ctx context.Context, agentID uuid.UUID) (*models.DockerInstance, error) {
	var instance models.DockerInstance
	err := r.db.WithContext(ctx).Where("agent_id = ?", agentID).First(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// Update updates a Docker instance
func (r *DockerInstanceRepository) Update(ctx context.Context, instance *models.DockerInstance) error {
	return r.db.WithContext(ctx).Save(instance).Error
}

// UpdateFields updates specific fields of a Docker instance
func (r *DockerInstanceRepository) UpdateFields(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.DockerInstance{}).Where("id = ?", id).Updates(fields).Error
}

// Delete deletes a Docker instance
func (r *DockerInstanceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.DockerInstance{}, "id = ?", id).Error
}

// ListInstances lists Docker instances with filtering, pagination, and sorting
func (r *DockerInstanceRepository) ListInstances(ctx context.Context, filter *DockerInstanceFilter) ([]*models.DockerInstance, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.DockerInstance{})

	// Apply filters
	if filter != nil {
		if filter.Name != "" {
			query = query.Where("name LIKE ?", "%"+filter.Name+"%")
		}
		if filter.HealthStatus != "" {
			query = query.Where("health_status = ?", filter.HealthStatus)
		}
		if filter.AgentID != nil {
			query = query.Where("agent_id = ?", *filter.AgentID)
		}
		if filter.HostID != nil {
			query = query.Where("host_id = ?", *filter.HostID)
		}
		if len(filter.Tags) > 0 {
			// PostgreSQL: tags @> '["tag1"]'
			// SQLite: JSON search (will use simple LIKE for now)
			for _, tag := range filter.Tags {
				query = query.Where("tags LIKE ?", "%"+tag+"%")
			}
		}
	}

	// Count total before pagination
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	if filter != nil {
		sortBy := "created_at"
		if filter.SortBy != "" {
			sortBy = filter.SortBy
		}
		sortOrder := "desc"
		if filter.SortOrder != "" {
			sortOrder = filter.SortOrder
		}
		query = query.Order(sortBy + " " + sortOrder)
	}

	// Apply pagination
	if filter != nil && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		if offset < 0 {
			offset = 0
		}
		query = query.Limit(filter.PageSize).Offset(offset)
	}

	var instances []*models.DockerInstance
	if err := query.Find(&instances).Error; err != nil {
		return nil, 0, err
	}

	return instances, total, nil
}

// ListByHealthStatus lists Docker instances by health status
func (r *DockerInstanceRepository) ListByHealthStatus(ctx context.Context, status string) ([]*models.DockerInstance, error) {
	var instances []*models.DockerInstance
	err := r.db.WithContext(ctx).Where("health_status = ?", status).Find(&instances).Error
	return instances, err
}

// ListOnlineInstances lists all online Docker instances
func (r *DockerInstanceRepository) ListOnlineInstances(ctx context.Context) ([]*models.DockerInstance, error) {
	return r.ListByHealthStatus(ctx, "online")
}

// SearchByName searches Docker instances by name (fuzzy search)
func (r *DockerInstanceRepository) SearchByName(ctx context.Context, name string) ([]*models.DockerInstance, error) {
	var instances []*models.DockerInstance
	err := r.db.WithContext(ctx).Where("name LIKE ?", "%"+name+"%").Find(&instances).Error
	return instances, err
}

// SearchByTags searches Docker instances by tags
func (r *DockerInstanceRepository) SearchByTags(ctx context.Context, tags []string) ([]*models.DockerInstance, error) {
	if len(tags) == 0 {
		return []*models.DockerInstance{}, nil
	}

	query := r.db.WithContext(ctx).Model(&models.DockerInstance{})
	for _, tag := range tags {
		query = query.Where("tags LIKE ?", "%"+tag+"%")
	}

	var instances []*models.DockerInstance
	err := query.Find(&instances).Error
	return instances, err
}

// UpdateHealthStatus updates health status and resource counts
func (r *DockerInstanceRepository) UpdateHealthStatus(ctx context.Context, id uuid.UUID, status string, containerCount, imageCount, volumeCount, networkCount int) error {
	updates := map[string]interface{}{
		"health_status":     status,
		"last_health_check": time.Now(),
		"container_count":   containerCount,
		"image_count":       imageCount,
		"volume_count":      volumeCount,
		"network_count":     networkCount,
	}
	if status == "online" {
		updates["last_connected_at"] = time.Now()
	}
	return r.db.WithContext(ctx).Model(&models.DockerInstance{}).Where("id = ?", id).Updates(updates).Error
}

// MarkOnline marks a Docker instance as online
func (r *DockerInstanceRepository) MarkOnline(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.DockerInstance{}).Where("id = ?", id).Updates(map[string]interface{}{
		"health_status":     "online",
		"last_connected_at": now,
		"last_health_check": now,
	}).Error
}

// MarkOffline marks a Docker instance as offline
func (r *DockerInstanceRepository) MarkOffline(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.DockerInstance{}).Where("id = ?", id).Updates(map[string]interface{}{
		"health_status":     "offline",
		"last_health_check": time.Now(),
	}).Error
}

// MarkArchived marks a Docker instance as archived
func (r *DockerInstanceRepository) MarkArchived(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.DockerInstance{}).Where("id = ?", id).Update("health_status", "archived").Error
}

// MarkAllInstancesOfflineByAgentID marks all instances associated with an agent as offline
func (r *DockerInstanceRepository) MarkAllInstancesOfflineByAgentID(ctx context.Context, agentID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.DockerInstance{}).
		Where("agent_id = ?", agentID).
		Updates(map[string]interface{}{
			"health_status":     "offline",
			"last_health_check": time.Now(),
		}).Error
}

// UpdateDockerInfo updates Docker daemon information
func (r *DockerInstanceRepository) UpdateDockerInfo(ctx context.Context, id uuid.UUID, info map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.DockerInstance{}).Where("id = ?", id).Updates(info).Error
}

// Count returns the total number of Docker instances
func (r *DockerInstanceRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.DockerInstance{}).Count(&count).Error
	return count, err
}

// CountByHealthStatus returns the count of instances by health status
func (r *DockerInstanceRepository) CountByHealthStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.DockerInstance{}).Where("health_status = ?", status).Count(&count).Error
	return count, err
}

// GetStatistics returns statistics about Docker instances
func (r *DockerInstanceRepository) GetStatistics(ctx context.Context) (*DockerInstanceStatistics, error) {
	stats := &DockerInstanceStatistics{}

	// Get total count
	total, err := r.Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.Total = total

	// Get counts by status
	online, err := r.CountByHealthStatus(ctx, "online")
	if err != nil {
		return nil, err
	}
	stats.Online = online

	offline, err := r.CountByHealthStatus(ctx, "offline")
	if err != nil {
		return nil, err
	}
	stats.Offline = offline

	archived, err := r.CountByHealthStatus(ctx, "archived")
	if err != nil {
		return nil, err
	}
	stats.Archived = archived

	unknown, err := r.CountByHealthStatus(ctx, "unknown")
	if err != nil {
		return nil, err
	}
	stats.Unknown = unknown

	return stats, nil
}
