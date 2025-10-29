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

// TestRecordingStatsK8s tests GET /api/v1/recordings/stats with K8s statistics
// Reference: 010-k8s-pod-009 T007 - Recording statistics API contract test
// MUST FAIL until K8s type statistics aggregation is implemented
func TestRecordingStatsK8s(t *testing.T) {
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

	// Create test recordings of different types
	recordings := []struct {
		recordingType string
		fileSize      int64
		duration      int
		metadata      string
	}{
		{
			recordingType: models.RecordingTypeK8sNode,
			fileSize:      1024,
			duration:      120,
			metadata:      fmt.Sprintf(`{"cluster_id": "%s", "node_name": "node-1"}`, clusterID),
		},
		{
			recordingType: models.RecordingTypeK8sNode,
			fileSize:      2048,
			duration:      180,
			metadata:      fmt.Sprintf(`{"cluster_id": "%s", "node_name": "node-2"}`, clusterID),
		},
		{
			recordingType: models.RecordingTypeK8sPod,
			fileSize:      4096,
			duration:      300,
			metadata:      fmt.Sprintf(`{"cluster_id": "%s", "namespace": "default", "pod_name": "nginx", "container_name": "nginx"}`, clusterID),
		},
		{
			recordingType: models.RecordingTypeK8sPod,
			fileSize:      8192,
			duration:      600,
			metadata:      fmt.Sprintf(`{"cluster_id": "%s", "namespace": "production", "pod_name": "api", "container_name": "api"}`, clusterID),
		},
		{
			recordingType: models.RecordingTypeDocker,
			fileSize:      512,
			duration:      60,
			metadata:      `{"instance_id": "docker-001"}`,
		},
		{
			recordingType: models.RecordingTypeWebSSH,
			fileSize:      256,
			duration:      30,
			metadata:      `{"host_id": "host-001"}`,
		},
	}

	for i, rec := range recordings {
		recording := &models.TerminalRecording{
			RecordingType: rec.recordingType,
			UserID:        helper.AdminUser.ID,
			Username:      helper.AdminUser.Username,
			TypeMetadata:  datatypes.JSON([]byte(rec.metadata)),
			FilePath:      fmt.Sprintf("/recordings/%s/test-%d.cast", rec.recordingType, i),
			FileSize:      rec.fileSize,
			Duration:      rec.duration,
		}
		err := helper.DB.Create(recording).Error
		require.NoError(t, err)
	}

	t.Run("overall statistics include K8s types", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings/stats", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		// Assert overall counts
		totalCount, ok := stats["total_count"].(float64)
		require.True(t, ok, "total_count should be numeric")
		assert.GreaterOrEqual(t, totalCount, 6.0, "Should have at least 6 recordings")

		// Assert total size
		totalSize, ok := stats["total_size"].(float64)
		require.True(t, ok, "total_size should be numeric")
		expectedTotalSize := float64(1024 + 2048 + 4096 + 8192 + 512 + 256)
		assert.Equal(t, expectedTotalSize, totalSize, "Total size should match sum of all recordings")

		// Assert total duration
		totalDuration, ok := stats["total_duration"].(float64)
		require.True(t, ok, "total_duration should be numeric")
		expectedTotalDuration := float64(120 + 180 + 300 + 600 + 60 + 30)
		assert.Equal(t, expectedTotalDuration, totalDuration, "Total duration should match sum")
	})

	t.Run("statistics by_type includes k8s_node and k8s_pod", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings/stats", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		// Assert by_type statistics
		byType, ok := stats["by_type"].(map[string]interface{})
		require.True(t, ok, "by_type should be present")

		// Check k8s_node statistics
		k8sNodeStats, ok := byType["k8s_node"].(map[string]interface{})
		require.True(t, ok, "by_type.k8s_node should be present")

		assert.Equal(t, float64(2), k8sNodeStats["count"], "k8s_node count should be 2")
		assert.Equal(t, float64(1024+2048), k8sNodeStats["total_size"], "k8s_node total_size should be 3072")
		assert.Equal(t, float64(120+180), k8sNodeStats["total_duration"], "k8s_node total_duration should be 300")

		// Check k8s_pod statistics
		k8sPodStats, ok := byType["k8s_pod"].(map[string]interface{})
		require.True(t, ok, "by_type.k8s_pod should be present")

		assert.Equal(t, float64(2), k8sPodStats["count"], "k8s_pod count should be 2")
		assert.Equal(t, float64(4096+8192), k8sPodStats["total_size"], "k8s_pod total_size should be 12288")
		assert.Equal(t, float64(300+600), k8sPodStats["total_duration"], "k8s_pod total_duration should be 900")

		// Check docker statistics
		dockerStats, ok := byType["docker"].(map[string]interface{})
		require.True(t, ok, "by_type.docker should be present")
		assert.Equal(t, float64(1), dockerStats["count"])

		// Check webssh statistics
		websshStats, ok := byType["webssh"].(map[string]interface{})
		require.True(t, ok, "by_type.webssh should be present")
		assert.Equal(t, float64(1), websshStats["count"])
	})

	t.Run("statistics filtered by cluster_id", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/recordings/stats?cluster_id=%s", clusterID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		// Should only include K8s recordings for this cluster
		totalCount := stats["total_count"].(float64)
		assert.Equal(t, 4.0, totalCount, "Should have 4 K8s recordings for cluster")

		// Verify by_type only shows k8s types
		byType := stats["by_type"].(map[string]interface{})
		assert.Contains(t, byType, "k8s_node")
		assert.Contains(t, byType, "k8s_pod")
		assert.NotContains(t, byType, "docker", "Docker recordings should not be in cluster filter")
		assert.NotContains(t, byType, "webssh", "WebSSH recordings should not be in cluster filter")
	})

	t.Run("statistics filtered by recording_type=k8s_node", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings/stats?recording_type=k8s_node", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		// Should only show k8s_node statistics
		totalCount := stats["total_count"].(float64)
		assert.Equal(t, 2.0, totalCount, "Should have 2 k8s_node recordings")

		totalSize := stats["total_size"].(float64)
		assert.Equal(t, float64(1024+2048), totalSize)

		totalDuration := stats["total_duration"].(float64)
		assert.Equal(t, float64(120+180), totalDuration)

		// by_type should only contain k8s_node
		byType := stats["by_type"].(map[string]interface{})
		assert.Len(t, byType, 1, "by_type should only have one entry")
		assert.Contains(t, byType, "k8s_node")
	})

	t.Run("statistics filtered by recording_type=k8s_pod", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings/stats?recording_type=k8s_pod", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		totalCount := stats["total_count"].(float64)
		assert.Equal(t, 2.0, totalCount, "Should have 2 k8s_pod recordings")

		totalSize := stats["total_size"].(float64)
		assert.Equal(t, float64(4096+8192), totalSize)

		totalDuration := stats["total_duration"].(float64)
		assert.Equal(t, float64(300+600), totalDuration)
	})

	t.Run("statistics with combined filters", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/recordings/stats?cluster_id=%s&recording_type=k8s_pod", clusterID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		totalCount := stats["total_count"].(float64)
		assert.Equal(t, 2.0, totalCount, "Should have 2 k8s_pod recordings in cluster")

		byType := stats["by_type"].(map[string]interface{})
		assert.Len(t, byType, 1, "Should only have k8s_pod type")
		assert.Contains(t, byType, "k8s_pod")
	})

	t.Run("verify average calculations", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/recordings/stats", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		// Check if average_size and average_duration are calculated
		if avgSize, ok := stats["average_size"].(float64); ok {
			totalSize := stats["total_size"].(float64)
			totalCount := stats["total_count"].(float64)
			expectedAvg := totalSize / totalCount
			assert.InDelta(t, expectedAvg, avgSize, 0.01, "average_size should match calculation")
		}

		if avgDuration, ok := stats["average_duration"].(float64); ok {
			totalDuration := stats["total_duration"].(float64)
			totalCount := stats["total_count"].(float64)
			expectedAvg := totalDuration / totalCount
			assert.InDelta(t, expectedAvg, avgDuration, 0.01, "average_duration should match calculation")
		}
	})

	t.Run("empty result when no recordings match filter", func(t *testing.T) {
		fakeClusterID := uuid.New().String()
		path := fmt.Sprintf("/api/v1/recordings/stats?cluster_id=%s", fakeClusterID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		totalCount := stats["total_count"].(float64)
		assert.Equal(t, 0.0, totalCount, "Should have 0 recordings for non-existent cluster")

		totalSize := stats["total_size"].(float64)
		assert.Equal(t, 0.0, totalSize)

		totalDuration := stats["total_duration"].(float64)
		assert.Equal(t, 0.0, totalDuration)
	})
}
