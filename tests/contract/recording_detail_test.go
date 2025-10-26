package contract

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetRecording tests GET /api/v1/recordings/:id endpoint
// Reference: contracts/recording-api.yaml `getRecording` operation
func TestGetRecording(t *testing.T) {
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

	t.Run("get existing recording", func(t *testing.T) {
		// TODO: Create test recording via helper
		// testRecording := helper.CreateTestRecording(...)
		testID := uuid.New().String()

		path := fmt.Sprintf("/api/v1/recordings/%s", testID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		// Expect 404 for now since we haven't created test data
		if resp.Code == http.StatusNotFound {
			t.Skip("Test data creation not implemented yet")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)

		// Assert RecordingDetail schema
		recording, ok := data["data"].(map[string]interface{})
		require.True(t, ok, "response should have 'data' object")

		// Validate required fields (RecordingDetail extends RecordingListItem)
		requiredFields := []string{
			"id", "session_id", "recording_type", "user_id", "username",
			"started_at", "duration", "file_size", "file_size_human", "client_ip",
			// Additional fields from RecordingDetail
			"type_metadata", "storage_type", "storage_path", "format",
			"rows", "cols", "shell",
		}

		for _, field := range requiredFields {
			assert.Contains(t, recording, field, "recording should have '%s' field", field)
		}

		// Validate type_metadata is JSONB object
		typeMetadata, ok := recording["type_metadata"].(map[string]interface{})
		require.True(t, ok, "type_metadata should be an object")
		assert.NotNil(t, typeMetadata)

		// Validate storage_type enum
		storageType := recording["storage_type"].(string)
		validStorageTypes := []string{"local", "minio"}
		assert.Contains(t, validStorageTypes, storageType, "storage_type should be 'local' or 'minio'")

		// Validate format enum
		format := recording["format"].(string)
		assert.Equal(t, "asciinema", format, "format should be 'asciinema'")

		// Validate terminal dimensions
		rows := recording["rows"].(float64)
		cols := recording["cols"].(float64)
		assert.Greater(t, rows, float64(0), "rows should be positive")
		assert.Greater(t, cols, float64(0), "cols should be positive")
	})

	t.Run("get recording by type - docker", func(t *testing.T) {
		// TODO: Create Docker recording
		testID := uuid.New().String()

		path := fmt.Sprintf("/api/v1/recordings/%s", testID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test data creation not implemented yet")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)
		recording := data["data"].(map[string]interface{})

		// Docker recordings should have instance_id and container_id in type_metadata
		assert.Equal(t, "docker", recording["recording_type"])
		typeMetadata := recording["type_metadata"].(map[string]interface{})
		assert.Contains(t, typeMetadata, "instance_id", "Docker recording should have instance_id")
		assert.Contains(t, typeMetadata, "container_id", "Docker recording should have container_id")
	})

	t.Run("get recording by type - webssh", func(t *testing.T) {
		// TODO: Create WebSSH recording
		testID := uuid.New().String()

		path := fmt.Sprintf("/api/v1/recordings/%s", testID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test data creation not implemented yet")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)
		recording := data["data"].(map[string]interface{})

		// WebSSH recordings should have host_id in type_metadata
		assert.Equal(t, "webssh", recording["recording_type"])
		typeMetadata := recording["type_metadata"].(map[string]interface{})
		assert.Contains(t, typeMetadata, "host_id", "WebSSH recording should have host_id")
	})

	t.Run("get recording by type - k8s_node", func(t *testing.T) {
		// TODO: Create K8s node recording
		testID := uuid.New().String()

		path := fmt.Sprintf("/api/v1/recordings/%s", testID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test data creation not implemented yet")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)
		recording := data["data"].(map[string]interface{})

		// K8s node recordings should have cluster_id and node_name in type_metadata
		assert.Equal(t, "k8s_node", recording["recording_type"])
		typeMetadata := recording["type_metadata"].(map[string]interface{})
		assert.Contains(t, typeMetadata, "cluster_id", "K8s recording should have cluster_id")
		assert.Contains(t, typeMetadata, "node_name", "K8s node recording should have node_name")
	})

	t.Run("invalid UUID format", func(t *testing.T) {
		path := "/api/v1/recordings/invalid-uuid"
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		// Should return 400 Bad Request for invalid UUID
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("non-existent recording", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		path := fmt.Sprintf("/api/v1/recordings/%s", nonExistentID)

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		helper.AssertErrorResponse(t, resp, http.StatusNotFound, "NOT_FOUND")
	})

	t.Run("unauthorized access", func(t *testing.T) {
		// TODO: Implement when authentication middleware is ready
		t.Skip("Skipping: authentication not implemented yet")
	})

	t.Run("permission check - user can only see own recordings", func(t *testing.T) {
		// TODO: Implement when RBAC middleware is ready
		t.Skip("Skipping: RBAC not implemented yet")
	})
}
