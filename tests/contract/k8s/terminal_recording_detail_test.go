package k8s_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/tests/contract"
)

// TestGetRecordingDetailK8sMetadata tests GET /api/v1/recordings/:id with K8s metadata
// Reference: 010-k8s-pod-009 T005 - Recording detail API contract test
// MUST FAIL until K8s metadata serialization is implemented
func TestGetRecordingDetailK8sMetadata(t *testing.T) {
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

	clusterID := uuid.New().String()

	t.Run("k8s_node recording detail with metadata", func(t *testing.T) {
		// Create K8s node recording
		nodeRecording := &models.TerminalRecording{
			RecordingType: models.RecordingTypeK8sNode,
			UserID:        helper.AdminUser.ID,
			Username:      helper.AdminUser.Username,
			TypeMetadata: datatypes.JSON([]byte(fmt.Sprintf(`{
				"cluster_id": "%s",
				"node_name": "master-node-1",
				"node_ip": "192.168.1.10"
			}`, clusterID))),
			FilePath: "/recordings/k8s_node/2025-10-28/master-node-1.cast",
			FileSize: 2048,
			Duration: 180,
		}
		err := helper.DB.Create(nodeRecording).Error
		require.NoError(t, err)

		// Request recording detail
		path := fmt.Sprintf("/api/v1/recordings/%s", nodeRecording.ID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		recording := data["data"].(map[string]interface{})

		// Assert recording type
		assert.Equal(t, models.RecordingTypeK8sNode, recording["recording_type"])

		// Assert type_metadata structure for k8s_node
		metadata, ok := recording["type_metadata"].(map[string]interface{})
		require.True(t, ok, "type_metadata should be present and be a map")

		// Required fields for k8s_node
		assert.Equal(t, clusterID, metadata["cluster_id"], "cluster_id should match")
		assert.Equal(t, "master-node-1", metadata["node_name"], "node_name should match")
		assert.Equal(t, "192.168.1.10", metadata["node_ip"], "node_ip should match")
	})

	t.Run("k8s_pod recording detail with metadata", func(t *testing.T) {
		// Create K8s pod recording
		podRecording := &models.TerminalRecording{
			RecordingType: models.RecordingTypeK8sPod,
			UserID:        helper.AdminUser.ID,
			Username:      helper.AdminUser.Username,
			TypeMetadata: datatypes.JSON([]byte(fmt.Sprintf(`{
				"cluster_id": "%s",
				"namespace": "production",
				"pod_name": "api-server-abc123",
				"container_name": "api",
				"pod_ip": "10.244.1.5"
			}`, clusterID))),
			FilePath: "/recordings/k8s_pod/2025-10-28/api-server-abc123.cast",
			FileSize: 4096,
			Duration: 600,
		}
		err := helper.DB.Create(podRecording).Error
		require.NoError(t, err)

		// Request recording detail
		path := fmt.Sprintf("/api/v1/recordings/%s", podRecording.ID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		recording := data["data"].(map[string]interface{})

		// Assert recording type
		assert.Equal(t, models.RecordingTypeK8sPod, recording["recording_type"])

		// Assert type_metadata structure for k8s_pod
		metadata, ok := recording["type_metadata"].(map[string]interface{})
		require.True(t, ok, "type_metadata should be present")

		// Required fields for k8s_pod
		assert.Equal(t, clusterID, metadata["cluster_id"], "cluster_id should match")
		assert.Equal(t, "production", metadata["namespace"], "namespace should match")
		assert.Equal(t, "api-server-abc123", metadata["pod_name"], "pod_name should match")
		assert.Equal(t, "api", metadata["container_name"], "container_name should match")
		assert.Equal(t, "10.244.1.5", metadata["pod_ip"], "pod_ip should match")
	})

	t.Run("verify type_metadata is properly deserialized", func(t *testing.T) {
		// Create recording with complex metadata
		recording := &models.TerminalRecording{
			RecordingType: models.RecordingTypeK8sPod,
			UserID:        helper.AdminUser.ID,
			Username:      helper.AdminUser.Username,
			TypeMetadata: datatypes.JSON([]byte(fmt.Sprintf(`{
				"cluster_id": "%s",
				"namespace": "kube-system",
				"pod_name": "coredns-abc",
				"container_name": "coredns",
				"labels": {
					"app": "coredns",
					"tier": "control-plane"
				}
			}`, clusterID))),
			FilePath: "/recordings/k8s_pod/2025-10-28/coredns-abc.cast",
			FileSize: 1024,
			Duration: 300,
		}
		err := helper.DB.Create(recording).Error
		require.NoError(t, err)

		// Request detail
		path := fmt.Sprintf("/api/v1/recordings/%s", recording.ID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		rec := data["data"].(map[string]interface{})

		// Verify nested metadata fields are properly deserialized
		metadata := rec["type_metadata"].(map[string]interface{})
		labels, ok := metadata["labels"].(map[string]interface{})
		require.True(t, ok, "labels should be present and be a map")
		assert.Equal(t, "coredns", labels["app"])
		assert.Equal(t, "control-plane", labels["tier"])
	})

	t.Run("metadata validation", func(t *testing.T) {
		// Verify that the response includes proper metadata structure
		// even when optional fields are present

		recording := &models.TerminalRecording{
			RecordingType: models.RecordingTypeK8sNode,
			UserID:        helper.AdminUser.ID,
			Username:      helper.AdminUser.Username,
			TypeMetadata: datatypes.JSON([]byte(fmt.Sprintf(`{
				"cluster_id": "%s",
				"node_name": "worker-1"
			}`, clusterID))),
			FilePath: "/recordings/k8s_node/2025-10-28/worker-1.cast",
			FileSize: 512,
			Duration: 120,
		}
		err := helper.DB.Create(recording).Error
		require.NoError(t, err)

		path := fmt.Sprintf("/api/v1/recordings/%s", recording.ID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		rec := data["data"].(map[string]interface{})

		// Basic response structure validation
		assert.NotEmpty(t, rec["id"], "id should be present")
		assert.NotEmpty(t, rec["session_id"], "session_id should be present")
		assert.NotEmpty(t, rec["user_id"], "user_id should be present")
		assert.NotEmpty(t, rec["username"], "username should be present")
		assert.NotEmpty(t, rec["recording_type"], "recording_type should be present")
		assert.NotEmpty(t, rec["created_at"], "created_at should be present")
		assert.NotNil(t, rec["type_metadata"], "type_metadata should not be nil")
	})
}
