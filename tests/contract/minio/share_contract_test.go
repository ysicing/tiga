package minio_contract

import (
    "bytes"
    "encoding/json"
    "fmt"
    "mime/multipart"
    "net/http"
    "testing"

    "github.com/stretchr/testify/assert"
    helpers "github.com/ysicing/tiga/tests/integration/minio"
)

func TestShareContract(t *testing.T) {
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
    bucket := "share-bucket"
    key := "dir/share.txt"

    // Create bucket
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

    // Upload a file via file API
    {
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/files", baseURL, instanceID)
    body := &bytes.Buffer{}
    w := multipart.NewWriter(body)
    _ = w.WriteField("bucket", bucket)
    _ = w.WriteField("name", key)
    fw, _ := w.CreateFormFile("file", "share.txt")
    _, _ = fw.Write([]byte("share me"))
    w.Close()
    req, _ := http.NewRequest("POST", url, body)
    req.Header.Set("Content-Type", w.FormDataContentType())
        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, http.StatusCreated, resp.StatusCode)
    }

    var shareID string

    // Create share
    {
        payload := map[string]interface{}{
            "instance_id": instanceID,
            "bucket": bucket,
            "key":    key,
            "expiry": "7d",
        }
        body, _ := json.Marshal(payload)
        url := fmt.Sprintf("%s/api/v1/minio/shares", baseURL)
        req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, http.StatusCreated, resp.StatusCode)
        var result map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&result)
        shareID = result["data"].(map[string]interface{})["id"].(string)
        assert.NotEmpty(t, shareID)
    }

    // List shares
    {
        url := fmt.Sprintf("%s/api/v1/minio/shares", baseURL)
        req, _ := http.NewRequest("GET", url, nil)
        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, http.StatusOK, resp.StatusCode)
    }

    // Revoke share
    {
        url := fmt.Sprintf("%s/api/v1/minio/shares/%s", baseURL, shareID)
        req, _ := http.NewRequest("DELETE", url, nil)
        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, http.StatusOK, resp.StatusCode)
    }
}

// no helper (inline multipart in test)
