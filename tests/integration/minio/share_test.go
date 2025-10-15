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

func Test_Integration_Share(t *testing.T) {
    env := SetupMinioTestEnv(t)
    if env.Server == nil { t.Skip("no docker"); return }
    defer env.CleanupFunc()

    baseURL := env.Server.URL
    instanceID := env.InstanceID.String()

    bucket := "share-int"
    key := "dir/share.txt"

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

    // Upload a file
    {
        buf := &bytes.Buffer{}
        w := multipart.NewWriter(buf)
        _ = w.WriteField("bucket", bucket)
        _ = w.WriteField("name", key)
        fw, _ := w.CreateFormFile("file", "share.txt")
        _, _ = io.Copy(fw, bytes.NewBufferString("shared"))
        w.Close()
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/files", baseURL, instanceID)
        req, _ := http.NewRequest("POST", url, buf)
        req.Header.Set("Content-Type", w.FormDataContentType())
        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()
        require.Equal(t, http.StatusCreated, resp.StatusCode)
    }

    var shareID string
    var presigned string
    // Create share
    {
        payload := map[string]interface{}{ "instance_id": instanceID, "bucket": bucket, "key": key, "expiry": "7d" }
        body, _ := json.Marshal(payload)
        url := fmt.Sprintf("%s/api/v1/minio/shares", baseURL)
        req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()
        require.Equal(t, http.StatusCreated, resp.StatusCode)
        var res map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&res)
        data := res["data"].(map[string]interface{})
        shareID = data["id"].(string)
        presigned = data["url"].(string)
    }

    // Unauthenticated access should succeed via presigned URL
    {
        resp, err := http.Get(presigned)
        require.NoError(t, err)
        defer resp.Body.Close()
        require.Equal(t, http.StatusOK, resp.StatusCode)
    }

    // List and Revoke
    {
        req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/minio/shares", baseURL), nil)
        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()
        require.Equal(t, http.StatusOK, resp.StatusCode)

        req2, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/minio/shares/%s", baseURL, shareID), nil)
        resp2, err := http.DefaultClient.Do(req2)
        require.NoError(t, err)
        defer resp2.Body.Close()
        require.Equal(t, http.StatusOK, resp2.StatusCode)
    }
}

