package cache

import (
	"context"
	"fmt"

	"github.com/ysicing/tiga/internal/repository"
)

// StatsCache manages statistics data caching
type StatsCache struct {
	manager *CacheManager
}

// NewStatsCache creates a new stats cache
func NewStatsCache(manager *CacheManager) *StatsCache {
	return &StatsCache{
		manager: manager,
	}
}

// GetInstanceStats retrieves instance statistics from cache
func (sc *StatsCache) GetInstanceStats(ctx context.Context) (*repository.InstanceStatistics, error) {
	if !sc.manager.IsEnabled() {
		return nil, ErrCacheMiss
	}

	key := sc.buildStatsKey("instance")
	var stats repository.InstanceStatistics

	if err := sc.manager.client.GetJSON(ctx, key, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// SetInstanceStats stores instance statistics in cache
func (sc *StatsCache) SetInstanceStats(ctx context.Context, stats *repository.InstanceStatistics) error {
	if !sc.manager.IsEnabled() {
		return nil
	}

	key := sc.buildStatsKey("instance")
	return sc.manager.client.SetJSON(ctx, key, stats, sc.manager.config.StatsTTL)
}

// GetAlertStats retrieves alert statistics from cache
func (sc *StatsCache) GetAlertStats(ctx context.Context) (*repository.AlertStatistics, error) {
	if !sc.manager.IsEnabled() {
		return nil, ErrCacheMiss
	}

	key := sc.buildStatsKey("alert")
	var stats repository.AlertStatistics

	if err := sc.manager.client.GetJSON(ctx, key, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// SetAlertStats stores alert statistics in cache
func (sc *StatsCache) SetAlertStats(ctx context.Context, stats *repository.AlertStatistics) error {
	if !sc.manager.IsEnabled() {
		return nil
	}

	key := sc.buildStatsKey("alert")
	return sc.manager.client.SetJSON(ctx, key, stats, sc.manager.config.StatsTTL)
}

// GetAuditStats retrieves audit log statistics from cache
func (sc *StatsCache) GetAuditStats(ctx context.Context) (*repository.AuditLogStatistics, error) {
	if !sc.manager.IsEnabled() {
		return nil, ErrCacheMiss
	}

	key := sc.buildStatsKey("audit")
	var stats repository.AuditLogStatistics

	if err := sc.manager.client.GetJSON(ctx, key, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// SetAuditStats stores audit log statistics in cache
func (sc *StatsCache) SetAuditStats(ctx context.Context, stats *repository.AuditLogStatistics) error {
	if !sc.manager.IsEnabled() {
		return nil
	}

	key := sc.buildStatsKey("audit")
	return sc.manager.client.SetJSON(ctx, key, stats, sc.manager.config.StatsTTL)
}

// InvalidateInstanceStats invalidates instance statistics cache
func (sc *StatsCache) InvalidateInstanceStats(ctx context.Context) error {
	if !sc.manager.IsEnabled() {
		return nil
	}

	key := sc.buildStatsKey("instance")
	return sc.manager.client.Delete(ctx, key)
}

// InvalidateAlertStats invalidates alert statistics cache
func (sc *StatsCache) InvalidateAlertStats(ctx context.Context) error {
	if !sc.manager.IsEnabled() {
		return nil
	}

	key := sc.buildStatsKey("alert")
	return sc.manager.client.Delete(ctx, key)
}

// InvalidateAuditStats invalidates audit statistics cache
func (sc *StatsCache) InvalidateAuditStats(ctx context.Context) error {
	if !sc.manager.IsEnabled() {
		return nil
	}

	key := sc.buildStatsKey("audit")
	return sc.manager.client.Delete(ctx, key)
}

// InvalidateAll invalidates all statistics caches
func (sc *StatsCache) InvalidateAll(ctx context.Context) error {
	if !sc.manager.IsEnabled() {
		return nil
	}

	pattern := sc.manager.CacheKey(sc.manager.config.StatsPrefix, "*")
	return sc.manager.client.DeletePattern(ctx, pattern)
}

// buildStatsKey builds a cache key for statistics
func (sc *StatsCache) buildStatsKey(statsType string) string {
	return sc.manager.CacheKey(sc.manager.config.StatsPrefix, statsType)
}

// GetCustomStats retrieves custom statistics from cache
func (sc *StatsCache) GetCustomStats(ctx context.Context, key string, dest interface{}) error {
	if !sc.manager.IsEnabled() {
		return ErrCacheMiss
	}

	cacheKey := sc.buildStatsKey(key)
	return sc.manager.client.GetJSON(ctx, cacheKey, dest)
}

// SetCustomStats stores custom statistics in cache
func (sc *StatsCache) SetCustomStats(ctx context.Context, key string, value interface{}) error {
	if !sc.manager.IsEnabled() {
		return nil
	}

	cacheKey := sc.buildStatsKey(key)
	return sc.manager.client.SetJSON(ctx, cacheKey, value, sc.manager.config.StatsTTL)
}

// IncrementCounter increments a statistics counter
func (sc *StatsCache) IncrementCounter(ctx context.Context, counterName string) (int64, error) {
	if !sc.manager.IsEnabled() {
		return 0, ErrCacheDisabled
	}

	key := sc.buildStatsKey(fmt.Sprintf("counter:%s", counterName))
	return sc.manager.client.Increment(ctx, key)
}

// DecrementCounter decrements a statistics counter
func (sc *StatsCache) DecrementCounter(ctx context.Context, counterName string) (int64, error) {
	if !sc.manager.IsEnabled() {
		return 0, ErrCacheDisabled
	}

	key := sc.buildStatsKey(fmt.Sprintf("counter:%s", counterName))
	return sc.manager.client.Decrement(ctx, key)
}

// GetCounter retrieves a counter value
func (sc *StatsCache) GetCounter(ctx context.Context, counterName string) (int64, error) {
	if !sc.manager.IsEnabled() {
		return 0, ErrCacheMiss
	}

	key := sc.buildStatsKey(fmt.Sprintf("counter:%s", counterName))
	val, err := sc.manager.client.Get(ctx, key)
	if err != nil {
		return 0, err
	}

	var count int64
	if _, err := fmt.Sscanf(val, "%d", &count); err != nil {
		return 0, fmt.Errorf("failed to parse counter: %w", err)
	}

	return count, nil
}
