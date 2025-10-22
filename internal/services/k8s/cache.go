package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	lru "github.com/hashicorp/golang-lru/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CacheEntry represents a cached resource list
type CacheEntry struct {
	Data            []byte    // JSON-encoded resource list
	ResourceVersion string    // K8s ResourceVersion for invalidation
	ExpiresAt       time.Time // Expiration time
	mutex           sync.RWMutex
}

// CacheService handles caching of Kubernetes resources with LRU eviction
type CacheService struct {
	cache      *lru.Cache[string, *CacheEntry] // LRU cache with bounded size
	mu         sync.RWMutex                    // Protects cache operations
	defaultTTL time.Duration
	maxEntries int
}

const (
	// DefaultMaxCacheEntries is the default maximum number of cache entries
	DefaultMaxCacheEntries = 1000
	// DefaultCacheTTL is the default cache TTL
	DefaultCacheTTL = 5 * time.Minute
)

// NewCacheService creates a new cache service with a default TTL of 5 minutes
// and a maximum of 1000 entries (LRU eviction)
func NewCacheService() *CacheService {
	return NewCacheServiceWithConfig(DefaultMaxCacheEntries, DefaultCacheTTL)
}

// NewCacheServiceWithConfig creates a cache service with custom configuration
func NewCacheServiceWithConfig(maxEntries int, ttl time.Duration) *CacheService {
	cache, err := lru.New[string, *CacheEntry](maxEntries)
	if err != nil {
		// This should never happen with valid maxEntries
		panic(fmt.Sprintf("failed to create LRU cache: %v", err))
	}

	service := &CacheService{
		cache:      cache,
		defaultTTL: ttl,
		maxEntries: maxEntries,
	}

	// Start background cleanup goroutine for expired entries
	go service.cleanupExpired()

	return service
}

// Get retrieves a cached resource list
func (s *CacheService) Get(key string) ([]byte, bool) {
	s.mu.RLock()
	entry, exists := s.cache.Get(key) // LRU Get (thread-safe within LRU)
	s.mu.RUnlock()

	if !exists {
		return nil, false
	}

	entry.mutex.RLock()
	defer entry.mutex.RUnlock()

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry.Data, true
}

// Set stores a resource list in cache with LRU eviction
func (s *CacheService) Set(key string, data []byte, resourceVersion string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.cache.Get(key)
	if !exists {
		entry = &CacheEntry{}
	}

	entry.mutex.Lock()
	entry.Data = data
	entry.ResourceVersion = resourceVersion
	entry.ExpiresAt = time.Now().Add(s.defaultTTL)
	entry.mutex.Unlock()

	// Add to LRU cache (will evict oldest if at capacity)
	s.cache.Add(key, entry)
}

// Invalidate removes a cache entry
func (s *CacheService) Invalidate(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache.Remove(key)
}

// InvalidateByPrefix removes all cache entries with a given prefix
// Useful for invalidating all resources in a cluster or namespace
func (s *CacheService) InvalidateByPrefix(prefix string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// LRU cache requires iteration over keys
	keys := s.cache.Keys()
	for _, key := range keys {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			s.cache.Remove(key)
		}
	}
}

// CheckResourceVersion compares the cached resource version with the current one
// Returns true if the cache is still valid (resourceVersion matches)
func (s *CacheService) CheckResourceVersion(
	ctx context.Context,
	client dynamic.Interface,
	key string,
	gvr schema.GroupVersionResource,
	namespace string,
) (bool, error) {
	s.mu.RLock()
	entry, exists := s.cache.Get(key)
	s.mu.RUnlock()

	if !exists {
		return false, nil
	}

	entry.mutex.RLock()
	cachedVersion := entry.ResourceVersion
	entry.mutex.RUnlock()

	// Get current resource version from API
	var list *unstructured.UnstructuredList
	var err error

	if namespace == "" {
		list, err = client.Resource(gvr).List(ctx, metav1.ListOptions{Limit: 1})
	} else {
		list, err = client.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{Limit: 1})
	}

	if err != nil {
		return false, err
	}

	currentVersion := list.GetResourceVersion()

	// If versions match, cache is still valid
	return cachedVersion == currentVersion, nil
}

// GetOrFetch retrieves from cache or fetches from API if not cached or expired
func (s *CacheService) GetOrFetch(
	ctx context.Context,
	client dynamic.Interface,
	key string,
	gvr schema.GroupVersionResource,
	namespace string,
	fetchFunc func() (interface{}, error),
) ([]byte, error) {
	// Try to get from cache first
	if data, exists := s.Get(key); exists {
		// Optionally check if resource version is still valid
		valid, err := s.CheckResourceVersion(ctx, client, key, gvr, namespace)
		if err == nil && valid {
			return data, nil
		}
	}

	// Cache miss or invalid - fetch from API
	result, err := fetchFunc()
	if err != nil {
		return nil, err
	}

	// Serialize result
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	// Store in cache
	// Extract resource version if result is an unstructured list
	resourceVersion := ""
	if list, ok := result.(*unstructured.UnstructuredList); ok {
		resourceVersion = list.GetResourceVersion()
	}

	s.Set(key, data, resourceVersion)

	return data, nil
}

// cleanupExpired periodically removes expired cache entries
func (s *CacheService) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		s.mu.Lock()

		// Get all keys and check expiration
		keys := s.cache.Keys()
		for _, key := range keys {
			entry, exists := s.cache.Get(key)
			if !exists {
				continue
			}

			entry.mutex.RLock()
			expired := now.After(entry.ExpiresAt)
			entry.mutex.RUnlock()

			if expired {
				s.cache.Remove(key)
			}
		}

		s.mu.Unlock()
	}
}

// Stats returns cache statistics
func (s *CacheService) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalEntries := s.cache.Len()
	expiredEntries := 0
	now := time.Now()

	keys := s.cache.Keys()
	for _, key := range keys {
		entry, exists := s.cache.Get(key)
		if !exists {
			continue
		}

		entry.mutex.RLock()
		if now.After(entry.ExpiresAt) {
			expiredEntries++
		}
		entry.mutex.RUnlock()
	}

	return map[string]interface{}{
		"total_entries":   totalEntries,
		"expired_entries": expiredEntries,
		"valid_entries":   totalEntries - expiredEntries,
		"ttl_minutes":     s.defaultTTL.Minutes(),
		"max_entries":     s.maxEntries,
	}
}

// Clear removes all cache entries
func (s *CacheService) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache.Purge()
}

// GenerateCacheKey creates a consistent cache key
func GenerateCacheKey(clusterID uint, resourceType, namespace string) string {
	if namespace == "" {
		return fmt.Sprintf("%d:%s:_all", clusterID, resourceType)
	}
	return fmt.Sprintf("%d:%s:%s", clusterID, resourceType, namespace)
}
