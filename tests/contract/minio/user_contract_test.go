package minio_contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	helpers "github.com/ysicing/tiga/tests/integration/minio"
)

// MinIO user management API contract test using live testcontainer, skipped if Docker unavailable.
func TestUserContract(t *testing.T) {
	env := helpers.SetupMinioTestEnv(t)
	if env.CleanupFunc != nil {
		defer env.CleanupFunc()
	}

	// If SetupMinioTestEnv skipped (no Docker), return to mark as skipped
	if env.Server == nil {
		t.Skip("skipped due to unavailable test environment")
		return
	}

	baseURL := env.Server.URL
	instanceID := env.InstanceID.String()

	var createdAccessKey string
	var createdSecretKey string

	t.Run("POST /api/v1/minio/instances/{id}/users creates user and returns secret once", func(t *testing.T) {
		payload := map[string]interface{}{}
		body, _ := json.Marshal(payload)
		url := fmt.Sprintf("%s/api/v1/minio/instances/%s/users", baseURL, instanceID)
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		assert.NotEmpty(t, data["access_key"])
		assert.NotEmpty(t, data["secret_key"])

		createdAccessKey = data["access_key"].(string)
		createdSecretKey = data["secret_key"].(string)
		_ = createdSecretKey
	})

	t.Run("GET /api/v1/minio/instances/{id}/users lists users", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/minio/instances/%s/users", baseURL, instanceID)
		req, _ := http.NewRequest("GET", url, nil)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))
		assert.NotNil(t, result["data"])
	})

	t.Run("DELETE /api/v1/minio/instances/{id}/users/{username} deletes user", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/minio/instances/%s/users/%s", baseURL, instanceID, createdAccessKey)
		req, _ := http.NewRequest("DELETE", url, nil)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))
	})
}
