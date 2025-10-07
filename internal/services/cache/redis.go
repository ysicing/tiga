package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the Redis client with additional functionality
type RedisClient struct {
	client  *redis.Client
	prefix  string
	enabled bool
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	Prefix   string
	Enabled  bool
}

// NewRedisClient creates a new Redis client
func NewRedisClient(config *RedisConfig) (*RedisClient, error) {
	if !config.Enabled {
		return &RedisClient{
			enabled: false,
		}, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client:  client,
		prefix:  config.Prefix,
		enabled: true,
	}, nil
}

// IsEnabled returns whether Redis is enabled
func (r *RedisClient) IsEnabled() bool {
	return r.enabled
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// buildKey builds a cache key with prefix
func (r *RedisClient) buildKey(key string) string {
	if r.prefix != "" {
		return fmt.Sprintf("%s:%s", r.prefix, key)
	}
	return key
}

// Get retrieves a value from cache
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	if !r.enabled {
		return "", fmt.Errorf("cache is disabled")
	}

	val, err := r.client.Get(ctx, r.buildKey(key)).Result()
	if err == redis.Nil {
		return "", ErrCacheMiss
	}
	if err != nil {
		return "", fmt.Errorf("failed to get from cache: %w", err)
	}

	return val, nil
}

// Set stores a value in cache with expiration
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !r.enabled {
		return nil
	}

	// Convert value to string
	var strValue string
	switch v := value.(type) {
	case string:
		strValue = v
	case []byte:
		strValue = string(v)
	default:
		// JSON encode for complex types
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		strValue = string(data)
	}

	if err := r.client.Set(ctx, r.buildKey(key), strValue, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Delete removes a key from cache
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	if !r.enabled {
		return nil
	}

	if len(keys) == 0 {
		return nil
	}

	// Build keys with prefix
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = r.buildKey(key)
	}

	if err := r.client.Del(ctx, prefixedKeys...).Err(); err != nil {
		return fmt.Errorf("failed to delete from cache: %w", err)
	}

	return nil
}

// Exists checks if a key exists in cache
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	if !r.enabled {
		return false, nil
	}

	count, err := r.client.Exists(ctx, r.buildKey(key)).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return count > 0, nil
}

// Expire sets expiration on a key
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if !r.enabled {
		return nil
	}

	if err := r.client.Expire(ctx, r.buildKey(key), expiration).Err(); err != nil {
		return fmt.Errorf("failed to set expiration: %w", err)
	}

	return nil
}

// Increment increments a counter
func (r *RedisClient) Increment(ctx context.Context, key string) (int64, error) {
	if !r.enabled {
		return 0, fmt.Errorf("cache is disabled")
	}

	val, err := r.client.Incr(ctx, r.buildKey(key)).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment: %w", err)
	}

	return val, nil
}

// Decrement decrements a counter
func (r *RedisClient) Decrement(ctx context.Context, key string) (int64, error) {
	if !r.enabled {
		return 0, fmt.Errorf("cache is disabled")
	}

	val, err := r.client.Decr(ctx, r.buildKey(key)).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to decrement: %w", err)
	}

	return val, nil
}

// SetNX sets a value only if it doesn't exist (used for locks)
func (r *RedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	if !r.enabled {
		return false, fmt.Errorf("cache is disabled")
	}

	// Convert value to string
	var strValue string
	switch v := value.(type) {
	case string:
		strValue = v
	case []byte:
		strValue = string(v)
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return false, fmt.Errorf("failed to marshal value: %w", err)
		}
		strValue = string(data)
	}

	ok, err := r.client.SetNX(ctx, r.buildKey(key), strValue, expiration).Result()
	if err != nil {
		return false, fmt.Errorf("failed to setnx: %w", err)
	}

	return ok, nil
}

// GetJSON retrieves and unmarshals a JSON value
func (r *RedisClient) GetJSON(ctx context.Context, key string, dest interface{}) error {
	if !r.enabled {
		return ErrCacheMiss
	}

	val, err := r.Get(ctx, key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// SetJSON marshals and stores a JSON value
func (r *RedisClient) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !r.enabled {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return r.Set(ctx, key, data, expiration)
}

// DeletePattern deletes all keys matching a pattern
func (r *RedisClient) DeletePattern(ctx context.Context, pattern string) error {
	if !r.enabled {
		return nil
	}

	prefixedPattern := r.buildKey(pattern)

	iter := r.client.Scan(ctx, 0, prefixedPattern, 0).Iterator()
	var keys []string

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys: %w", err)
	}

	if len(keys) > 0 {
		// Delete without prefix (keys already have prefix from scan)
		if err := r.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete keys: %w", err)
		}
	}

	return nil
}

// GetTTL gets remaining time to live for a key
func (r *RedisClient) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	if !r.enabled {
		return 0, fmt.Errorf("cache is disabled")
	}

	ttl, err := r.client.TTL(ctx, r.buildKey(key)).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}

	return ttl, nil
}

// FlushDB flushes the current database (use with caution!)
func (r *RedisClient) FlushDB(ctx context.Context) error {
	if !r.enabled {
		return nil
	}

	if err := r.client.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("failed to flush DB: %w", err)
	}

	return nil
}

// Ping tests the connection
func (r *RedisClient) Ping(ctx context.Context) error {
	if !r.enabled {
		return fmt.Errorf("cache is disabled")
	}

	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}
