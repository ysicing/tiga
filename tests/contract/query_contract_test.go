package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestQueryContract tests the query execution API contract
// This test MUST FAIL initially as no implementation exists
func TestQueryContract(t *testing.T) {
	// Setup test server (will be replaced with actual server later)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"success":false,"error":"not implemented"}`))
	}))
	defer ts.Close()

	t.Run("POST /api/v1/database/instances/{id}/query executes SELECT and returns QueryResult", func(t *testing.T) {
		payload := map[string]interface{}{
			"database_name": "test_db",
			"query":         "SELECT * FROM users LIMIT 10",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/database/instances/1/query", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Expected: 200 OK with query result
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		assert.True(t, result["success"].(bool))
		data := result["data"].(map[string]interface{})
		assert.NotNil(t, data["columns"])
		assert.NotNil(t, data["rows"])
		assert.NotNil(t, data["row_count"])
		assert.NotNil(t, data["duration"])
	})

	t.Run("POST /api/v1/database/instances/{id}/query with DDL returns 400 error", func(t *testing.T) {
		payload := map[string]interface{}{
			"database_name": "test_db",
			"query":         "DROP TABLE users",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/database/instances/1/query", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Expected: 400 Bad Request with "DDL operations are forbidden"
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "DDL operations are forbidden")
	})

	t.Run("POST /api/v1/database/instances/{id}/query with DELETE without WHERE returns 400", func(t *testing.T) {
		payload := map[string]interface{}{
			"database_name": "test_db",
			"query":         "DELETE FROM users",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/database/instances/1/query", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Expected: 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "WHERE")
	})

	t.Run("POST /api/v1/database/instances/{id}/query with large result returns truncated flag", func(t *testing.T) {
		payload := map[string]interface{}{
			"database_name": "test_db",
			"query":         "SELECT * FROM large_table",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/database/instances/1/query", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Expected: 200 OK with truncated flag
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		assert.True(t, result["success"].(bool))
		data := result["data"].(map[string]interface{})
		// If truncated, should have truncated field
		if truncated, ok := data["truncated"]; ok {
			assert.True(t, truncated.(bool))
			assert.NotEmpty(t, data["message"])
		}
	})
}
