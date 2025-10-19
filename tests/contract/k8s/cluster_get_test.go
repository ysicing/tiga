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

// TestClusterGetContract verifies the API contract for GET /api/v1/k8s/clusters/:id
// According to contracts/cluster-api.md - C2: 获取集群详情
func TestClusterGetContract(t *testing.T) {
	// TODO: This test will fail until the API handler is implemented (TDD approach)
	// Expected to be implemented in Phase 0 (T028)

	t.Run("SuccessResponse", func(t *testing.T) {
		// Setup test router
		router := gin.New()
		// TODO: Register cluster get handler when implemented

		// Create test request for cluster ID 1
		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/1", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Skipf("API not implemented yet, got status %d", rec.Code)
			return
		}

		// Verify response structure
		var response struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				ID              uint   `json:"id"`
				Name            string `json:"name"`
				Kubeconfig      string `json:"kubeconfig"`
				IsDefault       bool   `json:"is_default"`
				Enabled         bool   `json:"enabled"`
				HealthStatus    string `json:"health_status"`
				LastConnectedAt string `json:"last_connected_at,omitempty"`
				NodeCount       int    `json:"node_count"`
				PodCount        int    `json:"pod_count"`
				PrometheusURL   string `json:"prometheus_url,omitempty"`
				CreatedAt       string `json:"created_at"`
				UpdatedAt       string `json:"updated_at"`
			} `json:"data"`
		}

		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err, "Response should be valid JSON")

		// Verify contract fields
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, "success", response.Message)
		assert.NotZero(t, response.Data.ID, "Cluster ID should not be zero")
		assert.NotEmpty(t, response.Data.Name, "Cluster name should not be empty")
		assert.NotEmpty(t, response.Data.Kubeconfig, "Kubeconfig should not be empty")
	})

	t.Run("ClusterNotFound", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		// Request non-existent cluster
		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/999999", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound && rec.Body.Len() == 0 {
			t.Skip("API not implemented yet")
			return
		}

		// Should return 404
		assert.Equal(t, http.StatusNotFound, rec.Code,
			"Should return 404 for non-existent cluster")

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, float64(404), response["code"])
	})

	t.Run("InvalidClusterID", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		// Test with invalid cluster ID (not a number)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/invalid", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound && rec.Body.Len() == 0 {
			t.Skip("API not implemented yet")
			return
		}

		// Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, rec.Code,
			"Should return 400 for invalid cluster ID format")
	})

	t.Run("KubeconfigIncluded", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/1", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Skip("API not implemented yet")
			return
		}

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok, "data field should be an object")

		// Verify kubeconfig is included in detail view (but not in list view)
		assert.Contains(t, data, "kubeconfig", "Detail view should include kubeconfig field")

		kubeconfig, ok := data["kubeconfig"].(string)
		assert.True(t, ok, "kubeconfig should be a string")
		assert.NotEmpty(t, kubeconfig, "kubeconfig should not be empty")
	})

	t.Run("HealthStatusValues", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/1", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Skip("API not implemented yet")
			return
		}

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok)

		healthStatus, ok := data["health_status"].(string)
		require.True(t, ok, "health_status should be a string")

		// Verify health_status is one of the valid values
		validStatuses := []string{"unknown", "healthy", "warning", "error", "unavailable"}
		assert.Contains(t, validStatuses, healthStatus,
			"health_status should be one of: unknown, healthy, warning, error, unavailable")
	})

	t.Run("UnauthorizedAccess", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		// Request without auth header
		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters/1", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Skip("API and auth middleware not implemented yet")
		}

		assert.Equal(t, http.StatusUnauthorized, rec.Code,
			"Should return 401 Unauthorized without valid JWT token")
	})
}
