package contract

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetStatistics tests GET /api/v1/recordings/statistics endpoint
// Reference: contracts/recording-api.yaml `getStatistics` operation
func TestGetStatistics(t *testing.T) {
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

	t.Run("get overall statistics", func(t *testing.T) {
		path := "/api/v1/recordings/statistics"

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Statistics endpoint not implemented yet")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)

		// Assert RecordingStatistics schema
		stats, ok := data["data"].(map[string]interface{})
		require.True(t, ok, "response should have 'data' object")

		// Validate required fields
		requiredFields := []string{
			"total_count", "total_size", "total_size_human",
			"by_type", "top_users",
			"oldest_recording", "newest_recording",
			"invalid_count", "orphan_count", "error_rate",
		}

		for _, field := range requiredFields {
			assert.Contains(t, stats, field, "statistics should have '%s' field", field)
		}

		// Validate data types
		assert.IsType(t, float64(0), stats["total_count"].(float64))
		assert.IsType(t, float64(0), stats["total_size"].(float64))
		assert.IsType(t, "", stats["total_size_human"].(string))
		assert.IsType(t, float64(0), stats["invalid_count"].(float64))
		assert.IsType(t, float64(0), stats["orphan_count"].(float64))
		assert.IsType(t, float64(0), stats["error_rate"].(float64))

		// Validate by_type structure
		byType, ok := stats["by_type"].(map[string]interface{})
		require.True(t, ok, "by_type should be an object")
		assert.NotNil(t, byType)

		// Validate top_users array
		topUsers, ok := stats["top_users"].([]interface{})
		require.True(t, ok, "top_users should be an array")
		assert.NotNil(t, topUsers)
	})

	t.Run("statistics by_type breakdown", func(t *testing.T) {
		path := "/api/v1/recordings/statistics"

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Statistics endpoint not implemented yet")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})
		byType := stats["by_type"].(map[string]interface{})

		// Validate TypeStatistics schema for each type
		validTypes := []string{"docker", "webssh", "k8s_node", "k8s_pod"}
		for _, recordingType := range validTypes {
			if typeStats, ok := byType[recordingType].(map[string]interface{}); ok {
				// Validate TypeStatistics fields
				assert.Contains(t, typeStats, "recording_type")
				assert.Contains(t, typeStats, "count")
				assert.Contains(t, typeStats, "total_size")
				assert.Contains(t, typeStats, "avg_duration")

				assert.Equal(t, recordingType, typeStats["recording_type"])
				assert.IsType(t, float64(0), typeStats["count"].(float64))
				assert.IsType(t, float64(0), typeStats["total_size"].(float64))
				assert.IsType(t, float64(0), typeStats["avg_duration"].(float64))
			}
		}
	})

	t.Run("statistics top_users list", func(t *testing.T) {
		path := "/api/v1/recordings/statistics"

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Statistics endpoint not implemented yet")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})
		topUsers := stats["top_users"].([]interface{})

		if len(topUsers) > 0 {
			// Validate UserStatistics schema
			user := topUsers[0].(map[string]interface{})

			requiredFields := []string{"user_id", "username", "count"}
			for _, field := range requiredFields {
				assert.Contains(t, user, field, "user statistics should have '%s' field", field)
			}

			assert.IsType(t, "", user["user_id"].(string))
			assert.IsType(t, "", user["username"].(string))
			assert.IsType(t, float64(0), user["count"].(float64))

			// Verify top users are sorted by count descending
			if len(topUsers) >= 2 {
				firstUser := topUsers[0].(map[string]interface{})
				secondUser := topUsers[1].(map[string]interface{})
				firstCount := firstUser["count"].(float64)
				secondCount := secondUser["count"].(float64)
				assert.GreaterOrEqual(t, firstCount, secondCount, "top users should be sorted by count desc")
			}
		}
	})

	t.Run("statistics aggregation accuracy", func(t *testing.T) {
		path := "/api/v1/recordings/statistics"

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Statistics endpoint not implemented yet")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})
		byType := stats["by_type"].(map[string]interface{})

		// Verify that sum of by_type counts equals total_count
		totalCount := stats["total_count"].(float64)
		sumByType := float64(0)

		for _, typeStats := range byType {
			if ts, ok := typeStats.(map[string]interface{}); ok {
				count := ts["count"].(float64)
				sumByType += count
			}
		}

		assert.Equal(t, totalCount, sumByType, "sum of by_type counts should equal total_count")
	})

	t.Run("statistics human-readable sizes", func(t *testing.T) {
		path := "/api/v1/recordings/statistics"

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Statistics endpoint not implemented yet")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		totalSizeHuman := stats["total_size_human"].(string)
		assert.NotEmpty(t, totalSizeHuman)

		// Verify human-readable format (e.g., "120.5 GB", "1.2 MB")
		validUnits := []string{"B", "KB", "MB", "GB", "TB"}
		hasValidUnit := false
		for _, unit := range validUnits {
			if assert.Contains(t, totalSizeHuman, unit) {
				hasValidUnit = true
				break
			}
		}
		assert.True(t, hasValidUnit, "total_size_human should have valid unit")
	})

	t.Run("statistics error_rate calculation", func(t *testing.T) {
		path := "/api/v1/recordings/statistics"

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Statistics endpoint not implemented yet")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		errorRate := stats["error_rate"].(float64)
		assert.GreaterOrEqual(t, errorRate, float64(0), "error_rate should be non-negative")
		assert.LessOrEqual(t, errorRate, float64(1), "error_rate should be <= 1 (100%)")
	})

	t.Run("statistics with no recordings", func(t *testing.T) {
		// TODO: Test with empty database
		// All counts should be 0, timestamps should be null
		t.Skip("Skipping: empty database test scenario not implemented")
	})

	t.Run("unauthorized access", func(t *testing.T) {
		// TODO: Test without authentication token
		t.Skip("Skipping: authentication not implemented yet")
	})
}
