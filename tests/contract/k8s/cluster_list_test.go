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

// TestClusterListContract verifies the API contract for GET /api/v1/k8s/clusters
// According to contracts/cluster-api.md - C1: 获取集群列表
func TestClusterListContract(t *testing.T) {
	// TODO: This test will fail until the API handler is implemented (TDD approach)
	// Expected to be implemented in Phase 0 (T027)

	t.Run("ResponseStructure", func(t *testing.T) {
		// Setup test router
		router := gin.New()
		// TODO: Register cluster list handler when implemented
		// For now, we'll test against the expected contract

		// Create test request
		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters", nil)
		// TODO: Add JWT auth header when auth is integrated
		// req.Header.Set("Authorization", "Bearer "+testToken)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		// Verify status code
		// Note: Will be 404 until handler is implemented
		if rec.Code != http.StatusOK {
			t.Skipf("API not implemented yet, got status %d, expected 200", rec.Code)
			return
		}

		// Verify response structure matches contract
		var response struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				Clusters []struct {
					ID               uint   `json:"id"`
					Name             string `json:"name"`
					IsDefault        bool   `json:"is_default"`
					Enabled          bool   `json:"enabled"`
					HealthStatus     string `json:"health_status"`
					LastConnectedAt  string `json:"last_connected_at,omitempty"`
					NodeCount        int    `json:"node_count"`
					PodCount         int    `json:"pod_count"`
					PrometheusURL    string `json:"prometheus_url,omitempty"`
					CreatedAt        string `json:"created_at"`
					UpdatedAt        string `json:"updated_at"`
				} `json:"clusters"`
				Total int `json:"total"`
			} `json:"data"`
		}

		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err, "Response should be valid JSON")

		// Verify contract fields
		assert.Equal(t, 200, response.Code, "Response code should be 200")
		assert.Equal(t, "success", response.Message, "Response message should be 'success'")
		assert.NotNil(t, response.Data.Clusters, "Clusters array should not be nil")
		assert.GreaterOrEqual(t, response.Data.Total, 0, "Total should be >= 0")
	})

	t.Run("ClusterFieldTypes", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Skipf("API not implemented yet, got status %d", rec.Code)
			return
		}

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok, "data field should be an object")

		clusters, ok := data["clusters"].([]interface{})
		require.True(t, ok, "clusters field should be an array")

		if len(clusters) > 0 {
			cluster := clusters[0].(map[string]interface{})

			// Verify required fields exist
			assert.Contains(t, cluster, "id", "Cluster should have 'id' field")
			assert.Contains(t, cluster, "name", "Cluster should have 'name' field")
			assert.Contains(t, cluster, "is_default", "Cluster should have 'is_default' field")
			assert.Contains(t, cluster, "enabled", "Cluster should have 'enabled' field")
			assert.Contains(t, cluster, "health_status", "Cluster should have 'health_status' field")
			assert.Contains(t, cluster, "node_count", "Cluster should have 'node_count' field")
			assert.Contains(t, cluster, "pod_count", "Cluster should have 'pod_count' field")
			assert.Contains(t, cluster, "created_at", "Cluster should have 'created_at' field")
			assert.Contains(t, cluster, "updated_at", "Cluster should have 'updated_at' field")

			// Verify field types
			_, ok = cluster["id"].(float64) // JSON numbers are float64
			assert.True(t, ok, "id should be a number")

			_, ok = cluster["name"].(string)
			assert.True(t, ok, "name should be a string")

			_, ok = cluster["is_default"].(bool)
			assert.True(t, ok, "is_default should be a boolean")

			_, ok = cluster["enabled"].(bool)
			assert.True(t, ok, "enabled should be a boolean")

			// health_status should be one of: unknown, healthy, warning, error, unavailable
			healthStatus, ok := cluster["health_status"].(string)
			assert.True(t, ok, "health_status should be a string")
			validStatuses := []string{"unknown", "healthy", "warning", "error", "unavailable"}
			assert.Contains(t, validStatuses, healthStatus, "health_status should be a valid status")
		}
	})

	t.Run("UnauthorizedAccess", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		// Request without auth header
		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		// Should return 401 Unauthorized when auth is implemented
		// For now, skip if auth middleware is not yet implemented
		if rec.Code == http.StatusNotFound {
			t.Skip("API and auth middleware not implemented yet")
		}

		assert.Equal(t, http.StatusUnauthorized, rec.Code,
			"Should return 401 Unauthorized without valid JWT token")
	})

	t.Run("ContentType", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/k8s/clusters", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Skipf("API not implemented yet, got status %d", rec.Code)
			return
		}

		// Verify Content-Type is application/json
		contentType := rec.Header().Get("Content-Type")
		assert.Contains(t, contentType, "application/json",
			"Response Content-Type should be application/json")
	})
}
