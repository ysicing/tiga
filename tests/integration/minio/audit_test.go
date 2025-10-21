package minio_integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ysicing/tiga/internal/models"
)

func Test_Integration_Audit(t *testing.T) {
	env := SetupMinioTestEnv(t)
	if env.Server == nil {
		t.Skip("no docker")
		return
	}
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

	// Verify an audit log exists in the unified audit_events table
	// MinIO audit has been migrated to the unified audit system
	var cnt int64
	err := env.DB.Model(&models.AuditEvent{}).
		Where("subsystem = ? AND action = ? AND resource_type = ?",
			models.SubsystemMinIO,
			models.ActionCreated,
			models.ResourceTypeMinIO).
		Count(&cnt).Error
	require.NoError(t, err)
	require.GreaterOrEqual(t, cnt, int64(1), "Should have at least one audit event for bucket creation")
}
