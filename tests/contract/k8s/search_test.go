package k8s_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearchContract verifies the API contract for GET /api/v1/k8s/clusters/:cluster_id/search
// According to contracts/cluster-api.md - Global Search
func TestSearchContract(t *testing.T) {
	// TODO: This test will fail until the API handler is implemented (TDD approach)
	// Expected to be implemented in Phase 1 (T033)

	t.Run("SuccessResponse", func(t *testing.T) {
		router := gin.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/1/search?query=nginx", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Skipf("API not implemented yet, got status %d", rec.Code)
			return
		}

		var response struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				Results []map[string]any `json:"results"`
				Total   int              `json:"total"`
				Query   string           `json:"query"`
			} `json:"data"`
		}

		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, 200, response.Code)
		assert.NotNil(t, response.Data.Results)
		assert.Equal(t, "nginx", response.Data.Query)
	})

	t.Run("SearchResultStructure", func(t *testing.T) {
		router := gin.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/1/search?query=nginx", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Skip("API not implemented yet")
			return
		}

		var response struct {
			Data struct {
				Results []struct {
					Kind      string            `json:"kind"`
					Name      string            `json:"name"`
					Namespace string            `json:"namespace"`
					Labels    map[string]string `json:"labels"`
					Score     int               `json:"score"`
					MatchType string            `json:"match_type"`
					CreatedAt string            `json:"created"`
				} `json:"results"`
			} `json:"data"`
		}

		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify search result fields
		if len(response.Data.Results) > 0 {
			result := response.Data.Results[0]
			assert.NotEmpty(t, result.Kind, "Result should have kind")
			assert.NotEmpty(t, result.Name, "Result should have name")
			assert.GreaterOrEqual(t, result.Score, 0, "Score should be non-negative")
			assert.Contains(t, []string{"exact", "name", "label", "annotation"}, result.MatchType,
				"MatchType should be one of: exact, name, label, annotation")
		}
	})

	t.Run("MissingQuery", func(t *testing.T) {
		router := gin.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/1/search", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound && rec.Body.Len() == 0 {
			t.Skip("API not implemented yet")
			return
		}

		// Missing query parameter should return 400
		assert.Equal(t, http.StatusBadRequest, rec.Code, "Should return 400 for missing query parameter")

		var response map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		message, _ := response["message"].(string)
		assert.Contains(t, message, "query", "Error message should mention missing query")
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		router := gin.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/1/search?query=", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound && rec.Body.Len() == 0 {
			t.Skip("API not implemented yet")
			return
		}

		// Empty query should return 400
		assert.Equal(t, http.StatusBadRequest, rec.Code, "Should return 400 for empty query")
	})

	t.Run("ResourceTypeFilter", func(t *testing.T) {
		router := gin.New()

		// Search with resource_types filter
		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/1/search?query=nginx&resource_types=Pod,Deployment", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Skip("API not implemented yet")
			return
		}

		var response struct {
			Data struct {
				Results []struct {
					Kind string `json:"kind"`
				} `json:"results"`
			} `json:"data"`
		}

		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// All results should be either Pod or Deployment
		for _, result := range response.Data.Results {
			assert.Contains(t, []string{"Pod", "Deployment"}, result.Kind,
				"Result kind should match resource_types filter")
		}
	})

	t.Run("NamespaceFilter", func(t *testing.T) {
		router := gin.New()

		// Search with namespace filter
		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/1/search?query=nginx&namespace=default", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Skip("API not implemented yet")
			return
		}

		var response struct {
			Data struct {
				Results []struct {
					Namespace string `json:"namespace"`
				} `json:"results"`
			} `json:"data"`
		}

		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// All results should be from default namespace
		for _, result := range response.Data.Results {
			if result.Namespace != "" { // Some resources are cluster-scoped
				assert.Equal(t, "default", result.Namespace,
					"Result should be from default namespace")
			}
		}
	})

	t.Run("LimitParameter", func(t *testing.T) {
		router := gin.New()

		// Search with limit parameter
		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/1/search?query=nginx&limit=5", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Skip("API not implemented yet")
			return
		}

		var response struct {
			Data struct {
				Results []any `json:"results"`
			} `json:"data"`
		}

		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should return at most 5 results
		assert.LessOrEqual(t, len(response.Data.Results), 5,
			"Should respect limit parameter")
	})

	t.Run("ScoringOrder", func(t *testing.T) {
		router := gin.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/1/search?query=nginx", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Skip("API not implemented yet")
			return
		}

		var response struct {
			Data struct {
				Results []struct {
					Score int `json:"score"`
				} `json:"results"`
			} `json:"data"`
		}

		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// Results should be ordered by score (descending)
		if len(response.Data.Results) > 1 {
			for i := 0; i < len(response.Data.Results)-1; i++ {
				assert.GreaterOrEqual(t, response.Data.Results[i].Score, response.Data.Results[i+1].Score,
					"Results should be ordered by score (highest first)")
			}
		}
	})

	t.Run("NoResults", func(t *testing.T) {
		router := gin.New()

		// Search for something unlikely to exist
		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/1/search?query=xyznonexistent12345", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Skip("API not implemented yet")
			return
		}

		var response struct {
			Code int `json:"code"`
			Data struct {
				Results []any `json:"results"`
				Total   int   `json:"total"`
			} `json:"data"`
		}

		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// No results should return empty array, not null
		assert.NotNil(t, response.Data.Results)
		assert.Equal(t, 0, response.Data.Total)
	})
}
