package minio_integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Integration_Permission(t *testing.T) {
	env := SetupMinioTestEnv(t)
	if env.Server == nil {
		t.Skip("no docker")
		return
	}
	defer env.CleanupFunc()

	baseURL := env.Server.URL
	instanceID := env.InstanceID.String()

	// Create user
	var accessKey string
	{
		url := fmt.Sprintf("%s/api/v1/minio/instances/%s/users", baseURL, instanceID)
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var res map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&res)
		accessKey = res["data"].(map[string]interface{})["access_key"].(string)
	}

	// Create bucket
	bucket := "perm-int"
	{
		payload := map[string]interface{}{"name": bucket}
		body, _ := json.Marshal(payload)
		url := fmt.Sprintf("%s/api/v1/minio/instances/%s/buckets", baseURL, instanceID)
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	// Grant readonly
	var permID string
	{
		payload := map[string]interface{}{"instance_id": instanceID, "user": accessKey, "bucket": bucket, "permission": "readonly"}
		body, _ := json.Marshal(payload)
		url := fmt.Sprintf("%s/api/v1/minio/permissions", baseURL)
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var res map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&res)
		permID = res["data"].(map[string]interface{})["id"].(string)
	}

	// Revoke
	{
		url := fmt.Sprintf("%s/api/v1/minio/permissions/%s?instance_id=%s&user=%s", baseURL, permID, instanceID, accessKey)
		req, _ := http.NewRequest("DELETE", url, nil)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}
}
