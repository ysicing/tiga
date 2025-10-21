package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// InstanceRepository handles instance data operations
type InstanceRepository struct {
	db *gorm.DB
}

// NewInstanceRepository creates a new InstanceRepository
func NewInstanceRepository(db *gorm.DB) *InstanceRepository {
	return &InstanceRepository{db: db}
}

// Create creates a new instance
func (r *InstanceRepository) Create(ctx context.Context, instance *models.Instance) error {
	if err := r.db.WithContext(ctx).Create(instance).Error; err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}
	return nil
}

// GetByID retrieves an instance by ID
func (r *InstanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Instance, error) {
	var instance models.Instance
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&instance).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("instance not found")
		}
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	return &instance, nil
}

// GetByName retrieves an instance by name
func (r *InstanceRepository) GetByName(ctx context.Context, name string) (*models.Instance, error) {
	var instance models.Instance
	err := r.db.WithContext(ctx).
		Where("name = ?", name).
		First(&instance).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("instance not found")
		}
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	return &instance, nil
}

// Update updates an instance
func (r *InstanceRepository) Update(ctx context.Context, instance *models.Instance) error {
	if err := r.db.WithContext(ctx).Save(instance).Error; err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}
	return nil
}

// UpdateFields updates specific fields of an instance
func (r *InstanceRepository) UpdateFields(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&models.Instance{}).
		Where("id = ?", id).
		Updates(fields)

	if result.Error != nil {
		return fmt.Errorf("failed to update instance fields: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("instance not found")
	}

	return nil
}

// Delete soft deletes an instance
func (r *InstanceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Instance{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete instance: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("instance not found")
	}

	return nil
}

// HardDelete permanently deletes an instance (including soft-deleted ones)
func (r *InstanceRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Unscoped().Delete(&models.Instance{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to hard delete instance: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("instance not found")
	}

	return nil
}

// ListInstancesFilter represents instance list filters
type ListInstancesFilter struct {
	ServiceType string   // Filter by service type
	Status      string   // Filter by status
	Environment string   // Filter by environment
	Tags        []string // Filter by tags (AND logic)
	Search      string   // Search in name, host, description
	Page        int      // Page number (1-based)
	PageSize    int      // Page size
}

// ListInstances retrieves a paginated list of instances with filters
func (r *InstanceRepository) ListInstances(ctx context.Context, filter *ListInstancesFilter) ([]*models.Instance, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Instance{})

	// Apply filters
	if filter.ServiceType != "" {
		query = query.Where("type = ?", filter.ServiceType)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.Environment != "" {
		query = query.Where("environment = ?", filter.Environment)
	}

	if len(filter.Tags) > 0 {
		// Tags filter with AND logic: instance must have all specified tags
		for _, tag := range filter.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}

			// Build JSON array snippet for parameter binding
			jsonArray := fmt.Sprintf(`["%s"]`, strings.ReplaceAll(tag, `"`, `\"`))

			switch r.db.Dialector.Name() {
			case "postgres":
				query = query.Where("(tags)::jsonb @> ?::jsonb", jsonArray)
			case "mysql":
				query = query.Where("JSON_CONTAINS(CAST(tags AS JSON), CAST(? AS JSON))", jsonArray)
			default:
				// Fallback to LIKE matching for other dialects (e.g. sqlite)
				query = query.Where("tags LIKE ?", fmt.Sprintf(`%%"%s"%%`, tag))
			}
		}
	}

	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("name LIKE ? OR display_name LIKE ? OR description LIKE ?", search, search, search)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count instances: %w", err)
	}

	// Pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Fetch results
	var instances []*models.Instance
	if err := query.Order("created_at DESC").Find(&instances).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list instances: %w", err)
	}

	return instances, total, nil
}

// ListByServiceType retrieves all instances of a specific service type
func (r *InstanceRepository) ListByServiceType(ctx context.Context, serviceType string) ([]*models.Instance, error) {
	var instances []*models.Instance
	err := r.db.WithContext(ctx).
		Where("type = ?", serviceType).
		Order("name ASC").
		Find(&instances).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list instances by service type: %w", err)
	}

	return instances, nil
}

// ListByStatus retrieves all instances with a specific status
func (r *InstanceRepository) ListByStatus(ctx context.Context, status string) ([]*models.Instance, error) {
	var instances []*models.Instance
	err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at DESC").
		Find(&instances).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list instances by status: %w", err)
	}

	return instances, nil
}

// CountByServiceType counts instances by service type
func (r *InstanceRepository) CountByServiceType(ctx context.Context, serviceType string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Instance{}).
		Where("type = ?", serviceType).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count instances by service type: %w", err)
	}

	return count, nil
}

// CountByStatus counts instances by status
func (r *InstanceRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Instance{}).
		Where("status = ?", status).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count instances by status: %w", err)
	}

	return count, nil
}

// ExistsName checks if an instance name exists
func (r *InstanceRepository) ExistsName(ctx context.Context, name string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).Model(&models.Instance{}).Where("name = ?", name)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check instance name existence: %w", err)
	}

	return count > 0, nil
}

