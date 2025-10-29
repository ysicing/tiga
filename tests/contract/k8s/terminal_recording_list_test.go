package k8s_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/tests/contract"
	"gorm.io/datatypes"
)

// TestListRecordingsK8sTypeFilter tests GET /api/v1/recordings with K8s type filtering
// Reference: 010-k8s-pod-009 T004 - Recording API contract test
// MUST FAIL until K8s recording filtering is implemented
func TestListRecordingsK8sTypeFilter(t *testing.T) {
	helper := contract.NewTestHelper(t)
	defer helper.Cleanup()

	// Setup test database and router
	if err := helper.SetupTestDB(); err != nil {
		t.Skip("Skipping test: database setup not implemented yet")
		return
	}
	if err := helper.SetupRouter(nil); err != nil {
		t.Skip("Skipping test: router setup not implemented yet")
		return
	}

	// Create test recordings with K8s types
	clusterID := "550e8400-e29b-41d4-a716-446655440000"

	// K8s Node recording
	nodeRecording := &models.TerminalRecording{
		RecordingType: models.RecordingTypeK8sNode,
		UserID:        helper.AdminUser.ID,
		Username:      helper.AdminUser.Username,
		TypeMetadata: datatypes.JSON([]byte(fmt.Sprintf(`{
			"cluster_id": "%s",
			"node_name": "test-node-1"
		}`, clusterID))),
		FilePath: "/recordings/k8s_node/2025-10-28/test-node-1.cast",
		FileSize: 1024,
		Duration: 120,
	}
	err := helper.DB.Create(nodeRecording).Error
	require.NoError(t, err, "Failed to create K8s node recording")

	// K8s Pod recording
	podRecording := &models.TerminalRecording{
		RecordingType: models.RecordingTypeK8sPod,
		UserID:        helper.AdminUser.ID,
		Username:      helper.AdminUser.Username,
		TypeMetadata: datatypes.JSON([]byte(fmt.Sprintf(`{
			"cluster_id": "%s",
			"namespace": "default",
			"pod_name": "nginx-pod",
			"container_name": "nginx"
		}`, clusterID))),
		FilePath: "/recordings/k8s_pod/2025-10-28/nginx-pod.cast",
		FileSize: 2048,
		Duration: 300,
	}
	err = helper.DB.Create(podRecording).Error
	require.NoError(t, err, "Failed to create K8s pod recording")

	// Docker recording (for comparison)
	dockerRecording := &models.TerminalRecording{
		RecordingType: models.RecordingTypeDocker,
		UserID:        helper.AdminUser.ID,
		Username:      helper.AdminUser.Username,
		TypeMetadata:  datatypes.JSON([]byte(`{"instance_id": "docker-001"}`)),
		FilePath:      "/recordings/docker/2025-10-28/container-1.cast",
		FileSize:      512,
		Duration:      60,
	}
	err = helper.DB.Create(dockerRecording).Error
	require.NoError(t, err, "Failed to create Docker recording")

	t.Run("filter by recording_type=k8s_node", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings?recording_type=k8s_node", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		// Assert at least one k8s_node recording found
		assert.GreaterOrEqual(t, len(recordings), 1, "Should have at least 1 k8s_node recording")

		// Verify all returned recordings are k8s_node type
		for _, rec := range recordings {
			recording := rec.(map[string]interface{})
			assert.Equal(t, models.RecordingTypeK8sNode, recording["recording_type"])

			// Verify type_metadata exists and contains cluster_id, node_name
			metadata, ok := recording["type_metadata"].(map[string]interface{})
			require.True(t, ok, "type_metadata should be present")
			assert.NotEmpty(t, metadata["cluster_id"], "cluster_id should be present")
			assert.NotEmpty(t, metadata["node_name"], "node_name should be present")
		}
	})

	t.Run("filter by recording_type=k8s_pod", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings?recording_type=k8s_pod", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		// Assert at least one k8s_pod recording found
		assert.GreaterOrEqual(t, len(recordings), 1, "Should have at least 1 k8s_pod recording")

		// Verify all returned recordings are k8s_pod type
		for _, rec := range recordings {
			recording := rec.(map[string]interface{})
			assert.Equal(t, models.RecordingTypeK8sPod, recording["recording_type"])

			// Verify type_metadata contains K8s pod information
			metadata, ok := recording["type_metadata"].(map[string]interface{})
			require.True(t, ok, "type_metadata should be present")
			assert.NotEmpty(t, metadata["cluster_id"], "cluster_id should be present")
			assert.NotEmpty(t, metadata["namespace"], "namespace should be present")
			assert.NotEmpty(t, metadata["pod_name"], "pod_name should be present")
			assert.NotEmpty(t, metadata["container_name"], "container_name should be present")
		}
	})

	t.Run("filter by cluster_id", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/recordings?cluster_id=%s", clusterID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		// Should find both k8s_node and k8s_pod recordings for this cluster
		assert.GreaterOrEqual(t, len(recordings), 2, "Should have at least 2 recordings for cluster")

		// Verify all recordings belong to the specified cluster
		for _, rec := range recordings {
			recording := rec.(map[string]interface{})
			recordingType := recording["recording_type"].(string)

			// Only K8s recordings should be returned
			assert.Contains(t, []string{models.RecordingTypeK8sNode, models.RecordingTypeK8sPod}, recordingType)

			metadata, ok := recording["type_metadata"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, clusterID, metadata["cluster_id"])
		}
	})

	t.Run("filter by node_name", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings?node_name=test-node-1", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		// Should find the k8s_node recording
		assert.GreaterOrEqual(t, len(recordings), 1, "Should have at least 1 node recording")

		// Verify all recordings match the node filter
		for _, rec := range recordings {
			recording := rec.(map[string]interface{})
			assert.Equal(t, models.RecordingTypeK8sNode, recording["recording_type"])

			metadata := recording["type_metadata"].(map[string]interface{})
			assert.Equal(t, "test-node-1", metadata["node_name"])
		}
	})

	t.Run("filter by namespace", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings?namespace=default", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		// Should find k8s_pod recordings in 'default' namespace
		assert.GreaterOrEqual(t, len(recordings), 1, "Should have at least 1 pod recording")

		// Verify all recordings match the namespace filter
		for _, rec := range recordings {
			recording := rec.(map[string]interface{})
			assert.Equal(t, models.RecordingTypeK8sPod, recording["recording_type"])

			metadata := recording["type_metadata"].(map[string]interface{})
			assert.Equal(t, "default", metadata["namespace"])
		}
	})

	t.Run("filter by pod_name", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings?pod_name=nginx-pod", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		// Should find the nginx-pod recording
		assert.GreaterOrEqual(t, len(recordings), 1, "Should have at least 1 pod recording")

		for _, rec := range recordings {
			recording := rec.(map[string]interface{})
			metadata := recording["type_metadata"].(map[string]interface{})
			assert.Equal(t, "nginx-pod", metadata["pod_name"])
		}
	})

	t.Run("filter by container_name", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings?container_name=nginx", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		// Should find recordings with the nginx container
		assert.GreaterOrEqual(t, len(recordings), 1, "Should have at least 1 container recording")

		for _, rec := range recordings {
			recording := rec.(map[string]interface{})
			metadata := recording["type_metadata"].(map[string]interface{})
			assert.Equal(t, "nginx", metadata["container_name"])
		}
	})

	t.Run("combined filters - cluster and namespace", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/recordings?cluster_id=%s&namespace=default", clusterID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		recordings := responseData["recordings"].([]interface{})

		// Should find pod recordings in this cluster and namespace
		assert.GreaterOrEqual(t, len(recordings), 1, "Should have pod recordings")

		for _, rec := range recordings {
			recording := rec.(map[string]interface{})
			metadata := recording["type_metadata"].(map[string]interface{})
			assert.Equal(t, clusterID, metadata["cluster_id"])
			assert.Equal(t, "default", metadata["namespace"])
		}
	})

	t.Run("pagination with K8s filters", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/recordings?recording_type=k8s_pod&page=1&limit=10")
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})

		// Verify pagination structure
		pagination, ok := responseData["pagination"].(map[string]interface{})
		require.True(t, ok, "pagination should be present")
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(10), pagination["limit"])
	})
}
