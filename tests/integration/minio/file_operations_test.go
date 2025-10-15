package minio_integration

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "testing"

    "github.com/stretchr/testify/require"
)

func Test_Integration_FileOperations(t *testing.T) {
    env := SetupMinioTestEnv(t)
    if env.Server == nil { t.Skip("no docker"); return }
    defer env.CleanupFunc()

    baseURL := env.Server.URL
    instanceID := env.InstanceID.String()

    bucket := "file-int"
    key := "dir/ops.txt"

    // Create bucket
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

    // Upload
    {
        buf := &bytes.Buffer{}
        w := multipart.NewWriter(buf)
        _ = w.WriteField("bucket", bucket)
        _ = w.WriteField("name", key)
        fw, _ := w.CreateFormFile("file", "ops.txt")
        _, _ = io.Copy(fw, bytes.NewBufferString("hello integration"))
        w.Close()
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/files", baseURL, instanceID)
        req, _ := http.NewRequest("POST", url, buf)
        req.Header.Set("Content-Type", w.FormDataContentType())
        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()
        require.Equal(t, http.StatusCreated, resp.StatusCode)
    }

    // List
    {
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/files?bucket=%s&prefix=dir/", baseURL, instanceID, bucket)
        req, _ := http.NewRequest("GET", url, nil)
        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()
        require.Equal(t, http.StatusOK, resp.StatusCode)
    }

    // Download presigned
    {
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/files/download?bucket=%s&key=%s", baseURL, instanceID, bucket, key)
        req, _ := http.NewRequest("GET", url, nil)
        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()
        require.Equal(t, http.StatusOK, resp.StatusCode)
    }

    // Delete
    {
        payload := map[string]interface{}{"bucket": bucket, "keys": []string{key}}
        body, _ := json.Marshal(payload)
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/files", baseURL, instanceID)
        req, _ := http.NewRequest("DELETE", url, bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()
        require.Equal(t, http.StatusOK, resp.StatusCode)
    }
}