// UpdateStatus updates instance status
func (r *InstanceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	result := r.db.WithContext(ctx).
		Model(&models.Instance{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		return fmt.Errorf("failed to update instance status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("instance not found")
	}

	return nil
}

// UpdateHealth updates instance health status
func (r *InstanceRepository) UpdateHealth(ctx context.Context, id uuid.UUID, healthStatus string, healthMessage *string) error {
	updates := map[string]interface{}{
		"health": healthStatus,
	}

	if healthMessage != nil {
		updates["health_message"] = *healthMessage
	}

	result := r.db.WithContext(ctx).
		Model(&models.Instance{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update instance health: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("instance not found")
	}

	return nil
}

// UpdateVersion updates instance version
func (r *InstanceRepository) UpdateVersion(ctx context.Context, id uuid.UUID, version string) error {
	result := r.db.WithContext(ctx).
		Model(&models.Instance{}).
		Where("id = ?", id).
		Update("version", version)

	if result.Error != nil {
		return fmt.Errorf("failed to update instance version: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("instance not found")
	}

	return nil
}

// AddTags adds tags to an instance
func (r *InstanceRepository) AddTags(ctx context.Context, id uuid.UUID, tags []string) error {
	instance, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Merge tags (avoid duplicates)
	tagMap := make(map[string]bool)
	for _, tag := range instance.Tags {
		tagMap[tag] = true
	}
	for _, tag := range tags {
		tagMap[tag] = true
	}

	newTags := make(models.StringArray, 0, len(tagMap))
	for tag := range tagMap {
		newTags = append(newTags, tag)
	}

	// Use Update instead of UpdateFields to leverage GORM's Value() method
	instance.Tags = newTags
	return r.Update(ctx, instance)
}

// RemoveTags removes tags from an instance
func (r *InstanceRepository) RemoveTags(ctx context.Context, id uuid.UUID, tags []string) error {
	instance, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Remove specified tags
	removeMap := make(map[string]bool)
	for _, tag := range tags {
		removeMap[tag] = true
	}

	newTags := make(models.StringArray, 0)
	for _, tag := range instance.Tags {
		if !removeMap[tag] {
			newTags = append(newTags, tag)
		}
	}

	// Use Update instead of UpdateFields to leverage GORM's Value() method
	instance.Tags = newTags
	return r.Update(ctx, instance)
}

// SearchByTag searches instances by tag (OR logic)
func (r *InstanceRepository) SearchByTag(ctx context.Context, tags []string) ([]*models.Instance, error) {
	if len(tags) == 0 {
		return []*models.Instance{}, nil
	}

	var instances []*models.Instance

	// Build OR condition for tags
	conditions := make([]string, len(tags))
	args := make([]interface{}, len(tags))
	for i, tag := range tags {
		conditions[i] = "? = ANY(tags)"
		args[i] = tag
	}

	whereClause := strings.Join(conditions, " OR ")

	err := r.db.WithContext(ctx).
		Where(whereClause, args...).
		Order("name ASC").
		Find(&instances).Error

	if err != nil {
		return nil, fmt.Errorf("failed to search instances by tag: %w", err)
	}

	return instances, nil
}

// GetStatistics retrieves instance statistics
type InstanceStatistics struct {
	TotalInstances     int64            `json:"total_instances"`
	ByServiceType      map[string]int64 `json:"by_service_type"`
	ByStatus           map[string]int64 `json:"by_status"`
	ByEnvironment      map[string]int64 `json:"by_environment"`
	HealthyInstances   int64            `json:"healthy_instances"`
	UnhealthyInstances int64            `json:"unhealthy_instances"`
}

func (r *InstanceRepository) GetStatistics(ctx context.Context) (*InstanceStatistics, error) {
	stats := &InstanceStatistics{
		ByServiceType: make(map[string]int64),
		ByStatus:      make(map[string]int64),
		ByEnvironment: make(map[string]int64),
	}

	// Total count
	if err := r.db.WithContext(ctx).Model(&models.Instance{}).Count(&stats.TotalInstances).Error; err != nil {
		return nil, fmt.Errorf("failed to count total instances: %w", err)
	}

	// Group by service type
	type ServiceTypeCount struct {
		Type  string
		Count int64
	}
	var serviceTypeCounts []ServiceTypeCount
	if err := r.db.WithContext(ctx).
		Model(&models.Instance{}).
		Select("type, COUNT(*) as count").
		Group("type").
		Scan(&serviceTypeCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to count by service type: %w", err)
	}
	for _, stc := range serviceTypeCounts {
		stats.ByServiceType[stc.Type] = stc.Count
	}

	// Group by status
	type StatusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []StatusCount
	if err := r.db.WithContext(ctx).
		Model(&models.Instance{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to count by status: %w", err)
	}
	for _, sc := range statusCounts {
		stats.ByStatus[sc.Status] = sc.Count
	}

	// Group by environment
	type EnvironmentCount struct {
		Environment string
		Count       int64
	}
	var environmentCounts []EnvironmentCount
	if err := r.db.WithContext(ctx).
		Model(&models.Instance{}).
		Select("environment, COUNT(*) as count").
		Group("environment").
		Scan(&environmentCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to count by environment: %w", err)
	}
	for _, ec := range environmentCounts {
		stats.ByEnvironment[ec.Environment] = ec.Count
	}

	// Healthy vs Unhealthy
	if err := r.db.WithContext(ctx).
		Model(&models.Instance{}).
		Where("health = ?", "healthy").
		Count(&stats.HealthyInstances).Error; err != nil {
		return nil, fmt.Errorf("failed to count healthy instances: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Model(&models.Instance{}).
		Where("health = ?", "unhealthy").
		Count(&stats.UnhealthyInstances).Error; err != nil {
		return nil, fmt.Errorf("failed to count unhealthy instances: %w", err)
	}

	return stats, nil
}
