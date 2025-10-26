package contract

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearchRecordings tests GET /api/v1/recordings/search endpoint
// Reference: contracts/recording-api.yaml `searchRecordings` operation
func TestSearchRecordings(t *testing.T) {
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

	t.Run("search by username", func(t *testing.T) {
		keyword := "testuser"
		query := url.QueryEscape(keyword)
		path := "/api/v1/recordings/search?q=" + query

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test data not available")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})

		// Assert recordings array
		recordings, ok := responseData["recordings"].([]interface{})
		require.True(t, ok, "response should have 'recordings' array")

		// Verify search results contain keyword in username
		for _, rec := range recordings {
			recording := rec.(map[string]interface{})
			username := recording["username"].(string)
			assert.Contains(t, username, keyword, "username should contain search keyword")
		}
	})

	t.Run("search by description", func(t *testing.T) {
		keyword := "production"
		query := url.QueryEscape(keyword)
		path := "/api/v1/recordings/search?q=" + query

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test data not available")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		// Verify search results contain keyword in description
		for _, rec := range recordings {
			recording := rec.(map[string]interface{})
			if description, ok := recording["description"].(string); ok {
				assert.Contains(t, description, keyword, "description should contain search keyword")
			}
		}
	})

	t.Run("search by tags", func(t *testing.T) {
		tag := "deployment"
		query := url.QueryEscape(tag)
		path := "/api/v1/recordings/search?q=" + query

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test data not available")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		// Verify search results contain keyword in tags
		for _, rec := range recordings {
			recording := rec.(map[string]interface{})
			if tags, ok := recording["tags"].(string); ok {
				assert.Contains(t, tags, tag, "tags should contain search keyword")
			}
		}
	})

	t.Run("search with pagination", func(t *testing.T) {
		keyword := "test"
		query := url.QueryEscape(keyword)
		path := "/api/v1/recordings/search?q=" + query + "&page=1&limit=5"

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		if resp.Code == http.StatusNotFound {
			t.Skip("Test data not available")
			return
		}

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})

		// Assert pagination structure
		pagination := responseData["pagination"].(map[string]interface{})
		helper.AssertPaginationStructure(t, pagination)

		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(5), pagination["limit"])

		// Assert results are limited
		recordings := responseData["recordings"].([]interface{})
		assert.LessOrEqual(t, len(recordings), 5, "should not exceed limit")
	})

	t.Run("search with no results", func(t *testing.T) {
		keyword := "nonexistentkeyword12345"
		query := url.QueryEscape(keyword)
		path := "/api/v1/recordings/search?q=" + query

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})

		recordings := responseData["recordings"].([]interface{})
		assert.Empty(t, recordings, "should return empty array when no results")

		pagination := responseData["pagination"].(map[string]interface{})
		assert.Equal(t, float64(0), pagination["total_count"])
	})

	t.Run("search without query parameter", func(t *testing.T) {
		path := "/api/v1/recordings/search"

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		// Should return 400 Bad Request when 'q' is missing
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("search with empty query", func(t *testing.T) {
		path := "/api/v1/recordings/search?q="

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		// Should return 400 Bad Request when 'q' is empty
		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("case-insensitive search", func(t *testing.T) {
		// TODO: Verify search is case-insensitive
		// Search for "TESTUSER" should match "testuser"
		t.Skip("Skipping: case-insensitive behavior not yet verified")
	})

	t.Run("unauthorized search", func(t *testing.T) {
		// TODO: Test without authentication token
		t.Skip("Skipping: authentication not implemented yet")
	})
}
