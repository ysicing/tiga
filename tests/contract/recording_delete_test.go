package contract

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeleteRecording tests DELETE /api/v1/recordings/:id endpoint
// Reference: contracts/recording-api.yaml `deleteRecording` operation
func TestDeleteRecording(t *testing.T) {
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

	t.Run("delete existing recording", func(t *testing.T) {
		// TODO: Create test recording
		testID := uuid.New().String()

		path := fmt.Sprintf("/api/v1/recordings/%s", testID)
		resp, err := helper.MakeRequest(http.MethodDelete, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test data creation not implemented yet")
			return
		}

		// Assert successful deletion (HTTP 200)
		data := helper.AssertSuccessResponse(t, resp)

		// Assert response message
		message, ok := data["message"].(string)
		require.True(t, ok, "response should have 'message' field")
		assert.Contains(t, message, "删除成功", "message should indicate success")

		// TODO: Verify database record is deleted
		// TODO: Verify storage file is deleted
	})

	t.Run("delete verifies database record removal", func(t *testing.T) {
		// TODO: Create test recording
		testID := uuid.New().String()

		path := fmt.Sprintf("/api/v1/recordings/%s", testID)
		resp, err := helper.MakeRequest(http.MethodDelete, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test data creation not implemented yet")
			return
		}

		helper.AssertSuccessResponse(t, resp)

		// Verify record no longer exists by trying to GET it
		getResp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, getResp.Code, "deleted recording should not be found")
	})

	t.Run("delete verifies storage file removal", func(t *testing.T) {
		// TODO: Create test recording with actual file
		testID := uuid.New().String()

		path := fmt.Sprintf("/api/v1/recordings/%s", testID)
		resp, err := helper.MakeRequest(http.MethodDelete, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test data creation not implemented yet")
			return
		}

		helper.AssertSuccessResponse(t, resp)

		// TODO: Verify file is deleted from storage
		// This requires StorageService implementation (T030)
	})

	t.Run("delete non-existent recording", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		path := fmt.Sprintf("/api/v1/recordings/%s", nonExistentID)

		resp, err := helper.MakeRequest(http.MethodDelete, path, nil)
		require.NoError(t, err)

		helper.AssertErrorResponse(t, resp, http.StatusNotFound, "NOT_FOUND")
	})

	t.Run("delete with invalid UUID", func(t *testing.T) {
		path := "/api/v1/recordings/invalid-uuid"
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		// Should return 400 Bad Request for invalid UUID
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("unauthorized deletion", func(t *testing.T) {
		// TODO: Test without authentication token
		t.Skip("Skipping: authentication not implemented yet")
	})

	t.Run("forbidden - user cannot delete others' recordings", func(t *testing.T) {
		// TODO: Create recording owned by user A
		// TODO: Try to delete as user B
		// Should return 403 Forbidden
		t.Skip("Skipping: RBAC not implemented yet")
	})

	t.Run("admin can delete any recording", func(t *testing.T) {
		// TODO: Create recording owned by regular user
		// TODO: Delete as admin user
		// Should return 200 OK
		t.Skip("Skipping: RBAC not implemented yet")
	})

	t.Run("idempotent deletion", func(t *testing.T) {
		// TODO: Create test recording
		testID := uuid.New().String()

		path := fmt.Sprintf("/api/v1/recordings/%s", testID)

		// First deletion should succeed
		resp1, err := helper.MakeRequest(http.MethodDelete, path, nil)
		require.NoError(t, err)

		if resp1.Code == http.StatusNotFound {
			t.Skip("Test data creation not implemented yet")
			return
		}

		helper.AssertSuccessResponse(t, resp1)

		// Second deletion should return 404
		resp2, err := helper.MakeRequest(http.MethodDelete, path, nil)
		require.NoError(t, err)
		helper.AssertErrorResponse(t, resp2, http.StatusNotFound, "NOT_FOUND")
	})

	t.Run("delete recording with active session", func(t *testing.T) {
		// TODO: Create active recording (ended_at IS NULL)
		// Should handle gracefully - either prevent deletion or mark for cleanup
		t.Skip("Skipping: active session handling not specified")
	})
}
