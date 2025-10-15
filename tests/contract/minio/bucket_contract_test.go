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

// Bucket API contract test using a live MinIO testcontainer.
func TestBucketContract(t *testing.T) {
    env := helpers.SetupMinioTestEnv(t)
    defer env.CleanupFunc()

    baseURL := env.Server.URL
    instanceID := env.InstanceID.String()

    t.Run("GET /api/v1/minio/instances/{id}/buckets lists buckets", func(t *testing.T) {
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/buckets", baseURL, instanceID)
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

    t.Run("POST /api/v1/minio/instances/{id}/buckets creates bucket", func(t *testing.T) {
        payload := map[string]interface{}{
            "name":     "test-bucket",
            "location": "us-east-1",
        }
        body, _ := json.Marshal(payload)

        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/buckets", baseURL, instanceID)
        req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")

        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusCreated, resp.StatusCode)
        var result map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&result)
        assert.True(t, result["success"].(bool))
        assert.NotNil(t, result["data"])
    })

    t.Run("GET /api/v1/minio/instances/{id}/buckets/{name} gets bucket info", func(t *testing.T) {
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/buckets/%s", baseURL, instanceID, "test-bucket")
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

    t.Run("PUT /api/v1/minio/instances/{id}/buckets/{name}/policy updates policy", func(t *testing.T) {
        payload := map[string]interface{}{
            "policy": map[string]interface{}{
                "Version": "2012-10-17",
                "Statement": []map[string]interface{}{
                    {
                        "Effect": "Allow",
                        "Action": []string{"s3:GetObject"},
                        "Resource": []string{
                            "arn:aws:s3:::test-bucket",
                            "arn:aws:s3:::test-bucket/*",
                        },
                    },
                },
            },
        }
        body, _ := json.Marshal(payload)

        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/buckets/%s/policy", baseURL, instanceID, "test-bucket")
        req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")

        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, http.StatusOK, resp.StatusCode)
        var result map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&result)
        assert.True(t, result["success"].(bool))
    })

    t.Run("DELETE /api/v1/minio/instances/{id}/buckets/{name} deletes bucket", func(t *testing.T) {
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/buckets/%s", baseURL, instanceID, "test-bucket")
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
