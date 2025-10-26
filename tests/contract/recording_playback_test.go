package contract

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetPlaybackContent tests GET /api/v1/recordings/:id/playback endpoint
// Reference: contracts/recording-api.yaml `getPlaybackContent` operation
func TestGetPlaybackContent(t *testing.T) {
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

	t.Run("get playback content - asciinema v2 format", func(t *testing.T) {
		// TODO: Create test recording with actual .cast file
		testID := uuid.New().String()

		path := fmt.Sprintf("/api/v1/recordings/%s/playback", testID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test recording not created yet")
			return
		}

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "application/json; charset=utf-8", resp.Header().Get("Content-Type"))

		// Asciinema v2 format validation
		// First line: JSON header
		// Remaining lines: Frame arrays [timestamp, event_type, data]
		content := resp.Body.String()
		lines := strings.Split(content, "\n")

		require.Greater(t, len(lines), 0, "content should have at least one line")

		// Validate first line is JSON header
		firstLine := lines[0]
		var header map[string]interface{}
		err = json.Unmarshal([]byte(firstLine), &header)
		require.NoError(t, err, "first line should be valid JSON header")

		// Validate header fields
		requiredHeaderFields := []string{"version", "width", "height", "timestamp"}
		for _, field := range requiredHeaderFields {
			assert.Contains(t, header, field, "header should have '%s' field", field)
		}

		// Validate version is 2
		version := header["version"].(float64)
		assert.Equal(t, float64(2), version, "asciinema version should be 2")

		// Validate dimensions
		width := header["width"].(float64)
		height := header["height"].(float64)
		assert.Greater(t, width, float64(0), "width should be positive")
		assert.Greater(t, height, float64(0), "height should be positive")

		// Validate frames (remaining lines)
		if len(lines) > 1 {
			for i := 1; i < len(lines); i++ {
				line := strings.TrimSpace(lines[i])
				if line == "" {
					continue
				}

				// Each frame should be a JSON array: [timestamp, event_type, data]
				var frame []interface{}
				err := json.Unmarshal([]byte(line), &frame)
				require.NoError(t, err, "frame %d should be valid JSON array", i)

				require.Equal(t, 3, len(frame), "frame should have 3 elements")

				// Validate timestamp (number)
				timestamp, ok := frame[0].(float64)
				require.True(t, ok, "timestamp should be a number")
				assert.GreaterOrEqual(t, timestamp, float64(0), "timestamp should be non-negative")

				// Validate event type (string: "o" for output, "i" for input)
				eventType, ok := frame[1].(string)
				require.True(t, ok, "event_type should be a string")
				validEventTypes := []string{"o", "i"}
				assert.Contains(t, validEventTypes, eventType, "event_type should be 'o' or 'i'")

				// Validate data (string)
				data, ok := frame[2].(string)
				require.True(t, ok, "data should be a string")
				assert.NotNil(t, data)
			}
		}
	})

	t.Run("playback content with large recording", func(t *testing.T) {
		// TODO: Test with recording >10MB
		// Should handle streaming or chunked response
		t.Skip("Skipping: large file handling test not implemented")
	})

	t.Run("playback non-existent recording", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		path := fmt.Sprintf("/api/v1/recordings/%s/playback", nonExistentID)

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		helper.AssertErrorResponse(t, resp, http.StatusNotFound, "NOT_FOUND")
	})

	t.Run("playback with invalid UUID", func(t *testing.T) {
		path := "/api/v1/recordings/invalid-uuid/playback"

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("playback recording without file", func(t *testing.T) {
		// TODO: Create database record without actual storage file
		// Should return appropriate error (500 or 404)
		t.Skip("Skipping: orphan recording test not implemented")
	})

	t.Run("playback recording from MinIO storage", func(t *testing.T) {
		// TODO: Test with storage_type='minio'
		// Should fetch from MinIO instead of local filesystem
		t.Skip("Skipping: MinIO storage not implemented (Phase 2)")
	})

	t.Run("validate frame timestamp order", func(t *testing.T) {
		// TODO: Create test recording
		testID := uuid.New().String()

		path := fmt.Sprintf("/api/v1/recordings/%s/playback", testID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test recording not created yet")
			return
		}

		content := resp.Body.String()
		scanner := bufio.NewScanner(strings.NewReader(content))

		// Skip header line
		scanner.Scan()

		var prevTimestamp float64
		frameCount := 0

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var frame []interface{}
			err := json.Unmarshal([]byte(line), &frame)
			require.NoError(t, err)

			timestamp := frame[0].(float64)

			if frameCount > 0 {
				assert.GreaterOrEqual(t, timestamp, prevTimestamp,
					"frames should be in chronological order (timestamp %f should be >= %f)",
					timestamp, prevTimestamp)
			}

			prevTimestamp = timestamp
			frameCount++
		}

		assert.Greater(t, frameCount, 0, "should have at least one frame")
	})

	t.Run("unauthorized playback", func(t *testing.T) {
		// TODO: Test without authentication token
		t.Skip("Skipping: authentication not implemented yet")
	})

	t.Run("forbidden - user cannot view others' recordings", func(t *testing.T) {
		// TODO: Test RBAC permissions
		t.Skip("Skipping: RBAC not implemented yet")
	})
}
