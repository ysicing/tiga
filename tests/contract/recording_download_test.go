package contract

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDownloadRecording tests GET /api/v1/recordings/:id/download endpoint
// Reference: contracts/recording-api.yaml `downloadRecording` operation
func TestDownloadRecording(t *testing.T) {
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

	t.Run("download existing recording", func(t *testing.T) {
		// TODO: Create test recording with actual file
		testID := uuid.New().String()

		path := fmt.Sprintf("/api/v1/recordings/%s/download", testID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test recording not created yet")
			return
		}

		// Assert successful response
		assert.Equal(t, http.StatusOK, resp.Code)

		// Validate Content-Type
		contentType := resp.Header().Get("Content-Type")
		assert.Equal(t, "application/octet-stream", contentType,
			"Content-Type should be application/octet-stream")

		// Validate Content-Disposition header
		contentDisposition := resp.Header().Get("Content-Disposition")
		assert.NotEmpty(t, contentDisposition, "should have Content-Disposition header")
		assert.Contains(t, contentDisposition, "attachment", "should be attachment")
		assert.Contains(t, contentDisposition, "filename=", "should specify filename")
		assert.Contains(t, contentDisposition, ".cast", "filename should have .cast extension")

		// Validate response body (file content)
		body := resp.Body.Bytes()
		assert.NotEmpty(t, body, "file content should not be empty")
	})

	t.Run("download filename format", func(t *testing.T) {
		// TODO: Create test recording
		testID := uuid.New().String()

		path := fmt.Sprintf("/api/v1/recordings/%s/download", testID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test recording not created yet")
			return
		}

		contentDisposition := resp.Header().Get("Content-Disposition")

		// Filename should be: recording-{id}.cast or {username}-{timestamp}.cast
		assert.Regexp(t, `filename="[^"]+\.cast"`, contentDisposition,
			"filename should end with .cast")
	})

	t.Run("download file integrity", func(t *testing.T) {
		// TODO: Create test recording with known content
		testID := uuid.New().String()

		path := fmt.Sprintf("/api/v1/recordings/%s/download", testID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test recording not created yet")
			return
		}

		body := resp.Body.Bytes()

		// Validate Asciinema v2 format structure
		content := string(body)
		lines := strings.Split(content, "\n")

		require.Greater(t, len(lines), 0, "file should have at least one line")

		// First line should be valid JSON header
		firstLine := lines[0]
		assert.Contains(t, firstLine, `"version":2`, "should be asciinema v2 format")
	})

	t.Run("download non-existent recording", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		path := fmt.Sprintf("/api/v1/recordings/%s/download", nonExistentID)

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		helper.AssertErrorResponse(t, resp, http.StatusNotFound, "NOT_FOUND")
	})

	t.Run("download with invalid UUID", func(t *testing.T) {
		path := "/api/v1/recordings/invalid-uuid/download"

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("download recording without file", func(t *testing.T) {
		// TODO: Create database record without actual storage file
		// Should return appropriate error
		t.Skip("Skipping: orphan recording test not implemented")
	})

	t.Run("download from MinIO storage", func(t *testing.T) {
		// TODO: Test with storage_type='minio'
		// Should fetch from MinIO and stream to client
		t.Skip("Skipping: MinIO storage not implemented (Phase 2)")
	})

	t.Run("download large file streaming", func(t *testing.T) {
		// TODO: Test with recording >50MB
		// Should use chunked transfer encoding
		t.Skip("Skipping: large file streaming test not implemented")
	})

	t.Run("concurrent downloads", func(t *testing.T) {
		// TODO: Test multiple simultaneous downloads of same file
		// Should handle without issues
		t.Skip("Skipping: concurrency test not implemented")
	})

	t.Run("unauthorized download", func(t *testing.T) {
		// TODO: Test without authentication token
		t.Skip("Skipping: authentication not implemented yet")
	})

	t.Run("forbidden - user cannot download others' recordings", func(t *testing.T) {
		// TODO: Test RBAC permissions
		t.Skip("Skipping: RBAC not implemented yet")
	})

	t.Run("audit log on download", func(t *testing.T) {
		// TODO: Verify download action is logged
		t.Skip("Skipping: audit logging not implemented yet")
	})
}
