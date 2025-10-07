package cache

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
)

// InstanceCache manages instance data caching
type InstanceCache struct {
	manager *CacheManager
}

// NewInstanceCache creates a new instance cache
func NewInstanceCache(manager *CacheManager) *InstanceCache {
	return &InstanceCache{
		manager: manager,
	}
}

// GetInstance retrieves an instance from cache
func (ic *InstanceCache) GetInstance(ctx context.Context, id uuid.UUID) (*models.Instance, error) {
	if !ic.manager.IsEnabled() {
		return nil, ErrCacheMiss
	}

	key := ic.buildInstanceKey(id)
	var instance models.Instance

	if err := ic.manager.client.GetJSON(ctx, key, &instance); err != nil {
		return nil, err
	}

	return &instance, nil
}

// SetInstance stores an instance in cache
func (ic *InstanceCache) SetInstance(ctx context.Context, instance *models.Instance) error {
	if !ic.manager.IsEnabled() {
		return nil
	}

	key := ic.buildInstanceKey(instance.ID)
	return ic.manager.client.SetJSON(ctx, key, instance, ic.manager.config.InstanceTTL)
}

// DeleteInstance removes an instance from cache
func (ic *InstanceCache) DeleteInstance(ctx context.Context, id uuid.UUID) error {
	if !ic.manager.IsEnabled() {
		return nil
	}

	key := ic.buildInstanceKey(id)
	return ic.manager.client.Delete(ctx, key)
}

// InvalidateInstance invalidates instance cache
func (ic *InstanceCache) InvalidateInstance(ctx context.Context, id uuid.UUID) error {
	// Delete the instance itself
	if err := ic.DeleteInstance(ctx, id); err != nil {
		return err
	}

	// Also invalidate instance lists
	return ic.InvalidateInstanceLists(ctx)
}

// InvalidateInstanceLists invalidates all instance list caches
func (ic *InstanceCache) InvalidateInstanceLists(ctx context.Context) error {
	if !ic.manager.IsEnabled() {
		return nil
	}

	pattern := ic.manager.CacheKey(ic.manager.config.ListPrefix, "instances:*")
	return ic.manager.client.DeletePattern(ctx, pattern)
}

// GetInstanceList retrieves a cached instance list
func (ic *InstanceCache) GetInstanceList(ctx context.Context, cacheKey string) ([]*models.Instance, error) {
	if !ic.manager.IsEnabled() {
		return nil, ErrCacheMiss
	}

	key := ic.manager.CacheKey(ic.manager.config.ListPrefix, cacheKey)
	var instances []*models.Instance

	if err := ic.manager.client.GetJSON(ctx, key, &instances); err != nil {
		return nil, err
	}

	return instances, nil
}

// SetInstanceList stores an instance list in cache
func (ic *InstanceCache) SetInstanceList(ctx context.Context, cacheKey string, instances []*models.Instance) error {
	if !ic.manager.IsEnabled() {
		return nil
	}

	key := ic.manager.CacheKey(ic.manager.config.ListPrefix, cacheKey)
	return ic.manager.client.SetJSON(ctx, key, instances, ic.manager.config.ListTTL)
}

// buildInstanceKey builds a cache key for an instance
func (ic *InstanceCache) buildInstanceKey(id uuid.UUID) string {
	return ic.manager.CacheKey(ic.manager.config.InstancePrefix, id.String())
}

// GetInstancesByType retrieves instances by type from cache
func (ic *InstanceCache) GetInstancesByType(ctx context.Context, instanceType string) ([]*models.Instance, error) {
	cacheKey := fmt.Sprintf("instances:type:%s", instanceType)
	return ic.GetInstanceList(ctx, cacheKey)
}

// SetInstancesByType stores instances by type in cache
func (ic *InstanceCache) SetInstancesByType(ctx context.Context, instanceType string, instances []*models.Instance) error {
	cacheKey := fmt.Sprintf("instances:type:%s", instanceType)
	return ic.SetInstanceList(ctx, cacheKey, instances)
}

// GetInstancesByStatus retrieves instances by status from cache
func (ic *InstanceCache) GetInstancesByStatus(ctx context.Context, status string) ([]*models.Instance, error) {
	cacheKey := fmt.Sprintf("instances:status:%s", status)
	return ic.GetInstanceList(ctx, cacheKey)
}

// SetInstancesByStatus stores instances by status in cache
func (ic *InstanceCache) SetInstancesByStatus(ctx context.Context, status string, instances []*models.Instance) error {
	cacheKey := fmt.Sprintf("instances:status:%s", status)
	return ic.SetInstanceList(ctx, cacheKey, instances)
}

// InvalidateAll invalidates all instance-related caches
func (ic *InstanceCache) InvalidateAll(ctx context.Context) error {
	if !ic.manager.IsEnabled() {
		return nil
	}

	// Invalidate all instance keys
	pattern1 := ic.manager.CacheKey(ic.manager.config.InstancePrefix, "*")
	if err := ic.manager.client.DeletePattern(ctx, pattern1); err != nil {
		return err
	}

	// Invalidate all instance lists
	return ic.InvalidateInstanceLists(ctx)
}
