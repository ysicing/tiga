package minio_integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Integration_Instance(t *testing.T) {
	env := SetupMinioTestEnv(t)
	if env.Server == nil {
		t.Skip("no docker")
		return
	}
	defer env.CleanupFunc()

	baseURL := env.Server.URL

	// Test connection
	{
		url := fmt.Sprintf("%s/api/v1/minio/instances/%s/test", baseURL, env.InstanceID.String())
		req, _ := http.NewRequest("POST", url, nil)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Additional failure scenarios will be covered when auth + audit middlewares are enabled in tests
}
