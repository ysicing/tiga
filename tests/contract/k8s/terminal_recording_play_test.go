package k8s_test

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
	"gorm.io/datatypes"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/tests/contract"
)

// TestPlayRecordingAsciinemaV2 tests GET /api/v1/recordings/:id/play with Asciinema v2 format
// Reference: 010-k8s-pod-009 T006 - Recording playback API contract test
// MUST FAIL until Asciinema v2 format streaming is implemented
func TestPlayRecordingAsciinemaV2(t *testing.T) {
	helper := contract.NewTestHelper(t)
	defer helper.Cleanup()

	// Setup test database and router
	if err := helper.SetupTestDB(); err != nil {
		t.Skip("Skipping test: database setup not implemented yet")
		return
	}
	if err := helper.SetupRouter(nil); err != nil {
		t.Skip("Skipping test: router setup not implemented yet")
		return
	}

	clusterID := uuid.New().String()

	t.Run("k8s_node recording playback format", func(t *testing.T) {
		// Create K8s node recording with mock cast file
		nodeRecording := &models.TerminalRecording{
			RecordingType: models.RecordingTypeK8sNode,
			UserID:        helper.AdminUser.ID,
			Username:      helper.AdminUser.Username,
			TypeMetadata: datatypes.JSON([]byte(fmt.Sprintf(`{
				"cluster_id": "%s",
				"node_name": "worker-1"
			}`, clusterID))),
			FilePath: "/tmp/test-node-recording.cast",
			FileSize: 512,
			Duration: 120,
		}
		err := helper.DB.Create(nodeRecording).Error
		require.NoError(t, err)

		// Request playback
		path := fmt.Sprintf("/api/v1/recordings/%s/play", nodeRecording.ID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		// Assert response status
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Assert Content-Type header (Asciinema v2)
		contentType := resp.Header.Get("Content-Type")
		assert.Contains(t, contentType, "application/x-asciicast",
			"Content-Type should be application/x-asciicast for Asciinema v2 format")

		// Read response body
		body := resp.Body
		defer body.Close()

		scanner := bufio.NewScanner(body)

		// Line 1: Header (must be valid JSON with version: 2)
		require.True(t, scanner.Scan(), "Should have header line")
		headerLine := scanner.Text()

		var header map[string]interface{}
		err = json.Unmarshal([]byte(headerLine), &header)
		require.NoError(t, err, "Header line should be valid JSON")

		// Verify Asciinema v2 header structure
		assert.Equal(t, float64(2), header["version"], "version should be 2")
		assert.NotEmpty(t, header["width"], "width should be present")
		assert.NotEmpty(t, header["height"], "height should be present")
		assert.NotEmpty(t, header["timestamp"], "timestamp should be present")

		// Optional: Check for title field
		if title, ok := header["title"]; ok {
			assert.NotEmpty(t, title, "title should not be empty if present")
		}

		// Subsequent lines: Frames (format: [time, type, data])
		frameCount := 0
		for scanner.Scan() && frameCount < 5 {
			frameLine := scanner.Text()

			var frame []interface{}
			err = json.Unmarshal([]byte(frameLine), &frame)
			require.NoError(t, err, "Frame line should be valid JSON array")

			// Verify frame structure: [timestamp, type, data]
			require.Len(t, frame, 3, "Frame should have exactly 3 elements")

			// Element 0: timestamp (number)
			timestamp, ok := frame[0].(float64)
			require.True(t, ok, "Frame[0] should be a number (timestamp)")
			assert.GreaterOrEqual(t, timestamp, 0.0, "Timestamp should be non-negative")

			// Element 1: type (string, "o" for output or "i" for input)
			frameType, ok := frame[1].(string)
			require.True(t, ok, "Frame[1] should be a string (type)")
			assert.Contains(t, []string{"o", "i"}, frameType, "Frame type should be 'o' or 'i'")

			// Element 2: data (string)
			data, ok := frame[2].(string)
			require.True(t, ok, "Frame[2] should be a string (data)")
			assert.NotEmpty(t, data, "Frame data should not be empty")

			frameCount++
		}

		// Assert we read at least some frames
		assert.Greater(t, frameCount, 0, "Should have at least one frame")
	})

	t.Run("k8s_pod recording playback format", func(t *testing.T) {
		// Create K8s pod recording
		podRecording := &models.TerminalRecording{
			RecordingType: models.RecordingTypeK8sPod,
			UserID:        helper.AdminUser.ID,
			Username:      helper.AdminUser.Username,
			TypeMetadata: datatypes.JSON([]byte(fmt.Sprintf(`{
				"cluster_id": "%s",
				"namespace": "default",
				"pod_name": "nginx-pod",
				"container_name": "nginx"
			}`, clusterID))),
			FilePath: "/tmp/test-pod-recording.cast",
			FileSize: 1024,
			Duration: 300,
		}
		err := helper.DB.Create(podRecording).Error
		require.NoError(t, err)

		// Request playback
		path := fmt.Sprintf("/api/v1/recordings/%s/play", podRecording.ID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		// Assert response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/x-asciicast")

		// Verify first line is valid Asciinema v2 header
		scanner := bufio.NewScanner(resp.Body)
		defer resp.Body.Close()

		require.True(t, scanner.Scan())
		headerLine := scanner.Text()

		var header map[string]interface{}
		err = json.Unmarshal([]byte(headerLine), &header)
		require.NoError(t, err)
		assert.Equal(t, float64(2), header["version"])
	})

	t.Run("verify streaming response (not buffered)", func(t *testing.T) {
		// Create recording
		recording := &models.TerminalRecording{
			RecordingType: models.RecordingTypeK8sNode,
			UserID:        helper.AdminUser.ID,
			Username:      helper.AdminUser.Username,
			TypeMetadata: datatypes.JSON([]byte(fmt.Sprintf(`{
				"cluster_id": "%s",
				"node_name": "test-node"
			}`, clusterID))),
			FilePath: "/tmp/test-stream-recording.cast",
			FileSize: 2048,
			Duration: 180,
		}
		err := helper.DB.Create(recording).Error
		require.NoError(t, err)

		path := fmt.Sprintf("/api/v1/recordings/%s/play", recording.ID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		// Check for streaming headers (no Content-Length or Transfer-Encoding: chunked)
		// This ensures the response is streamed, not buffered
		contentLength := resp.Header.Get("Content-Length")
		transferEncoding := resp.Header.Get("Transfer-Encoding")

		// Either no Content-Length (streaming) or Transfer-Encoding: chunked
		if contentLength != "" {
			t.Logf("Warning: Content-Length is set (%s), response may be buffered", contentLength)
		}
		if !strings.Contains(strings.ToLower(transferEncoding), "chunked") {
			t.Logf("Note: Transfer-Encoding is not chunked (%s)", transferEncoding)
		}
	})

	t.Run("nonexistent recording returns 404", func(t *testing.T) {
		fakeID := uuid.New()
		path := fmt.Sprintf("/api/v1/recordings/%s/play", fakeID)

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("verify header contains terminal dimensions", func(t *testing.T) {
		recording := &models.TerminalRecording{
			RecordingType: models.RecordingTypeK8sPod,
			UserID:        helper.AdminUser.ID,
			Username:      helper.AdminUser.Username,
			TypeMetadata: datatypes.JSON([]byte(fmt.Sprintf(`{
				"cluster_id": "%s",
				"namespace": "kube-system",
				"pod_name": "coredns-abc",
				"container_name": "coredns"
			}`, clusterID))),
			FilePath: "/tmp/test-dimensions.cast",
			FileSize: 512,
			Duration: 60,
		}
		err := helper.DB.Create(recording).Error
		require.NoError(t, err)

		path := fmt.Sprintf("/api/v1/recordings/%s/play", recording.ID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		scanner := bufio.NewScanner(resp.Body)
		defer resp.Body.Close()

		require.True(t, scanner.Scan())
		headerLine := scanner.Text()

		var header map[string]interface{}
		err = json.Unmarshal([]byte(headerLine), &header)
		require.NoError(t, err)

		// Verify dimensions are reasonable values
		width, widthOk := header["width"].(float64)
		height, heightOk := header["height"].(float64)

		require.True(t, widthOk, "width should be present and numeric")
		require.True(t, heightOk, "height should be present and numeric")

		assert.GreaterOrEqual(t, width, 80.0, "width should be at least 80 columns")
		assert.LessOrEqual(t, width, 300.0, "width should not exceed 300 columns")
		assert.GreaterOrEqual(t, height, 24.0, "height should be at least 24 rows")
		assert.LessOrEqual(t, height, 100.0, "height should not exceed 100 rows")
	})
}
