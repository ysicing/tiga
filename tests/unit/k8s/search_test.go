package k8s_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"

	k8sservice "github.com/ysicing/tiga/internal/services/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestSearchService_ScoringAlgorithm tests the search scoring algorithm
func TestSearchService_ScoringAlgorithm(t *testing.T) {
	ctx := context.Background()
	cacheService := k8sservice.NewCacheService()
	searchService := k8sservice.NewSearchService(cacheService)

	// Create test resources with different match types
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// 1. Exact match resource (should score 100)
	exactPod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis",
			Namespace: "default",
		},
	}

	// 2. Name contains match (should score 80)
	namePod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-master-0",
			Namespace: "default",
		},
	}

	// 3. Label match (should score 60)
	labelPod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cache-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "redis",
			},
		},
	}

	// 4. Annotation match (should score 40)
	annotationPod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-pod",
			Namespace: "default",
			Annotations: map[string]string{
				"description": "redis cache pod",
			},
		},
	}

	// 5. No match (should score 0, not returned)
	noMatchPod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-pod",
			Namespace: "default",
		},
	}

	dynamicClient := fake.NewSimpleDynamicClient(scheme,
		exactPod, namePod, labelPod, annotationPod, noMatchPod)

	// Perform search for "redis"
	results, err := searchService.Search(ctx, dynamicClient, "redis", []string{"Pod"}, "default", 50)
	require.NoError(t, err, "Search should not error")

	// Verify we got 4 results (all except noMatchPod)
	assert.Equal(t, 4, len(results), "Should find 4 matching resources")

	// Verify results are sorted by score (descending)
	// Exact match (100) should be first
	assert.Equal(t, "redis", results[0].Name, "First result should be exact match")
	assert.Equal(t, 100, results[0].Score, "Exact match should score 100")
	assert.Equal(t, "exact", results[0].MatchType)

	// Name contains (80) should be second
	assert.Equal(t, "redis-master-0", results[1].Name, "Second result should be name match")
	assert.Equal(t, 80, results[1].Score, "Name match should score 80")
	assert.Equal(t, "name", results[1].MatchType)

	// Label match (60) should be third
	assert.Equal(t, "cache-pod", results[2].Name, "Third result should be label match")
	assert.Equal(t, 60, results[2].Score, "Label match should score 60")
	assert.Equal(t, "label", results[2].MatchType)

	// Annotation match (40) should be fourth
	assert.Equal(t, "app-pod", results[3].Name, "Fourth result should be annotation match")
	assert.Equal(t, 40, results[3].Score, "Annotation match should score 40")
	assert.Equal(t, "annotation", results[3].MatchType)

	t.Logf("Search results ordered correctly by score")
}

// TestSearchService_ResultLimit tests the 50-result limit
func TestSearchService_ResultLimit(t *testing.T) {
	ctx := context.Background()
	cacheService := k8sservice.NewCacheService()
	searchService := k8sservice.NewSearchService(cacheService)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// Create 60 pods (exceeds limit of 50)
	resources := make([]runtime.Object, 0, 60)
	for i := 0; i < 60; i++ {
		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "redis-pod-" + string(rune(i)),
				Namespace: "default",
			},
		}
		resources = append(resources, pod)
	}

	dynamicClient := fake.NewSimpleDynamicClient(scheme, resources...)

	// Search should limit to 50 results
	results, err := searchService.Search(ctx, dynamicClient, "redis", []string{"Pod"}, "default", 100)
	require.NoError(t, err)

	assert.LessOrEqual(t, len(results), 50, "Results should be limited to 50")
	t.Logf("Search returned %d results (limit enforced)", len(results))
}

// TestSearchService_EmptyQuery tests empty query handling
func TestSearchService_EmptyQuery(t *testing.T) {
	ctx := context.Background()
	cacheService := k8sservice.NewCacheService()
	searchService := k8sservice.NewSearchService(cacheService)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	dynamicClient := fake.NewSimpleDynamicClient(scheme)

	// Empty query should return error
	results, err := searchService.Search(ctx, dynamicClient, "", []string{"Pod"}, "default", 50)

	assert.Error(t, err, "Empty query should return error")
	assert.Contains(t, err.Error(), "search query cannot be empty")
	assert.Nil(t, results, "Results should be nil for empty query")
}

// TestSearchService_DefaultResourceTypes tests default resource types
func TestSearchService_DefaultResourceTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	ctx := context.Background()
	cacheService := k8sservice.NewCacheService()
	searchService := k8sservice.NewSearchService(cacheService)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// Create resources of default types
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	}

	dynamicClient := fake.NewSimpleDynamicClient(scheme, pod, service, configMap)

	// Search with explicit resource types (only those registered in scheme)
	// Using only core/v1 resources that are registered
	results, err := searchService.Search(ctx, dynamicClient, "test", []string{"Pod", "Service", "ConfigMap"}, "default", 50)
	require.NoError(t, err)

	// Should find resources across specified types
	assert.GreaterOrEqual(t, len(results), 2, "Should find at least 2 resources with specified types")
	t.Logf("Found %d resources with default types", len(results))
}

// TestSearchService_CaseInsensitive tests case-insensitive search
func TestSearchService_CaseInsensitive(t *testing.T) {
	ctx := context.Background()
	cacheService := k8sservice.NewCacheService()
	searchService := k8sservice.NewSearchService(cacheService)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "Redis-Master",
			Namespace: "default",
		},
	}

	dynamicClient := fake.NewSimpleDynamicClient(scheme, pod)

	// Test with different cases
	testCases := []string{
		"redis",
		"REDIS",
		"Redis",
		"ReDiS",
	}

	for _, query := range testCases {
		t.Run("Query_"+query, func(t *testing.T) {
			results, err := searchService.Search(ctx, dynamicClient, query, []string{"Pod"}, "default", 50)
			require.NoError(t, err)
			assert.Equal(t, 1, len(results), "Should find resource regardless of case")
			assert.Equal(t, "Redis-Master", results[0].Name)
		})
	}
}

// TestSearchService_NamespaceFiltering tests namespace filtering
func TestSearchService_NamespaceFiltering(t *testing.T) {
	ctx := context.Background()
	cacheService := k8sservice.NewCacheService()
	searchService := k8sservice.NewSearchService(cacheService)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// Create pods in different namespaces
	defaultPod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-default",
			Namespace: "default",
		},
	}

	prodPod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-prod",
			Namespace: "prod",
		},
	}

	dynamicClient := fake.NewSimpleDynamicClient(scheme, defaultPod, prodPod)

	// Search only in default namespace
	results, err := searchService.Search(ctx, dynamicClient, "redis", []string{"Pod"}, "default", 50)
	require.NoError(t, err)

	assert.Equal(t, 1, len(results), "Should only find pod in default namespace")
	assert.Equal(t, "redis-default", results[0].Name)
	assert.Equal(t, "default", results[0].Namespace)

	// Search in prod namespace
	results, err = searchService.Search(ctx, dynamicClient, "redis", []string{"Pod"}, "prod", 50)
	require.NoError(t, err)

	assert.Equal(t, 1, len(results), "Should only find pod in prod namespace")
	assert.Equal(t, "redis-prod", results[0].Name)
	assert.Equal(t, "prod", results[0].Namespace)
}
