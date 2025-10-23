package docker

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	defaultCacheTTL = 5 * time.Minute
)

// cacheEntry represents a cached item with expiration
type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// CacheKey represents the structure of cache keys
type CacheKey struct {
	InstanceID   uuid.UUID
	ResourceType string // "containers", "images", "volumes", "networks"
	Filter       string // Optional filter string for fine-grained caching
}

// String returns string representation of cache key
func (k CacheKey) String() string {
	if k.Filter != "" {
		return k.InstanceID.String() + ":" + k.ResourceType + ":" + k.Filter
	}
	return k.InstanceID.String() + ":" + k.ResourceType
}

// DockerCacheService manages cached Docker API responses
type DockerCacheService struct {
	cache map[string]*cacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewDockerCacheService creates a new DockerCacheService
func NewDockerCacheService() *DockerCacheService {
	service := &DockerCacheService{
		cache: make(map[string]*cacheEntry),
		ttl:   defaultCacheTTL,
	}

	// Start background cleanup goroutine
	go service.cleanupExpired()

	return service
}

// NewDockerCacheServiceWithTTL creates a new DockerCacheService with custom TTL
func NewDockerCacheServiceWithTTL(ttl time.Duration) *DockerCacheService {
	service := &DockerCacheService{
		cache: make(map[string]*cacheEntry),
		ttl:   ttl,
	}

	go service.cleanupExpired()

	return service
}

// Get retrieves a value from cache
func (s *DockerCacheService) Get(key CacheKey) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.cache[key.String()]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	logrus.WithFields(logrus.Fields{
		"cache_key": key.String(),
		"ttl_left":  time.Until(entry.expiresAt).String(),
	}).Debug("Cache hit")

	return entry.data, true
}

// Set stores a value in cache with default TTL
func (s *DockerCacheService) Set(key CacheKey, data interface{}) {
	s.SetWithTTL(key, data, s.ttl)
}

// SetWithTTL stores a value in cache with custom TTL
func (s *DockerCacheService) SetWithTTL(key CacheKey, data interface{}, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache[key.String()] = &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}

	logrus.WithFields(logrus.Fields{
		"cache_key": key.String(),
		"ttl":       ttl.String(),
	}).Debug("Cache set")
}

// Delete removes a specific cache entry
func (s *DockerCacheService) Delete(key CacheKey) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.cache, key.String())

	logrus.WithField("cache_key", key.String()).Debug("Cache entry deleted")
}

// DeleteByInstance removes all cache entries for a specific instance
func (s *DockerCacheService) DeleteByInstance(instanceID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prefix := instanceID.String() + ":"
	deletedCount := 0

	for key := range s.cache {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(s.cache, key)
			deletedCount++
		}
	}

	logrus.WithFields(logrus.Fields{
		"instance_id":   instanceID,
		"deleted_count": deletedCount,
	}).Debug("Deleted cache entries for instance")
}

// DeleteByInstanceAndType removes all cache entries for a specific instance and resource type
func (s *DockerCacheService) DeleteByInstanceAndType(instanceID uuid.UUID, resourceType string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prefix := instanceID.String() + ":" + resourceType
	deletedCount := 0

	for key := range s.cache {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(s.cache, key)
			deletedCount++
		}
	}

	logrus.WithFields(logrus.Fields{
		"instance_id":   instanceID,
		"resource_type": resourceType,
		"deleted_count": deletedCount,
	}).Debug("Deleted cache entries for instance and type")
}

// Clear removes all cache entries
func (s *DockerCacheService) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := len(s.cache)
	s.cache = make(map[string]*cacheEntry)

	logrus.WithField("cleared_count", count).Info("Cache cleared")
}

// InvalidateOnHealthChange invalidates cache when instance health status changes
// This should be called when an instance goes offline or changes state
func (s *DockerCacheService) InvalidateOnHealthChange(instanceID uuid.UUID, newStatus string) {
	logrus.WithFields(logrus.Fields{
		"instance_id": instanceID,
		"new_status":  newStatus,
	}).Debug("Invalidating cache due to health status change")

	// Clear all cache for this instance when it goes offline or becomes unavailable
	if newStatus == "offline" || newStatus == "archived" || newStatus == "unknown" {
		s.DeleteByInstance(instanceID)
	}
}

// GetCacheStats returns cache statistics
func (s *DockerCacheService) GetCacheStats() CacheStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.cache)
	expired := 0
	now := time.Now()

	for _, entry := range s.cache {
		if now.After(entry.expiresAt) {
			expired++
		}
	}

	return CacheStats{
		Total:   total,
		Active:  total - expired,
		Expired: expired,
	}
}

// cleanupExpired periodically removes expired entries
func (s *DockerCacheService) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()

		now := time.Now()
		deletedCount := 0

		for key, entry := range s.cache {
			if now.After(entry.expiresAt) {
				delete(s.cache, key)
				deletedCount++
			}
		}

		s.mu.Unlock()

		if deletedCount > 0 {
			logrus.WithField("deleted_count", deletedCount).Debug("Cleaned up expired cache entries")
		}
	}
}

// CacheStats represents cache statistics
type CacheStats struct {
	Total   int `json:"total"`
	Active  int `json:"active"`
	Expired int `json:"expired"`
}

// Helper methods for common cache operations

// GetContainersCache retrieves cached containers list
func (s *DockerCacheService) GetContainersCache(instanceID uuid.UUID, filter string) (interface{}, bool) {
	key := CacheKey{
		InstanceID:   instanceID,
		ResourceType: "containers",
		Filter:       filter,
	}
	return s.Get(key)
}

// SetContainersCache stores containers list in cache
func (s *DockerCacheService) SetContainersCache(instanceID uuid.UUID, filter string, data interface{}) {
	key := CacheKey{
		InstanceID:   instanceID,
		ResourceType: "containers",
		Filter:       filter,
	}
	s.Set(key, data)
}

// GetImagesCache retrieves cached images list
func (s *DockerCacheService) GetImagesCache(instanceID uuid.UUID, filter string) (interface{}, bool) {
	key := CacheKey{
		InstanceID:   instanceID,
		ResourceType: "images",
		Filter:       filter,
	}
	return s.Get(key)
}

// SetImagesCache stores images list in cache
func (s *DockerCacheService) SetImagesCache(instanceID uuid.UUID, filter string, data interface{}) {
	key := CacheKey{
		InstanceID:   instanceID,
		ResourceType: "images",
		Filter:       filter,
	}
	s.Set(key, data)
}

// InvalidateContainersCache invalidates containers cache for an instance
func (s *DockerCacheService) InvalidateContainersCache(instanceID uuid.UUID) {
	s.DeleteByInstanceAndType(instanceID, "containers")
}

// InvalidateImagesCache invalidates images cache for an instance
func (s *DockerCacheService) InvalidateImagesCache(instanceID uuid.UUID) {
	s.DeleteByInstanceAndType(instanceID, "images")
}
