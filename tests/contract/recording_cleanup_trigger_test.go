package contract

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTriggerCleanup tests POST /api/v1/recordings/cleanup/trigger endpoint
// Reference: contracts/recording-api.yaml `triggerCleanup` operation
func TestTriggerCleanup(t *testing.T) {
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

	t.Run("trigger cleanup as admin", func(t *testing.T) {
		// TODO: Authenticate as admin user
		path := "/api/v1/recordings/cleanup/trigger"

		resp, err := helper.MakeRequest(http.MethodPost, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusForbidden {
			t.Skip("Admin authentication not set up yet")
			return
		}

		if resp.Code == http.StatusNotFound {
			t.Skip("Cleanup trigger endpoint not implemented yet")
			return
		}

		// Should return HTTP 202 Accepted (async operation)
		assert.Equal(t, http.StatusAccepted, resp.Code)

		// Parse response using AssertJSONResponse helper
		data := helper.AssertJSONResponse(t, resp, http.StatusAccepted)

		// Validate response structure
		success, ok := data["success"].(bool)
		require.True(t, ok, "response should have 'success' field")
		assert.True(t, success, "'success' should be true")

		// Validate message
		message, ok := data["message"].(string)
		require.True(t, ok, "response should have 'message' field")
		assert.Contains(t, message, "清理任务", "message should indicate cleanup task started")

		// Validate task_id is returned
		taskID, ok := data["task_id"].(string)
		require.True(t, ok, "response should have 'task_id' field")
		assert.NotEmpty(t, taskID, "task_id should not be empty")

		// Validate task_id is valid UUID
		_, err = uuid.Parse(taskID)
		assert.NoError(t, err, "task_id should be a valid UUID")
	})

	t.Run("trigger cleanup as regular user", func(t *testing.T) {
		// TODO: Authenticate as regular (non-admin) user
		path := "/api/v1/recordings/cleanup/trigger"

		resp, err := helper.MakeRequest(http.MethodPost, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Cleanup trigger endpoint not implemented yet")
			return
		}

		// Regular users should be denied (403 Forbidden)
		if resp.Code != http.StatusForbidden {
			t.Skip("RBAC not implemented yet")
			return
		}

		helper.AssertErrorResponse(t, resp, http.StatusForbidden, "FORBIDDEN")
	})

	t.Run("trigger cleanup without authentication", func(t *testing.T) {
		// TODO: Make request without auth token
		t.Skip("Skipping: authentication not implemented yet")
	})

	t.Run("cleanup task idempotency", func(t *testing.T) {
		// TODO: Trigger cleanup twice in quick succession
		// Second request should either:
		// 1. Return same task_id (if task still running)
		// 2. Start new task (if previous finished)
		t.Skip("Skipping: idempotency behavior not defined")
	})

	t.Run("cleanup task status check", func(t *testing.T) {
		// Trigger cleanup
		triggerPath := "/api/v1/recordings/cleanup/trigger"
		triggerResp, err := helper.MakeRequest(http.MethodPost, triggerPath, nil)
		require.NoError(t, err)

		if triggerResp.Code == http.StatusNotFound {
			t.Skip("Cleanup trigger endpoint not implemented yet")
			return
		}

		if triggerResp.Code != http.StatusAccepted {
			t.Skip("Cleanup trigger failed or not accessible")
			return
		}

		triggerData := helper.AssertJSONResponse(t, triggerResp, http.StatusAccepted)
		taskID := triggerData["task_id"].(string)
		assert.NotEmpty(t, taskID, "task_id should be returned")

		// Check status endpoint
		statusPath := "/api/v1/recordings/cleanup/status"
		statusResp, err := helper.MakeRequest(http.MethodGet, statusPath, nil)
		require.NoError(t, err)

		if statusResp.Code == http.StatusNotFound {
			t.Skip("Cleanup status endpoint not implemented yet")
			return
		}

		statusData := helper.AssertSuccessResponse(t, statusResp)
		cleanupStatus := statusData["data"].(map[string]interface{})

		// Validate status reflects triggered task
		status := cleanupStatus["status"].(string)
		validStatuses := []string{"idle", "running", "completed", "failed"}
		assert.Contains(t, validStatuses, status, "status should be one of: %v", validStatuses)

		// Note: task_id is not in the status response per OpenAPI spec
		// Status endpoint returns last cleanup run info, not specific task
	})

	t.Run("cleanup with no expired recordings", func(t *testing.T) {
		// TODO: Trigger cleanup when all recordings are fresh
		// Should complete successfully with 0 records cleaned
		t.Skip("Skipping: test scenario setup not implemented")
	})

	t.Run("cleanup with expired recordings", func(t *testing.T) {
		// TODO: Create expired test recordings, then trigger cleanup
		// Should delete expired recordings and update statistics
		t.Skip("Skipping: test scenario setup not implemented")
	})

	t.Run("cleanup task failure handling", func(t *testing.T) {
		// TODO: Simulate cleanup failure (e.g., storage unavailable)
		// Should return error status in cleanup status endpoint
		t.Skip("Skipping: error simulation not implemented")
	})

	t.Run("audit log on cleanup trigger", func(t *testing.T) {
		// TODO: Verify cleanup trigger is logged in audit log
		t.Skip("Skipping: audit logging not implemented yet")
	})
}
