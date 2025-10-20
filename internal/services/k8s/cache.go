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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CacheEntry represents a cached resource list
type CacheEntry struct {
	Data            []byte    // JSON-encoded resource list
	ResourceVersion string    // K8s ResourceVersion for invalidation
	ExpiresAt       time.Time // Expiration time
	mutex           sync.RWMutex
}

// CacheService handles caching of Kubernetes resources
type CacheService struct {
	cache      map[string]*CacheEntry // Key: "clusterID:resourceType:namespace"
	mu         sync.RWMutex
	defaultTTL time.Duration
}

// NewCacheService creates a new cache service with a default TTL of 5 minutes
func NewCacheService() *CacheService {
	service := &CacheService{
		cache:      make(map[string]*CacheEntry),
		defaultTTL: 5 * time.Minute,
	}

	// Start background cleanup goroutine
	go service.cleanupExpired()

	return service
}

// Get retrieves a cached resource list
func (s *CacheService) Get(key string) ([]byte, bool) {
	s.mu.RLock()
	entry, exists := s.cache[key]
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

// Set stores a resource list in cache
func (s *CacheService) Set(key string, data []byte, resourceVersion string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.cache[key]
	if !exists {
		entry = &CacheEntry{}
		s.cache[key] = entry
	}

	entry.mutex.Lock()
	defer entry.mutex.Unlock()

	entry.Data = data
	entry.ResourceVersion = resourceVersion
	entry.ExpiresAt = time.Now().Add(s.defaultTTL)
}

// Invalidate removes a cache entry
func (s *CacheService) Invalidate(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.cache, key)
}

// InvalidateByPrefix removes all cache entries with a given prefix
// Useful for invalidating all resources in a cluster or namespace
func (s *CacheService) InvalidateByPrefix(prefix string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key := range s.cache {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(s.cache, key)
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
	entry, exists := s.cache[key]
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

		for key, entry := range s.cache {
			entry.mutex.RLock()
			expired := now.After(entry.ExpiresAt)
			entry.mutex.RUnlock()

			if expired {
				delete(s.cache, key)
			}
		}

		s.mu.Unlock()
	}
}

// Stats returns cache statistics
func (s *CacheService) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalEntries := len(s.cache)
	expiredEntries := 0
	now := time.Now()

	for _, entry := range s.cache {
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
	}
}

// Clear removes all cache entries
func (s *CacheService) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache = make(map[string]*CacheEntry)
}

// GenerateCacheKey creates a consistent cache key
func GenerateCacheKey(clusterID uint, resourceType, namespace string) string {
	if namespace == "" {
		return fmt.Sprintf("%d:%s:_all", clusterID, resourceType)
	}
	return fmt.Sprintf("%d:%s:%s", clusterID, resourceType, namespace)
}
