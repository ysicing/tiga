package k8s

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SearchResult represents a search result item
type SearchResult struct {
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	Score     int               `json:"score"`
	MatchType string            `json:"match_type"` // exact, name, label, annotation
	Created   time.Time         `json:"created"`
}

// SearchService handles global resource searching
type SearchService struct {
	cacheService *CacheService
	searchCache  map[string]*searchCacheEntry
	cacheMutex   sync.RWMutex
}

type searchCacheEntry struct {
	Results   []SearchResult
	ExpiresAt time.Time
}

// NewSearchService creates a new search service
func NewSearchService(cacheService *CacheService) *SearchService {
	service := &SearchService{
		cacheService: cacheService,
		searchCache:  make(map[string]*searchCacheEntry),
	}

	// Start background cleanup
	go service.cleanupSearchCache()

	return service
}

// Search performs a global search across multiple resource types
func (s *SearchService) Search(
	ctx context.Context,
	client dynamic.Interface,
	query string,
	resourceTypes []string,
	namespace string,
	limit int,
) ([]SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	if limit <= 0 || limit > 50 {
		limit = 50 // Default and max limit
	}

	// Check cache first
	cacheKey := s.generateSearchCacheKey(query, resourceTypes, namespace)
	if results, exists := s.getFromSearchCache(cacheKey); exists {
		return s.limitResults(results, limit), nil
	}

	// Default resource types if none specified
	if len(resourceTypes) == 0 {
		resourceTypes = []string{"Pod", "Deployment", "Service", "ConfigMap"}
	}

	// Concurrent search across resource types
	resultsChan := make(chan []SearchResult, len(resourceTypes))
	errorsChan := make(chan error, len(resourceTypes))
	var wg sync.WaitGroup

	for _, resourceType := range resourceTypes {
		wg.Add(1)
		go func(resType string) {
			defer wg.Done()

			results, err := s.searchResourceType(ctx, client, query, resType, namespace)
			if err != nil {
				errorsChan <- err
				return
			}

			resultsChan <- results
		}(resourceType)
	}

	// Wait for all searches to complete
	wg.Wait()
	close(resultsChan)
	close(errorsChan)

	// Collect all results
	var allResults []SearchResult
	for results := range resultsChan {
		allResults = append(allResults, results...)
	}

	// Sort by score (descending)
	s.sortByScore(allResults)

	// Cache the results for 5 minutes
	s.setSearchCache(cacheKey, allResults)

	// Return limited results
	return s.limitResults(allResults, limit), nil
}

// searchResourceType searches a specific resource type
func (s *SearchService) searchResourceType(
	ctx context.Context,
	client dynamic.Interface,
	query string,
	resourceType string,
	namespace string,
) ([]SearchResult, error) {
	gvr, err := s.getGVRForResourceType(resourceType)
	if err != nil {
		return nil, err
	}

	var list *unstructured.UnstructuredList
	if namespace == "" {
		list, err = client.Resource(gvr).List(ctx, metav1.ListOptions{})
	} else {
		list, err = client.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return nil, err
	}

	var results []SearchResult
	queryLower := strings.ToLower(query)

	for _, item := range list.Items {
		score, matchType := s.scoreResource(item, queryLower)
		if score > 0 {
			results = append(results, SearchResult{
				Kind:      item.GetKind(),
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
				Labels:    item.GetLabels(),
				Score:     score,
				MatchType: matchType,
				Created:   item.GetCreationTimestamp().Time,
			})
		}
	}

	return results, nil
}

// scoreResource calculates a score for a resource based on the query
func (s *SearchService) scoreResource(resource unstructured.Unstructured, queryLower string) (int, string) {
	name := strings.ToLower(resource.GetName())

	// Exact name match: 100 points
	if name == queryLower {
		return 100, "exact"
	}

	// Name contains query: 80 points
	if strings.Contains(name, queryLower) {
		return 80, "name"
	}

	// Label key or value match: 60 points
	labels := resource.GetLabels()
	for key, value := range labels {
		keyLower := strings.ToLower(key)
		valueLower := strings.ToLower(value)

		if keyLower == queryLower || valueLower == queryLower ||
			strings.Contains(keyLower, queryLower) || strings.Contains(valueLower, queryLower) {
			return 60, "label"
		}
	}

	// Annotation key or value match: 40 points
	annotations := resource.GetAnnotations()
	for key, value := range annotations {
		keyLower := strings.ToLower(key)
		valueLower := strings.ToLower(value)

		if keyLower == queryLower || valueLower == queryLower ||
			strings.Contains(keyLower, queryLower) || strings.Contains(valueLower, queryLower) {
			return 40, "annotation"
		}
	}

	return 0, ""
}

// sortByScore sorts results by score in descending order (bubble sort for simplicity)
func (s *SearchService) sortByScore(results []SearchResult) {
	n := len(results)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if results[j].Score < results[j+1].Score {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
}

// limitResults limits the number of results returned
func (s *SearchService) limitResults(results []SearchResult, limit int) []SearchResult {
	if len(results) <= limit {
		return results
	}
	return results[:limit]
}

// getGVRForResourceType maps resource type to GVR
func (s *SearchService) getGVRForResourceType(resourceType string) (schema.GroupVersionResource, error) {
	gvrMap := map[string]schema.GroupVersionResource{
		"Pod":        {Group: "", Version: "v1", Resource: "pods"},
		"Deployment": {Group: "apps", Version: "v1", Resource: "deployments"},
		"Service":    {Group: "", Version: "v1", Resource: "services"},
		"ConfigMap":  {Group: "", Version: "v1", Resource: "configmaps"},
		"Secret":     {Group: "", Version: "v1", Resource: "secrets"},
		"Ingress":    {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
	}

	gvr, exists := gvrMap[resourceType]
	if !exists {
		return schema.GroupVersionResource{}, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	return gvr, nil
}

// generateSearchCacheKey creates a cache key for search results
func (s *SearchService) generateSearchCacheKey(query string, resourceTypes []string, namespace string) string {
	typeStr := strings.Join(resourceTypes, ",")
	return fmt.Sprintf("search:%s:%s:%s", query, typeStr, namespace)
}

// getFromSearchCache retrieves cached search results
func (s *SearchService) getFromSearchCache(key string) ([]SearchResult, bool) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	entry, exists := s.searchCache[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry.Results, true
}

// setSearchCache stores search results in cache
func (s *SearchService) setSearchCache(key string, results []SearchResult) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.searchCache[key] = &searchCacheEntry{
		Results:   results,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
}

// cleanupSearchCache periodically removes expired search cache entries
func (s *SearchService) cleanupSearchCache() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		s.cacheMutex.Lock()

		for key, entry := range s.searchCache {
			if now.After(entry.ExpiresAt) {
				delete(s.searchCache, key)
			}
		}

		s.cacheMutex.Unlock()
	}
}

// ClearSearchCache clears all search cache entries
func (s *SearchService) ClearSearchCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.searchCache = make(map[string]*searchCacheEntry)
}
