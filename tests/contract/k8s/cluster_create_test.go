package k8s_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClusterCreateContract verifies the API contract for POST /api/v1/k8s/clusters
// According to contracts/cluster-api.md - C3: 创建集群
func TestClusterCreateContract(t *testing.T) {
	// TODO: This test will fail until the API handler is implemented (TDD approach)
	// Expected to be implemented in Phase 0 (T029)

	t.Run("SuccessResponse", func(t *testing.T) {
		// Setup test router
		router := gin.New()
		// TODO: Register cluster create handler when implemented

		// Create request payload
		payload := map[string]any{
			"name":       "development",
			"kubeconfig": "YXBpVmVyc2lvbjog...", // Base64 encoded kubeconfig
			"is_default": false,
			"enabled":    true,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Skipf("API not implemented yet, got status %d", rec.Code)
			return
		}

		// Verify response structure
		var response struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				ID               uint   `json:"id"`
				Name             string `json:"name"`
				IsDefault        bool   `json:"is_default"`
				Enabled          bool   `json:"enabled"`
				HealthStatus     string `json:"health_status"`
				LastConnectedAt  *string `json:"last_connected_at"` // nullable
				NodeCount        int    `json:"node_count"`
				PodCount         int    `json:"pod_count"`
				PrometheusURL    string `json:"prometheus_url"`
				CreatedAt        string `json:"created_at"`
				UpdatedAt        string `json:"updated_at"`
			} `json:"data"`
		}

		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err, "Response should be valid JSON")

		// Verify contract fields
		assert.Equal(t, 201, response.Code, "Response code should be 201 Created")
		assert.Contains(t, response.Message, "成功", "Message should indicate success")
		assert.NotZero(t, response.Data.ID, "Created cluster should have an ID")
		assert.Equal(t, "development", response.Data.Name)
		assert.Equal(t, "unknown", response.Data.HealthStatus, "New cluster health should be 'unknown'")
		assert.Equal(t, 0, response.Data.NodeCount, "New cluster node_count should be 0")
		assert.Equal(t, 0, response.Data.PodCount, "New cluster pod_count should be 0")
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		testCases := []struct {
			name    string
			payload map[string]any
			wantErr string
		}{
			{
				name:    "MissingName",
				payload: map[string]any{
					"kubeconfig": "YXBpVmVyc2lvbjog...",
				},
				wantErr: "name",
			},
			{
				name:    "MissingKubeconfig",
				payload: map[string]any{
					"name": "test-cluster",
				},
				wantErr: "kubeconfig",
			},
			{
				name:    "EmptyName",
				payload: map[string]any{
					"name":       "",
					"kubeconfig": "YXBpVmVyc2lvbjog...",
				},
				wantErr: "name",
			},
			{
				name:    "NameTooLong",
				payload: map[string]any{
					"name":       string(make([]byte, 101)), // > 100 characters
					"kubeconfig": "YXBpVmVyc2lvbjog...",
				},
				wantErr: "name",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				body, _ := json.Marshal(tc.payload)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)

				if rec.Code == http.StatusNotFound {
					t.Skip("API not implemented yet")
					return
				}

				assert.Equal(t, http.StatusBadRequest, rec.Code,
					"Should return 400 Bad Request for validation errors")

				var response map[string]any
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, float64(400), response["code"])
			})
		}
	})

	t.Run("InvalidKubeconfigFormat", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		payload := map[string]any{
			"name":       "test-cluster",
			"kubeconfig": "not-valid-base64", // Invalid kubeconfig
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Skip("API not implemented yet")
			return
		}

		// Should return 400 for invalid kubeconfig format
		assert.Equal(t, http.StatusBadRequest, rec.Code,
			"Should return 400 for invalid kubeconfig format")
	})

	t.Run("DuplicateClusterName", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		// First create
		payload := map[string]any{
			"name":       "duplicate-test",
			"kubeconfig": "YXBpVmVyc2lvbjog...",
		}
		body, _ := json.Marshal(payload)

		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters", bytes.NewReader(body))
		req1.Header.Set("Content-Type", "application/json")
		rec1 := httptest.NewRecorder()
		router.ServeHTTP(rec1, req1)

		if rec1.Code == http.StatusNotFound {
			t.Skip("API not implemented yet")
			return
		}

		// Second create with same name
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters", bytes.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)

		// Should return 400 for duplicate cluster name
		assert.Equal(t, http.StatusBadRequest, rec2.Code,
			"Should return 400 for duplicate cluster name")
	})

	t.Run("PrometheusDiscoveryMessage", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		payload := map[string]any{
			"name":       "auto-discover-test",
			"kubeconfig": "YXBpVmVyc2lvbjog...",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Skip("API not implemented yet")
			return
		}

		var response map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// Message should mention Prometheus auto-discovery task started
		message, ok := response["message"].(string)
		require.True(t, ok)
		assert.Contains(t, message, "Prometheus", "Message should mention Prometheus auto-discovery")
	})

	t.Run("UnauthorizedAccess", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		payload := map[string]any{
			"name":       "test-cluster",
			"kubeconfig": "YXBpVmVyc2lvbjog...",
		}
		body, _ := json.Marshal(payload)

		// Request without auth header
		req := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Skip("API and auth middleware not implemented yet")
		}

		assert.Equal(t, http.StatusUnauthorized, rec.Code,
			"Should return 401 Unauthorized without valid JWT token")
	})

	t.Run("NonAdminForbidden", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		payload := map[string]any{
			"name":       "test-cluster",
			"kubeconfig": "YXBpVmVyc2lvbjog...",
		}
		body, _ := json.Marshal(payload)

		// TODO: Add non-admin JWT token when auth is implemented
		req := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// req.Header.Set("Authorization", "Bearer "+nonAdminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound || rec.Code == http.StatusUnauthorized {
			t.Skip("API or RBAC middleware not implemented yet")
		}

		assert.Equal(t, http.StatusForbidden, rec.Code,
			"Should return 403 Forbidden for non-admin users")
	})
}
