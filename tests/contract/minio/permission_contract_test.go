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

func TestPermissionContract(t *testing.T) {
    env := helpers.SetupMinioTestEnv(t)
    if env.CleanupFunc != nil {
        defer env.CleanupFunc()
    }
    if env.Server == nil {
        t.Skip("skipped due to unavailable test environment")
        return
    }

    baseURL := env.Server.URL
    instanceID := env.InstanceID.String()

    // Create user
    {
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/users", baseURL, instanceID)
        req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(`{}`)))
        req.Header.Set("Content-Type", "application/json")
        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, http.StatusCreated, resp.StatusCode)
    }

    // Create bucket
    bucket := "perm-bucket"
    {
        payload := map[string]interface{}{"name": bucket}
        body, _ := json.Marshal(payload)
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/buckets", baseURL, instanceID)
        req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, http.StatusCreated, resp.StatusCode)
    }

    // List users to get created access key
    var accessKey string
    {
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/users", baseURL, instanceID)
        req, _ := http.NewRequest("GET", url, nil)
        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()
        var result map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&result)
        arr, _ := result["data"].([]interface{})
        assert.GreaterOrEqual(t, len(arr), 1)
        // Pick first user that is not minioadmin if exists
        for _, it := range arr {
            m := it.(map[string]interface{})
            if m["access_key"].(string) != "minioadmin" {
                accessKey = m["access_key"].(string)
                break
            }
        }
        if accessKey == "" {
            // fallback to first
            accessKey = arr[0].(map[string]interface{})["access_key"].(string)
        }
    }

    // Grant permission readonly on bucket
    var permID string
    {
        payload := map[string]interface{}{
            "instance_id": instanceID,
            "user":        accessKey,
            "bucket":      bucket,
            "permission":  "readonly",
        }
        body, _ := json.Marshal(payload)
        url := fmt.Sprintf("%s/api/v1/minio/permissions", baseURL)
        req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, http.StatusCreated, resp.StatusCode)
        var result map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&result)
        permID = result["data"].(map[string]interface{})["id"].(string)
        assert.NotEmpty(t, permID)
    }

    // List permissions by user
    {
        url := fmt.Sprintf("%s/api/v1/minio/permissions?instance_id=%s&user=%s", baseURL, instanceID, accessKey)
        req, _ := http.NewRequest("GET", url, nil)
        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, http.StatusOK, resp.StatusCode)
    }

    // Revoke
    {
        url := fmt.Sprintf("%s/api/v1/minio/permissions/%s?instance_id=%s&user=%s", baseURL, permID, instanceID, accessKey)
        req, _ := http.NewRequest("DELETE", url, nil)
        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, http.StatusOK, resp.StatusCode)
    }
}

