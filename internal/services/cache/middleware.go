package cache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CacheMiddleware creates a middleware for caching HTTP responses
func CacheMiddleware(manager *CacheManager, cacheTTL time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only cache GET requests
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		// Skip if cache is disabled
		if !manager.IsEnabled() {
			c.Next()
			return
		}

		// Build cache key from request
		cacheKey := buildHTTPCacheKey(c.Request)

		// Try to get from cache
		cached, err := manager.client.Get(c.Request.Context(), cacheKey)
		if err == nil {
			// Cache hit - return cached response
			c.Data(http.StatusOK, "application/json", []byte(cached))
			c.Abort()
			return
		}

		// Cache miss - capture response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		c.Next()

		// Cache successful responses (200-299)
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			manager.client.Set(c.Request.Context(), cacheKey, writer.body.Bytes(), cacheTTL)
		}
	}
}

// responseWriter captures response body for caching
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// buildHTTPCacheKey builds a cache key from HTTP request
func buildHTTPCacheKey(r *http.Request) string {
	// Include URL, query parameters, and relevant headers
	keyParts := []string{
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
	}

	// Include Authorization header if present (for user-specific caching)
	if auth := r.Header.Get("Authorization"); auth != "" {
		keyParts = append(keyParts, auth)
	}

	key := strings.Join(keyParts, "|")

	// Hash the key
	hasher := sha256.New()
	hasher.Write([]byte(key))
	return "http:" + hex.EncodeToString(hasher.Sum(nil))[:16]
}

// CacheableEndpoint wraps a handler to make it cacheable
func CacheableEndpoint(manager *CacheManager, ttl time.Duration, handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !manager.IsEnabled() {
			handler(c)
			return
		}

		cacheKey := buildHTTPCacheKey(c.Request)

		// Try cache
		if cached, err := manager.client.Get(c.Request.Context(), cacheKey); err == nil {
			c.Data(http.StatusOK, "application/json", []byte(cached))
			return
		}

		// Capture response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		handler(c)

		// Cache on success
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			manager.client.Set(c.Request.Context(), cacheKey, writer.body.Bytes(), ttl)
		}
	}
}

// InvalidateCacheMiddleware adds cache invalidation helpers to context
func InvalidateCacheMiddleware(manager *CacheManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add invalidation functions to context
		c.Set("invalidateCache", func(patterns ...string) error {
			for _, pattern := range patterns {
				if err := manager.client.DeletePattern(c.Request.Context(), pattern); err != nil {
					return err
				}
			}
			return nil
		})

		c.Next()

		// Auto-invalidate on write operations
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			// Invalidate related caches based on URL
			pattern := buildInvalidationPattern(c.Request.URL.Path)
			manager.client.DeletePattern(c.Request.Context(), pattern)
		}
	}
}

// buildInvalidationPattern builds a pattern for cache invalidation
func buildInvalidationPattern(urlPath string) string {
	// Extract resource type from URL
	parts := strings.Split(strings.Trim(urlPath, "/"), "/")
	if len(parts) > 0 {
		return "http:*" + parts[0] + "*"
	}
	return "http:*"
}

// WarmupHelper provides helper functions for cache warmup
type WarmupHelper struct {
	manager *CacheManager
}

// NewWarmupHelper creates a new warmup helper
func NewWarmupHelper(manager *CacheManager) *WarmupHelper {
	return &WarmupHelper{
		manager: manager,
	}
}

// WarmupEndpoint warms up cache for a specific endpoint
func (wh *WarmupHelper) WarmupEndpoint(ctx context.Context, url string, data interface{}) error {
	if !wh.manager.IsEnabled() {
		return nil
	}

	cacheKey := "warmup:" + url
	return wh.manager.client.SetJSON(ctx, cacheKey, data, wh.manager.config.DefaultTTL)
}

// WarmupMultiple warms up multiple endpoints concurrently
func (wh *WarmupHelper) WarmupMultiple(ctx context.Context, warmups map[string]interface{}) error {
	if !wh.manager.IsEnabled() {
		return nil
	}

	for url, data := range warmups {
		if err := wh.WarmupEndpoint(ctx, url, data); err != nil {
			// Log error but continue
			continue
		}
	}

	return nil
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	HitRate     float64 `json:"hit_rate"`
	TotalKeys   int64   `json:"total_keys"`
	MemoryUsage int64   `json:"memory_usage_bytes"`
	Evictions   int64   `json:"evictions"`
	ExpiredKeys int64   `json:"expired_keys"`
}

// GetCacheStats retrieves cache statistics
func GetCacheStats(ctx context.Context, client *RedisClient) (*CacheStats, error) {
	if !client.IsEnabled() {
		return &CacheStats{}, nil
	}

	// Get stats from Redis INFO command
	// This is a simplified version - real implementation would parse INFO output
	stats := &CacheStats{
		Hits:   0,
		Misses: 0,
	}

	// Calculate hit rate
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}

	return stats, nil
}

// HealthCheck performs a cache health check
func HealthCheck(ctx context.Context, client *RedisClient) error {
	if !client.IsEnabled() {
		return nil
	}

	return client.Ping(ctx)
}

// CacheTester provides testing utilities for cache
type CacheTester struct {
	manager *CacheManager
}

// NewCacheTester creates a new cache tester
func NewCacheTester(manager *CacheManager) *CacheTester {
	return &CacheTester{
		manager: manager,
	}
}

// TestCacheRoundTrip tests cache read/write
func (ct *CacheTester) TestCacheRoundTrip(ctx context.Context) error {
	if !ct.manager.IsEnabled() {
		return ErrCacheDisabled
	}

	testKey := "test:roundtrip"
	testValue := "test-value-" + time.Now().Format(time.RFC3339Nano)

	// Write
	if err := ct.manager.client.Set(ctx, testKey, testValue, 1*time.Minute); err != nil {
		return err
	}

	// Read
	retrieved, err := ct.manager.client.Get(ctx, testKey)
	if err != nil {
		return err
	}

	// Verify
	if retrieved != testValue {
		return ErrCacheMiss
	}

	// Cleanup
	return ct.manager.client.Delete(ctx, testKey)
}

// MockCache provides a mock cache for testing (in-memory)
type MockCache struct {
	data map[string]interface{}
}

// NewMockCache creates a new mock cache
func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]interface{}),
	}
}

// Get retrieves from mock cache
func (mc *MockCache) Get(ctx context.Context, key string) (string, error) {
	val, ok := mc.data[key]
	if !ok {
		return "", ErrCacheMiss
	}
	return val.(string), nil
}

// Set stores in mock cache
func (mc *MockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	mc.data[key] = value
	return nil
}

// Delete removes from mock cache
func (mc *MockCache) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		delete(mc.data, key)
	}
	return nil
}

// Exists checks existence in mock cache
func (mc *MockCache) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := mc.data[key]
	return ok, nil
}

// GetJSON retrieves JSON from mock cache
func (mc *MockCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	return ErrCacheMiss
}

// SetJSON stores JSON in mock cache
func (mc *MockCache) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	mc.data[key] = value
	return nil
}

// DeletePattern deletes pattern in mock cache
func (mc *MockCache) DeletePattern(ctx context.Context, pattern string) error {
	return nil
}

// IsEnabled returns true for mock cache
func (mc *MockCache) IsEnabled() bool {
	return true
}
