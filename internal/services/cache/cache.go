package cache

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCacheMiss is returned when a key is not found in cache
	ErrCacheMiss = errors.New("cache miss")

	// ErrCacheDisabled is returned when cache operations are attempted but cache is disabled
	ErrCacheDisabled = errors.New("cache is disabled")
)

// Cache defines the interface for cache operations
type Cache interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) (string, error)

	// Set stores a value in cache with expiration
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	// Delete removes keys from cache
	Delete(ctx context.Context, keys ...string) error

	// Exists checks if a key exists
	Exists(ctx context.Context, key string) (bool, error)

	// GetJSON retrieves and unmarshals a JSON value
	GetJSON(ctx context.Context, key string, dest interface{}) error

	// SetJSON marshals and stores a JSON value
	SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	// DeletePattern deletes all keys matching a pattern
	DeletePattern(ctx context.Context, pattern string) error

	// IsEnabled returns whether cache is enabled
	IsEnabled() bool
}

// CacheManager manages different cache namespaces
type CacheManager struct {
	client *RedisClient
	config *CacheConfig
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	// Default TTLs for different data types
	DefaultTTL     time.Duration
	InstanceTTL    time.Duration
	StatsTTL       time.Duration
	SessionTTL     time.Duration
	QueryResultTTL time.Duration
	ListTTL        time.Duration

	// Cache key prefixes
	InstancePrefix    string
	StatsPrefix       string
	SessionPrefix     string
	QueryResultPrefix string
	ListPrefix        string
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		DefaultTTL:     5 * time.Minute,
		InstanceTTL:    10 * time.Minute,
		StatsTTL:       1 * time.Minute,
		SessionTTL:     24 * time.Hour,
		QueryResultTTL: 30 * time.Second,
		ListTTL:        2 * time.Minute,

		InstancePrefix:    "instance",
		StatsPrefix:       "stats",
		SessionPrefix:     "session",
		QueryResultPrefix: "query",
		ListPrefix:        "list",
	}
}

// NewCacheManager creates a new cache manager
func NewCacheManager(client *RedisClient, config *CacheConfig) *CacheManager {
	if config == nil {
		config = DefaultCacheConfig()
	}

	return &CacheManager{
		client: client,
		config: config,
	}
}

// IsEnabled returns whether cache is enabled
func (cm *CacheManager) IsEnabled() bool {
	return cm.client.IsEnabled()
}

// GetClient returns the underlying Redis client
func (cm *CacheManager) GetClient() *RedisClient {
	return cm.client
}

// CacheKey builds a cache key with namespace
func (cm *CacheManager) CacheKey(namespace, key string) string {
	return namespace + ":" + key
}

// InvalidateByPattern invalidates all keys matching a pattern
func (cm *CacheManager) InvalidateByPattern(ctx context.Context, pattern string) error {
	return cm.client.DeletePattern(ctx, pattern)
}

// Clear clears all cache (use with caution!)
func (cm *CacheManager) Clear(ctx context.Context) error {
	return cm.client.FlushDB(ctx)
}

// Warmup pre-warms cache with frequently accessed data
func (cm *CacheManager) Warmup(ctx context.Context, warmupFuncs ...func(context.Context) error) error {
	if !cm.IsEnabled() {
		return nil
	}

	for _, fn := range warmupFuncs {
		if err := fn(ctx); err != nil {
			// Log error but continue with other warmup functions
			continue
		}
	}

	return nil
}
