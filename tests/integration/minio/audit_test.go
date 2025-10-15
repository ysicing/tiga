package minio_integration

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "testing"

    "github.com/stretchr/testify/require"
)

func Test_Integration_Audit(t *testing.T) {
    env := SetupMinioTestEnv(t)
    if env.Server == nil { t.Skip("no docker"); return }
    defer env.CleanupFunc()

    baseURL := env.Server.URL
    instanceID := env.InstanceID.String()

    // Create a bucket to generate audit log
    bucket := "audit-bucket"
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

    // Verify an audit log exists for bucket create
    type Row struct{ Count int }
    var cnt int64
    err := env.DB.Table("minio_audit_logs").Where("operation_type = ? AND resource_type = ? AND resource_name = ?", "bucket", "bucket", bucket).Count(&cnt).Error
    require.NoError(t, err)
    require.GreaterOrEqual(t, cnt, int64(1))
}
