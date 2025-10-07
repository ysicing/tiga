package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// QueryCache manages query result caching
type QueryCache struct {
	manager *CacheManager
}

// NewQueryCache creates a new query cache
func NewQueryCache(manager *CacheManager) *QueryCache {
	return &QueryCache{
		manager: manager,
	}
}

// QueryResult represents a cached query result
type QueryResult struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Count   int                      `json:"count"`
}

// GetQueryResult retrieves a cached query result
func (qc *QueryCache) GetQueryResult(ctx context.Context, query string, params ...interface{}) (*QueryResult, error) {
	if !qc.manager.IsEnabled() {
		return nil, ErrCacheMiss
	}

	key := qc.buildQueryKey(query, params...)
	var result QueryResult

	if err := qc.manager.client.GetJSON(ctx, key, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SetQueryResult stores a query result in cache
func (qc *QueryCache) SetQueryResult(ctx context.Context, query string, result *QueryResult, params ...interface{}) error {
	if !qc.manager.IsEnabled() {
		return nil
	}

	key := qc.buildQueryKey(query, params...)
	return qc.manager.client.SetJSON(ctx, key, result, qc.manager.config.QueryResultTTL)
}

// InvalidateQuery invalidates a specific query cache
func (qc *QueryCache) InvalidateQuery(ctx context.Context, query string, params ...interface{}) error {
	if !qc.manager.IsEnabled() {
		return nil
	}

	key := qc.buildQueryKey(query, params...)
	return qc.manager.client.Delete(ctx, key)
}

// InvalidateTableQueries invalidates all queries related to a table
func (qc *QueryCache) InvalidateTableQueries(ctx context.Context, tableName string) error {
	if !qc.manager.IsEnabled() {
		return nil
	}

	pattern := qc.manager.CacheKey(qc.manager.config.QueryResultPrefix, fmt.Sprintf("table:%s:*", tableName))
	return qc.manager.client.DeletePattern(ctx, pattern)
}

// InvalidateAll invalidates all query caches
func (qc *QueryCache) InvalidateAll(ctx context.Context) error {
	if !qc.manager.IsEnabled() {
		return nil
	}

	pattern := qc.manager.CacheKey(qc.manager.config.QueryResultPrefix, "*")
	return qc.manager.client.DeletePattern(ctx, pattern)
}

// buildQueryKey builds a cache key for a query
func (qc *QueryCache) buildQueryKey(query string, params ...interface{}) string {
	// Create a hash of the query and parameters
	hasher := sha256.New()
	hasher.Write([]byte(query))

	for _, param := range params {
		hasher.Write([]byte(fmt.Sprintf("%v", param)))
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	return qc.manager.CacheKey(qc.manager.config.QueryResultPrefix, hash[:16])
}

// GetList retrieves a generic list from cache
func (qc *QueryCache) GetList(ctx context.Context, listKey string, dest interface{}) error {
	if !qc.manager.IsEnabled() {
		return ErrCacheMiss
	}

	key := qc.manager.CacheKey(qc.manager.config.ListPrefix, listKey)
	return qc.manager.client.GetJSON(ctx, key, dest)
}

// SetList stores a generic list in cache
func (qc *QueryCache) SetList(ctx context.Context, listKey string, value interface{}) error {
	if !qc.manager.IsEnabled() {
		return nil
	}

	key := qc.manager.CacheKey(qc.manager.config.ListPrefix, listKey)
	return qc.manager.client.SetJSON(ctx, key, value, qc.manager.config.ListTTL)
}

// InvalidateList invalidates a specific list cache
func (qc *QueryCache) InvalidateList(ctx context.Context, listKey string) error {
	if !qc.manager.IsEnabled() {
		return nil
	}

	key := qc.manager.CacheKey(qc.manager.config.ListPrefix, listKey)
	return qc.manager.client.Delete(ctx, key)
}

// LockCache manages distributed locks using Redis
type LockCache struct {
	manager *CacheManager
}

// NewLockCache creates a new lock cache
func NewLockCache(manager *CacheManager) *LockCache {
	return &LockCache{
		manager: manager,
	}
}

// AcquireLock attempts to acquire a distributed lock
func (lc *LockCache) AcquireLock(ctx context.Context, lockKey string, ttl ...interface{}) (bool, error) {
	if !lc.manager.IsEnabled() {
		return false, ErrCacheDisabled
	}

	key := lc.buildLockKey(lockKey)

	// Default TTL is 30 seconds
	lockTTL := lc.manager.config.DefaultTTL
	if len(ttl) > 0 {
		if duration, ok := ttl[0].(int); ok {
			lockTTL = time.Duration(duration) * lc.manager.config.DefaultTTL
		}
	}

	// Use SetNX to acquire lock
	acquired, err := lc.manager.client.SetNX(ctx, key, "1", lockTTL)
	if err != nil {
		return false, err
	}

	return acquired, nil
}

// ReleaseLock releases a distributed lock
func (lc *LockCache) ReleaseLock(ctx context.Context, lockKey string) error {
	if !lc.manager.IsEnabled() {
		return nil
	}

	key := lc.buildLockKey(lockKey)
	return lc.manager.client.Delete(ctx, key)
}

// IsLocked checks if a lock is currently held
func (lc *LockCache) IsLocked(ctx context.Context, lockKey string) (bool, error) {
	if !lc.manager.IsEnabled() {
		return false, ErrCacheDisabled
	}

	key := lc.buildLockKey(lockKey)
	return lc.manager.client.Exists(ctx, key)
}

// ExtendLock extends a lock's expiration
func (lc *LockCache) ExtendLock(ctx context.Context, lockKey string, ttl ...interface{}) error {
	if !lc.manager.IsEnabled() {
		return ErrCacheDisabled
	}

	key := lc.buildLockKey(lockKey)

	lockTTL := lc.manager.config.DefaultTTL
	if len(ttl) > 0 {
		if duration, ok := ttl[0].(int); ok {
			lockTTL = time.Duration(duration) * lc.manager.config.DefaultTTL
		}
	}

	return lc.manager.client.Expire(ctx, key, lockTTL)
}

// buildLockKey builds a cache key for a lock
func (lc *LockCache) buildLockKey(lockKey string) string {
	return lc.manager.CacheKey("lock", lockKey)
}

// WithLock executes a function with a distributed lock
func (lc *LockCache) WithLock(ctx context.Context, lockKey string, fn func() error) error {
	if !lc.manager.IsEnabled() {
		// If cache is disabled, just execute the function
		return fn()
	}

	// Try to acquire lock
	acquired, err := lc.AcquireLock(ctx, lockKey)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		return fmt.Errorf("failed to acquire lock: already locked")
	}

	// Ensure lock is released
	defer lc.ReleaseLock(ctx, lockKey)

	// Execute function
	return fn()
}
