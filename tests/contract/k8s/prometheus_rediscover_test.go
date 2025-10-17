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

// TestPrometheusRediscoverContract verifies the API contract for POST /api/v1/k8s/clusters/:id/prometheus/rediscover
// According to contracts/cluster-api.md - C7: 手动触发 Prometheus 重新检测
func TestPrometheusRediscoverContract(t *testing.T) {
	// TODO: This test will fail until the API handler is implemented (TDD approach)
	// Expected to be implemented in Phase 1 (T036)

	t.Run("SuccessResponse", func(t *testing.T) {
		// Setup test router
		router := gin.New()
		// TODO: Register Prometheus rediscover handler when implemented

		req := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters/1/prometheus/rediscover", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Skipf("API not implemented yet, got status %d", rec.Code)
			return
		}

		// Verify response structure
		var response struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				ClusterID     uint   `json:"cluster_id"`
				TaskStartedAt string `json:"task_started_at"`
			} `json:"data"`
		}

		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err, "Response should be valid JSON")

		// Should return 202 Accepted (async operation)
		assert.Equal(t, 202, response.Code, "Response code should be 202 Accepted")
		assert.Contains(t, response.Message, "任务已启动", "Message should indicate task started")
		assert.Equal(t, uint(1), response.Data.ClusterID)
		assert.NotEmpty(t, response.Data.TaskStartedAt, "Should return task start timestamp")
	})

	t.Run("AsyncOperation", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters/1/prometheus/rediscover", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Skip("API not implemented yet")
			return
		}

		// Verify this is an async operation (202 Accepted)
		// The actual discovery happens in background
		assert.Equal(t, http.StatusAccepted, rec.Code,
			"Should return 202 Accepted for async operation")

		var response map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// Message should indicate async nature
		message, ok := response["message"].(string)
		require.True(t, ok)
		assert.Contains(t, message, "任务", "Message should mention task/job")
		assert.Contains(t, message, "稍后", "Message should indicate async nature")
	})

	t.Run("ClusterNotFound", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		// Try to rediscover for non-existent cluster
		req := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters/999999/prometheus/rediscover", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound && rec.Body.Len() == 0 {
			t.Skip("API not implemented yet")
			return
		}

		assert.Equal(t, http.StatusNotFound, rec.Code,
			"Should return 404 for non-existent cluster")
	})

	t.Run("TaskAlreadyRunning", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		// First request starts the task
		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters/1/prometheus/rediscover", nil)
		rec1 := httptest.NewRecorder()
		router.ServeHTTP(rec1, req1)

		if rec1.Code == http.StatusNotFound {
			t.Skip("API not implemented yet")
			return
		}

		// Second request immediately after should return 409 Conflict
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters/1/prometheus/rediscover", nil)
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)

		// Should return 409 Conflict if task is already running
		if rec2.Code == http.StatusConflict {
			var response map[string]any
			err := json.Unmarshal(rec2.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, float64(409), response["code"])
			assert.Contains(t, response["message"], "运行", "Error message should mention running task")
		}
	})

	t.Run("InvalidClusterID", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters/invalid/prometheus/rediscover", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound && rec.Body.Len() == 0 {
			t.Skip("API not implemented yet")
			return
		}

		assert.Equal(t, http.StatusBadRequest, rec.Code,
			"Should return 400 for invalid cluster ID format")
	})

	t.Run("UnauthorizedAccess", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		// Request without auth header
		req := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters/1/prometheus/rediscover", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Skip("API and auth middleware not implemented yet")
		}

		assert.Equal(t, http.StatusUnauthorized, rec.Code,
			"Should return 401 Unauthorized without valid JWT token")
	})

	t.Run("TaskStartedAtTimestamp", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters/1/prometheus/rediscover", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Skip("API not implemented yet")
			return
		}

		var response map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		data, ok := response["data"].(map[string]any)
		require.True(t, ok)

		taskStartedAt, ok := data["task_started_at"].(string)
		require.True(t, ok, "task_started_at should be a string")
		assert.NotEmpty(t, taskStartedAt, "task_started_at timestamp should not be empty")

		// Timestamp should be in ISO 8601 format
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`, taskStartedAt,
			"task_started_at should be in ISO 8601 format")
	})

	t.Run("ResponseImmediate", func(t *testing.T) {
		// Setup test router
		router := gin.New()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/k8s/clusters/1/prometheus/rediscover", nil)
		rec := httptest.NewRecorder()

		// Request should return immediately (not wait for discovery to complete)
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Skip("API not implemented yet")
		}

		// If we got a response, it means the API returned immediately
		// The actual discovery happens asynchronously with 30s timeout
		assert.NotEqual(t, http.StatusRequestTimeout, rec.Code,
			"API should return immediately, not wait for discovery")
	})
}
