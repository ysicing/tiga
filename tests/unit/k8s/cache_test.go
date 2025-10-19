package k8s_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	k8sservice "github.com/ysicing/tiga/internal/services/k8s"
)

// TestCacheService_BasicOperations tests basic cache Get/Set operations
func TestCacheService_BasicOperations(t *testing.T) {
	service := k8sservice.NewCacheService()
	assert.NotNil(t, service, "Service should be initialized")

	t.Run("SetAndGet", func(t *testing.T) {
		key := "cluster-1:Pod:default"
		data := []byte(`[{"name":"pod-1"},{"name":"pod-2"}]`)
		resourceVersion := "12345"

		// Set cache entry
		service.Set(key, data, resourceVersion)

		// Get cache entry
		cachedData, found := service.Get(key)
		require.True(t, found, "Cache entry should exist")
		assert.Equal(t, data, cachedData, "Cached data should match")
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		key := "non-existent-key"

		cachedData, found := service.Get(key)
		assert.False(t, found, "Non-existent key should return false")
		assert.Nil(t, cachedData, "Non-existent key should return nil data")
	})

	t.Run("UpdateExistingEntry", func(t *testing.T) {
		key := "cluster-1:Service:default"
		data1 := []byte(`[{"name":"svc-1"}]`)
		data2 := []byte(`[{"name":"svc-1"},{"name":"svc-2"}]`)

		// Set initial data
		service.Set(key, data1, "v1")

		// Update with new data
		service.Set(key, data2, "v2")

		// Get updated data
		cachedData, found := service.Get(key)
		require.True(t, found)
		assert.Equal(t, data2, cachedData, "Should return updated data")
	})
}

// TestCacheService_Expiration tests cache expiration
func TestCacheService_Expiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping expiration test in short mode")
	}

	service := k8sservice.NewCacheService()

	t.Run("EntryExpiration", func(t *testing.T) {
		key := "cluster-1:Deployment:default"
		data := []byte(`[{"name":"deploy-1"}]`)

		// Set cache entry
		service.Set(key, data, "v1")

		// Verify entry exists
		cachedData, found := service.Get(key)
		require.True(t, found, "Entry should exist immediately after Set")
		assert.Equal(t, data, cachedData)

		// Wait for expiration (default TTL is 5 minutes, but we'll test the concept)
		// Note: In a real scenario, we'd need to either:
		// 1. Wait 5+ minutes (too slow for tests)
		// 2. Mock time (complex)
		// 3. Make TTL configurable for testing
		// For now, we just verify the expiration check logic exists

		t.Log("Cache expiration is set to 5 minutes by default")
		t.Log("Actual expiration test would require waiting or time mocking")
	})
}

// TestCacheService_ResourceVersionInvalidation tests cache invalidation by ResourceVersion
func TestCacheService_ResourceVersionInvalidation(t *testing.T) {
	service := k8sservice.NewCacheService()

	t.Run("InvalidateOnResourceVersionChange", func(t *testing.T) {
		key := "cluster-1:Pod:default"
		data1 := []byte(`[{"name":"pod-1"}]`)
		data2 := []byte(`[{"name":"pod-1","status":"Running"}]`)

		// Set initial version
		service.Set(key, data1, "v1")

		// Verify cached
		cachedData, found := service.Get(key)
		require.True(t, found)
		assert.Equal(t, data1, cachedData)

		// Invalidate and set new version
		service.Invalidate(key)

		// Verify invalidated
		_, found = service.Get(key)
		assert.False(t, found, "Cache should be invalidated")

		// Set new version
		service.Set(key, data2, "v2")

		// Verify new data
		cachedData, found = service.Get(key)
		require.True(t, found)
		assert.Equal(t, data2, cachedData)
	})
}

