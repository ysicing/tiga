package minio_contract

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "net/textproto"
    "strings"
    "testing"

    "github.com/stretchr/testify/assert"
    helpers "github.com/ysicing/tiga/tests/integration/minio"
)

func TestFileContract(t *testing.T) {
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
    bucket := "files-bucket"

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

    // Upload a small text file
    key := "hello/test.txt"
    content := "hello world"
    {
        var body bytes.Buffer
        writer := multipart.NewWriter(&body)
        _ = writer.WriteField("bucket", bucket)
        _ = writer.WriteField("name", key)

        // Create file part
        h := make(textproto.MIMEHeader)
        h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, "test.txt"))
        h.Set("Content-Type", "text/plain")
        part, _ := writer.CreatePart(h)
        _, _ = io.Copy(part, strings.NewReader(content))
        writer.Close()

        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/files", baseURL, instanceID)
        req, _ := http.NewRequest("POST", url, &body)
        req.Header.Set("Content-Type", writer.FormDataContentType())
        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, http.StatusCreated, resp.StatusCode)
    }

    // List
    {
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/files?bucket=%s&prefix=hello/", baseURL, instanceID, bucket)
        req, _ := http.NewRequest("GET", url, nil)
        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, http.StatusOK, resp.StatusCode)
    }

    // Get presigned download URL and fetch it
    {
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/files/download?bucket=%s&key=%s", baseURL, instanceID, bucket, key)
        req, _ := http.NewRequest("GET", url, nil)
        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, http.StatusOK, resp.StatusCode)

        var result map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&result)
        u := result["data"].(map[string]interface{})["url"].(string)
        // Fetch URL
        dl, err := http.Get(u)
        assert.NoError(t, err)
        defer dl.Body.Close()
        assert.Equal(t, http.StatusOK, dl.StatusCode)
    }

    // Delete file
    {
        payload := map[string]interface{}{"bucket": bucket, "keys": []string{key}}
        body, _ := json.Marshal(payload)
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/files", baseURL, instanceID)
        req, _ := http.NewRequest("DELETE", url, bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
        resp, err := http.DefaultClient.Do(req)
        assert.NoError(t, err)
        defer resp.Body.Close()
        assert.Equal(t, http.StatusOK, resp.StatusCode)
    }
}

