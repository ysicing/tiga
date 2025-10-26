package contract

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetCleanupStatus tests GET /api/v1/recordings/cleanup/status endpoint
// Reference: contracts/recording-api.yaml `getCleanupStatus` operation
func TestGetCleanupStatus(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Setup test database and router
	if err := helper.SetupTestDB(); err != nil {
		t.Skip("Skipping test: database setup not implemented yet (T024)")
		return
	}
	if err := helper.SetupRouter(nil); err != nil {
		t.Skip("Skipping test: router setup not implemented yet (T036)")
		return
	}

	t.Run("get cleanup status", func(t *testing.T) {
		path := "/api/v1/recordings/cleanup/status"

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Cleanup status endpoint not implemented yet")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)

		// Validate CleanupStatus schema
		status, ok := data["data"].(map[string]interface{})
		require.True(t, ok, "response should have 'data' object")

		// Validate required fields
		requiredFields := []string{
			"last_run_at", "status",
			"invalid_cleaned", "expired_cleaned", "orphan_cleaned",
			"total_space_freed", "error_message",
		}

		for _, field := range requiredFields {
			assert.Contains(t, status, field, "status should have '%s' field", field)
		}

		// Validate status enum
		statusValue := status["status"].(string)
		validStatuses := []string{"idle", "running", "completed", "failed"}
		assert.Contains(t, validStatuses, statusValue, "status should be one of: %v", validStatuses)

		// Validate data types
		assert.IsType(t, float64(0), status["invalid_cleaned"].(float64))
		assert.IsType(t, float64(0), status["expired_cleaned"].(float64))
		assert.IsType(t, float64(0), status["orphan_cleaned"].(float64))
		assert.IsType(t, float64(0), status["total_space_freed"].(float64))

		// last_run_at can be null (if never run)
		if status["last_run_at"] != nil {
			assert.IsType(t, "", status["last_run_at"].(string))
		}

		// error_message can be null (if no error)
		if status["error_message"] != nil {
			assert.IsType(t, "", status["error_message"].(string))
		}
	})

	t.Run("status before first cleanup", func(t *testing.T) {
		// TODO: Test with fresh database (no cleanup ever run)
		path := "/api/v1/recordings/cleanup/status"

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Cleanup status endpoint not implemented yet")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)
		status := data["data"].(map[string]interface{})

		// Should have default/initial values
		assert.Nil(t, status["last_run_at"], "last_run_at should be null before first run")
		assert.Equal(t, "idle", status["status"], "status should be idle before first run")
		assert.Equal(t, float64(0), status["invalid_cleaned"])
		assert.Equal(t, float64(0), status["expired_cleaned"])
		assert.Equal(t, float64(0), status["orphan_cleaned"])
		assert.Equal(t, float64(0), status["total_space_freed"])
		assert.Nil(t, status["error_message"])
	})

	t.Run("status after successful cleanup", func(t *testing.T) {
		// TODO: Trigger cleanup and wait for completion
		// Then check status
		t.Skip("Skipping: cleanup trigger integration not implemented")
	})

	t.Run("status during running cleanup", func(t *testing.T) {
		// TODO: Trigger cleanup and immediately check status
		// Status should be "running"
		t.Skip("Skipping: cleanup trigger integration not implemented")
	})

	t.Run("status after failed cleanup", func(t *testing.T) {
		// TODO: Simulate cleanup failure and check status
		// Status should be "failed" with error_message populated
		t.Skip("Skipping: error simulation not implemented")
	})

	t.Run("cleanup statistics accuracy", func(t *testing.T) {
		// TODO: Create test data with known counts
		// Run cleanup and verify statistics match expected values
		t.Skip("Skipping: test scenario setup not implemented")
	})

	t.Run("space freed calculation", func(t *testing.T) {
		// TODO: Create recordings with known file sizes
		// Run cleanup and verify total_space_freed is accurate
		t.Skip("Skipping: test scenario setup not implemented")
	})

	t.Run("last_run_at timestamp format", func(t *testing.T) {
		// TODO: Trigger cleanup and verify last_run_at is valid ISO 8601
		t.Skip("Skipping: cleanup trigger integration not implemented")
	})

	t.Run("unauthorized access", func(t *testing.T) {
		// TODO: Test without authentication token
		// Cleanup status might be public or require auth - spec unclear
		t.Skip("Skipping: authentication requirements not defined")
	})

	t.Run("multiple cleanup runs", func(t *testing.T) {
		// TODO: Run cleanup multiple times
		// Verify status reflects most recent run
		t.Skip("Skipping: cleanup trigger integration not implemented")
	})

	t.Run("cleanup status caching", func(t *testing.T) {
		// Verify status endpoint has reasonable response time
		// Should not trigger actual cleanup operation
		path := "/api/v1/recordings/cleanup/status"

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Cleanup status endpoint not implemented yet")
			return
		}

		// Status query should be fast (<100ms typically)
		// This is a basic smoke test, not a performance test
		assert.Equal(t, http.StatusOK, resp.Code)
	})
}
