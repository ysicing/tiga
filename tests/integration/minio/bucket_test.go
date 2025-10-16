package minio_integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Integration_Bucket(t *testing.T) {
	env := SetupMinioTestEnv(t)
	if env.Server == nil {
		t.Skip("no docker")
		return
	}
	defer env.CleanupFunc()

	baseURL := env.Server.URL
	bucket := "int-bucket"

	// Create bucket
	{
		payload := map[string]interface{}{"name": bucket}
		body, _ := json.Marshal(payload)
		url := fmt.Sprintf("%s/api/v1/minio/instances/%s/buckets", baseURL, env.InstanceID.String())
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	// Update policy to public read
	{
		payload := map[string]interface{}{
			"policy": map[string]interface{}{"Version": "2012-10-17", "Statement": []map[string]interface{}{{
				"Effect": "Allow", "Action": []string{"s3:GetObject"}, "Resource": []string{fmt.Sprintf("arn:aws:s3:::%s/*", bucket)},
			}}},
		}
		body, _ := json.Marshal(payload)
		url := fmt.Sprintf("%s/api/v1/minio/instances/%s/buckets/%s/policy", baseURL, env.InstanceID.String(), bucket)
		req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Delete bucket (empty)
	{
		url := fmt.Sprintf("%s/api/v1/minio/instances/%s/buckets/%s", baseURL, env.InstanceID.String(), bucket)
		req, _ := http.NewRequest("DELETE", url, nil)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}
}
