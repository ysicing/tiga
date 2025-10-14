package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestInstanceContract tests the instance management API contract
// This test MUST FAIL initially as no implementation exists
func TestInstanceContract(t *testing.T) {
	// Setup test server (will be replaced with actual server later)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"success":false,"error":"not implemented"}`))
	}))
	defer ts.Close()

	t.Run("GET /api/v1/database/instances returns 200 and instance list", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/database/instances", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Expected: 200 OK with array of instances
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		assert.True(t, result["success"].(bool))
		assert.NotNil(t, result["data"])
	})

	t.Run("POST /api/v1/database/instances creates instance and returns 201", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":        "Test MySQL",
			"type":        "mysql",
			"host":        "localhost",
			"port":        3306,
			"username":    "root",
			"password":    "test123",
			"description": "Test instance",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/database/instances", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Expected: 201 Created
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		assert.True(t, result["success"].(bool))
		assert.NotNil(t, result["data"])
	})

	t.Run("GET /api/v1/database/instances/{id} returns instance details", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/database/instances/1", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Expected: 200 OK with instance object
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		assert.True(t, result["success"].(bool))
		data := result["data"].(map[string]interface{})
		assert.NotNil(t, data["id"])
		assert.NotNil(t, data["name"])
		assert.NotNil(t, data["type"])
	})

	t.Run("DELETE /api/v1/database/instances/{id} deletes instance and returns 200", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", ts.URL+"/api/v1/database/instances/1", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Expected: 200 OK
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		assert.True(t, result["success"].(bool))
	})

	t.Run("POST /api/v1/database/instances/{id}/test returns connection status", func(t *testing.T) {
		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/database/instances/1/test", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Expected: 200 OK with status
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		assert.True(t, result["success"].(bool))
		data := result["data"].(map[string]interface{})
		assert.NotNil(t, data["status"])
	})
}
