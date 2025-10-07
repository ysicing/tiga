package repository

// Query Optimization Improvements for Instance Repository
//
// This file contains optimized versions of methods from instance_repo.go
// to address N+1 query problems and improve query performance.

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
)

// AddTagsOptimized adds tags to an instance without loading the entire instance
// Optimization: Uses raw SQL update to avoid N+1 query (GetByID + UpdateFields)
func (r *InstanceRepository) AddTagsOptimized(ctx context.Context, id uuid.UUID, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	// Use array_cat and array_distinct for efficient tag merging in PostgreSQL
	query := `
		UPDATE instances
		SET tags = array_distinct(array_cat(tags, $1::text[]))
		WHERE id = $2 AND deleted_at IS NULL
	`

	result := r.db.WithContext(ctx).Exec(query, tags, id)
	if result.Error != nil {
		return fmt.Errorf("failed to add tags: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("instance not found")
	}

	return nil
}

// RemoveTagsOptimized removes tags from an instance without loading the entire instance
// Optimization: Uses array operations to remove tags efficiently
func (r *InstanceRepository) RemoveTagsOptimized(ctx context.Context, id uuid.UUID, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	// Use array subtraction for efficient tag removal in PostgreSQL
	query := `
		UPDATE instances
		SET tags = array(SELECT unnest(tags) EXCEPT SELECT unnest($1::text[]))
		WHERE id = $2 AND deleted_at IS NULL
	`

	result := r.db.WithContext(ctx).Exec(query, tags, id)
	if result.Error != nil {
		return fmt.Errorf("failed to remove tags: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("instance not found")
	}

	return nil
}

// GetStatisticsOptimized retrieves instance statistics with a single query
// Optimization: Uses window functions to get all statistics in one query
func (r *InstanceRepository) GetStatisticsOptimized(ctx context.Context) (*InstanceStatistics, error) {
	query := `
		SELECT
			COUNT(*) as total_instances,
			COUNT(*) FILTER (WHERE health = 'healthy') as healthy_instances,
			COUNT(*) FILTER (WHERE health = 'unhealthy') as unhealthy_instances,
			jsonb_object_agg(
				COALESCE(type, 'unknown'),
				type_count
			) FILTER (WHERE type IS NOT NULL) as by_service_type,
			jsonb_object_agg(
				COALESCE(status, 'unknown'),
				status_count
			) FILTER (WHERE status IS NOT NULL) as by_status,
			jsonb_object_agg(
				COALESCE(environment, 'unknown'),
				env_count
			) FILTER (WHERE environment IS NOT NULL) as by_environment
		FROM (
			SELECT
				type,
				status,
				environment,
				health,
				COUNT(*) OVER (PARTITION BY type) as type_count,
				COUNT(*) OVER (PARTITION BY status) as status_count,
				COUNT(*) OVER (PARTITION BY environment) as env_count
			FROM instances
			WHERE deleted_at IS NULL
		) sub
	`

	type statsResult struct {
		TotalInstances     int64
		HealthyInstances   int64
		UnhealthyInstances int64
		ByServiceType      models.JSONB
		ByStatus           models.JSONB
		ByEnvironment      models.JSONB
	}

	var result statsResult
	if err := r.db.WithContext(ctx).Raw(query).Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	stats := &InstanceStatistics{
		TotalInstances:     result.TotalInstances,
		HealthyInstances:   result.HealthyInstances,
		UnhealthyInstances: result.UnhealthyInstances,
		ByServiceType:      make(map[string]int64),
		ByStatus:           make(map[string]int64),
		ByEnvironment:      make(map[string]int64),
	}

	// Convert JSONB to maps (JSONB is already map[string]interface{})
	if result.ByServiceType != nil {
		for k, v := range result.ByServiceType {
			if count, ok := v.(float64); ok {
				stats.ByServiceType[k] = int64(count)
			}
		}
	}

	if result.ByStatus != nil {
		for k, v := range result.ByStatus {
			if count, ok := v.(float64); ok {
				stats.ByStatus[k] = int64(count)
			}
		}
	}

	if result.ByEnvironment != nil {
		for k, v := range result.ByEnvironment {
			if count, ok := v.(float64); ok {
				stats.ByEnvironment[k] = int64(count)
			}
		}
	}

	return stats, nil
}

// ListInstancesWithCache retrieves instances with optional result caching
// Note: Actual caching implementation would use Redis or similar
// This is a placeholder showing where caching should be added
func (r *InstanceRepository) ListInstancesWithCache(ctx context.Context, filter *ListInstancesFilter) ([]*models.Instance, int64, error) {
	// TODO: Check cache here before querying database
	// cacheKey := fmt.Sprintf("instances:list:%v", filter)
	// if cachedResult := cache.Get(cacheKey); cachedResult != nil {
	//     return cachedResult, nil
	// }

	// Use the existing query
	instances, total, err := r.ListInstances(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// TODO: Store in cache with TTL
	// cache.Set(cacheKey, instances, 5*time.Minute)

	return instances, total, nil
}

// UpdateHealthBatch updates health status for multiple instances in a single query
// Optimization: Batch update instead of multiple individual updates
func (r *InstanceRepository) UpdateHealthBatch(ctx context.Context, updates []struct {
	ID            uuid.UUID
	HealthStatus  string
	HealthMessage *string
}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build a temporary table with updates and use UPDATE FROM
	query := `
		UPDATE instances i
		SET
			health = u.health,
			health_message = u.health_message,
			last_health_check = NOW(),
			updated_at = NOW()
		FROM (VALUES
	`

	values := make([]interface{}, 0, len(updates)*3)
	for i, update := range updates {
		if i > 0 {
			query += ","
		}
		query += fmt.Sprintf("($%d::uuid, $%d::varchar(32), $%d::text)", i*3+1, i*3+2, i*3+3)
		values = append(values, update.ID, update.HealthStatus)
		if update.HealthMessage != nil {
			values = append(values, *update.HealthMessage)
		} else {
			values = append(values, nil)
		}
	}

	query += `) AS u(id, health, health_message)
		WHERE i.id = u.id AND i.deleted_at IS NULL
	`

	if err := r.db.WithContext(ctx).Exec(query, values...).Error; err != nil {
		return fmt.Errorf("failed to batch update health: %w", err)
	}

	return nil
}

// GetInstanceIDsByType efficiently retrieves only instance IDs for a service type
// Optimization: SELECT only ID column instead of all columns
func (r *InstanceRepository) GetInstanceIDsByType(ctx context.Context, serviceType string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).
		Model(&models.Instance{}).
		Where("type = ? AND deleted_at IS NULL", serviceType).
		Pluck("id", &ids).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get instance IDs: %w", err)
	}

	return ids, nil
}
