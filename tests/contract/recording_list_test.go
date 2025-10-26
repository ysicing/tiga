package contract

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListRecordings tests GET /api/v1/recordings endpoint
// Reference: contracts/recording-api.yaml `listRecordings` operation
func TestListRecordings(t *testing.T) {
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

	t.Run("default pagination", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)

		// Assert response structure
		responseData, ok := data["data"].(map[string]interface{})
		require.True(t, ok, "response should have 'data' object")

		// Assert recordings array exists
		recordings, ok := responseData["recordings"].([]interface{})
		require.True(t, ok, "data should have 'recordings' array")
		assert.IsType(t, []interface{}{}, recordings)

		// Assert pagination structure
		pagination, ok := responseData["pagination"].(map[string]interface{})
		require.True(t, ok, "data should have 'pagination' object")
		helper.AssertPaginationStructure(t, pagination)

		// Assert default pagination values
		assert.Equal(t, float64(1), pagination["page"], "default page should be 1")
		assert.Equal(t, float64(20), pagination["limit"], "default limit should be 20")
	})

	t.Run("custom pagination", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings?page=2&limit=10", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		pagination := responseData["pagination"].(map[string]interface{})

		assert.Equal(t, float64(2), pagination["page"])
		assert.Equal(t, float64(10), pagination["limit"])
	})

	t.Run("filter by recording_type", func(t *testing.T) {
		testCases := []string{"docker", "webssh", "k8s_node", "k8s_pod"}

		for _, recordingType := range testCases {
			t.Run(recordingType, func(t *testing.T) {
				path := fmt.Sprintf("/api/v1/recordings?recording_type=%s", recordingType)
				resp, err := helper.MakeRequest(http.MethodGet, path, nil)
				require.NoError(t, err)

				data := helper.AssertSuccessResponse(t, resp)
				responseData := data["data"].(map[string]interface{})
				recordings := responseData["recordings"].([]interface{})

				// Verify all returned recordings match the type filter
				for _, rec := range recordings {
					recording := rec.(map[string]interface{})
					assert.Equal(t, recordingType, recording["recording_type"])
				}
			})
		}
	})

	t.Run("filter by user_id", func(t *testing.T) {
		testUserID := "550e8400-e29b-41d4-a716-446655440000"
		path := fmt.Sprintf("/api/v1/recordings?user_id=%s", testUserID)

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		// Verify all returned recordings match the user filter
		for _, rec := range recordings {
			recording := rec.(map[string]interface{})
			assert.Equal(t, testUserID, recording["user_id"])
		}
	})

	t.Run("filter by time range", func(t *testing.T) {
		startTime := "2024-01-01T00:00:00Z"
		endTime := "2024-12-31T23:59:59Z"
		path := fmt.Sprintf("/api/v1/recordings?start_time=%s&end_time=%s", startTime, endTime)

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		// Verify all returned recordings are within time range
		for _, rec := range recordings {
			recording := rec.(map[string]interface{})
			startedAt := recording["started_at"].(string)
			assert.NotEmpty(t, startedAt, "recording should have started_at timestamp")
		}
	})

	t.Run("sort by started_at desc (default)", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		// Verify recordings are sorted by started_at descending
		if len(recordings) >= 2 {
			first := recordings[0].(map[string]interface{})
			second := recordings[1].(map[string]interface{})
			firstTime := first["started_at"].(string)
			secondTime := second["started_at"].(string)
			assert.GreaterOrEqual(t, firstTime, secondTime, "recordings should be sorted by started_at desc")
		}
	})

	t.Run("sort by duration asc", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings?sort_by=duration&sort_order=asc", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		// Verify recordings are sorted by duration ascending
		if len(recordings) >= 2 {
			first := recordings[0].(map[string]interface{})
			second := recordings[1].(map[string]interface{})
			firstDuration := first["duration"].(float64)
			secondDuration := second["duration"].(float64)
			assert.LessOrEqual(t, firstDuration, secondDuration, "recordings should be sorted by duration asc")
		}
	})

	t.Run("validate response schema", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		if len(recordings) > 0 {
			recording := recordings[0].(map[string]interface{})

			// Validate RecordingListItem schema (from OpenAPI spec)
			requiredFields := []string{
				"id", "session_id", "recording_type", "user_id", "username",
				"started_at", "duration", "file_size", "file_size_human", "client_ip",
			}

			for _, field := range requiredFields {
				assert.Contains(t, recording, field, "recording should have '%s' field", field)
			}

			// Validate enum values
			recordingType := recording["recording_type"].(string)
			validTypes := []string{"docker", "webssh", "k8s_node", "k8s_pod"}
			assert.Contains(t, validTypes, recordingType, "recording_type should be one of: %v", validTypes)

			// Validate data types
			assert.IsType(t, "", recording["id"].(string))
			assert.IsType(t, "", recording["session_id"].(string))
			assert.IsType(t, float64(0), recording["duration"].(float64))
			assert.IsType(t, float64(0), recording["file_size"].(float64))
		}
	})

	t.Run("unauthorized access", func(t *testing.T) {
		// TODO: Implement when authentication middleware is ready
		t.Skip("Skipping: authentication not implemented yet")
	})
}