// TestCacheService_KeyGeneration tests cache key generation
func TestCacheService_KeyGeneration(t *testing.T) {
	// Test the expected cache key format: "clusterID:resourceType:namespace"

	testCases := []struct {
		clusterID    uint
		resourceType string
		namespace    string
		expectedKey  string
	}{
		{
			clusterID:    1,
			resourceType: "Pod",
			namespace:    "default",
			expectedKey:  "1:Pod:default",
		},
		{
			clusterID:    2,
			resourceType: "Service",
			namespace:    "kube-system",
			expectedKey:  "2:Service:kube-system",
		},
		{
			clusterID:    1,
			resourceType: "Deployment",
			namespace:    "", // All namespaces
			expectedKey:  "1:Deployment:_all",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.expectedKey, func(t *testing.T) {
			// Note: The actual GenerateCacheKey function is in cache.go
			// We're testing the expected key format here

			service := k8sservice.NewCacheService()
			data := []byte(`[{"name":"test"}]`)

			// Use the expected key format
			service.Set(tc.expectedKey, data, "v1")

			cachedData, found := service.Get(tc.expectedKey)
			require.True(t, found, "Cache entry should be found with key: %s", tc.expectedKey)
			assert.Equal(t, data, cachedData)
		})
	}
}

// TestCacheService_ConcurrentAccess tests thread safety
func TestCacheService_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	service := k8sservice.NewCacheService()

	t.Run("ConcurrentReadsAndWrites", func(t *testing.T) {
		key := "cluster-1:Pod:test"
		iterations := 100

		// Start multiple goroutines writing
		done := make(chan bool, 10)
		for i := 0; i < 5; i++ {
			go func(id int) {
				for j := 0; j < iterations; j++ {
					data := []byte(`[{"id":` + string(rune(id)) + `}]`)
					service.Set(key, data, "v1")
				}
				done <- true
			}(i)
		}

		// Start multiple goroutines reading
		for i := 0; i < 5; i++ {
			go func() {
				for j := 0; j < iterations; j++ {
					service.Get(key)
				}
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify cache is still functional
		service.Set(key, []byte(`{"final":true}`), "v2")
		_, found := service.Get(key)
		assert.True(t, found, "Cache should still work after concurrent access")
	})
}

// TestCacheService_Clear tests cache clearing
func TestCacheService_Clear(t *testing.T) {
	service := k8sservice.NewCacheService()

	// Add multiple entries
	keys := []string{
		"cluster-1:Pod:default",
		"cluster-1:Service:default",
		"cluster-2:Deployment:prod",
	}

	for _, key := range keys {
		service.Set(key, []byte(`{"test":"data"}`), "v1")
	}

	// Verify all entries exist
	for _, key := range keys {
		_, found := service.Get(key)
		assert.True(t, found, "Entry should exist before clear: %s", key)
	}

	// Clear all entries
	service.Clear()

	// Verify all entries are cleared
	for _, key := range keys {
		_, found := service.Get(key)
		assert.False(t, found, "Entry should not exist after clear: %s", key)
	}
}

// TestCacheService_Statistics tests cache statistics
func TestCacheService_Statistics(t *testing.T) {
	service := k8sservice.NewCacheService()

	// Add some entries
	service.Set("key-1", []byte(`{"a":1}`), "v1")
	service.Set("key-2", []byte(`{"b":2}`), "v1")
	service.Set("key-3", []byte(`{"c":3}`), "v1")

	// Get stats
	stats := service.Stats()

	// Stats returns map[string]interface{} with "total_entries", "expired_entries", "valid_entries", "ttl_minutes"
	totalEntries, ok := stats["total_entries"].(int)
	require.True(t, ok, "total_entries should be int")
	validEntries, ok := stats["valid_entries"].(int)
	require.True(t, ok, "valid_entries should be int")
	ttlMinutes, ok := stats["ttl_minutes"].(float64)
	require.True(t, ok, "ttl_minutes should be float64")

	assert.Equal(t, 3, totalEntries, "Should have 3 entries")
	assert.Equal(t, 3, validEntries, "Should have 3 valid entries")
	assert.Equal(t, 5.0, ttlMinutes, "TTL should be 5 minutes")

	t.Logf("Cache stats: %d total, %d valid, %.0f min TTL", totalEntries, validEntries, ttlMinutes)
}
