package minio_integration

import (
    "bytes"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "testing"

    "github.com/stretchr/testify/require"
)

func Test_Integration_Security(t *testing.T) {
    env := SetupMinioTestEnv(t)
    if env.Server == nil { t.Skip("no docker"); return }
    defer env.CleanupFunc()

    baseURL := env.Server.URL
    instanceID := env.InstanceID.String()

    bucket := "sec-int"
    // Create bucket
    {
        body := bytes.NewBufferString(fmt.Sprintf(`{"name":"%s"}`, bucket))
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/buckets", baseURL, instanceID)
        req, _ := http.NewRequest("POST", url, body)
        req.Header.Set("Content-Type", "application/json")
        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()
        require.Equal(t, http.StatusCreated, resp.StatusCode)
    }

    // Attempt path traversal upload
    {
        buf := &bytes.Buffer{}
        w := multipart.NewWriter(buf)
        _ = w.WriteField("bucket", bucket)
        _ = w.WriteField("name", "../evil.txt")
        fw, _ := w.CreateFormFile("file", "evil.txt")
        _, _ = io.Copy(fw, bytes.NewBufferString("boom"))
        w.Close()
        url := fmt.Sprintf("%s/api/v1/minio/instances/%s/files", baseURL, instanceID)
        req, _ := http.NewRequest("POST", url, buf)
        req.Header.Set("Content-Type", w.FormDataContentType())
        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()
        require.Equal(t, http.StatusBadRequest, resp.StatusCode)
    }
}

