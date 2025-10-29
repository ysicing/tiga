package k8s_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/tests/contract"
)

// TestAuditStatsK8s tests GET /api/v1/audit/stats with K8s statistics
// Reference: 010-k8s-pod-009 T010 - Audit statistics API contract test
// MUST FAIL until K8s audit statistics aggregation is implemented
func TestAuditStatsK8s(t *testing.T) {
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
	now := time.Now()

	// Create diverse test audit events
	events := []struct {
		action       models.Action
		resourceType models.ResourceType
		subsystem    models.SubsystemType
		success      bool
	}{
		{models.ActionCreateResource, models.ResourceTypeDeployment, models.SubsystemKubernetes, true},
		{models.ActionCreateResource, models.ResourceTypeService, models.SubsystemKubernetes, true},
		{models.ActionUpdateResource, models.ResourceTypeDeployment, models.SubsystemKubernetes, true},
		{models.ActionUpdateResource, models.ResourceTypeDeployment, models.SubsystemKubernetes, false}, // Failed
		{models.ActionDeleteResource, models.ResourceTypeDeployment, models.SubsystemKubernetes, true},
		{models.ActionViewResource, models.ResourceTypePod, models.SubsystemKubernetes, true},
		{models.ActionViewResource, models.ResourceTypePod, models.SubsystemKubernetes, true},
		{models.ActionViewResource, models.ResourceTypePod, models.SubsystemKubernetes, true},
		{models.ActionNodeTerminalAccess, models.ResourceTypeK8sNode, models.SubsystemKubernetes, true},
		{models.ActionNodeTerminalAccess, models.ResourceTypeK8sNode, models.SubsystemKubernetes, true},
		{models.ActionPodTerminalAccess, models.ResourceTypeK8sPod, models.SubsystemKubernetes, true},
		{models.ActionPodTerminalAccess, models.ResourceTypeK8sPod, models.SubsystemKubernetes, false}, // Failed
		// Non-K8s events
		{models.ActionCreated, models.ResourceTypeDockerContainer, models.SubsystemDocker, true},
		{models.ActionCreated, models.ResourceTypeDockerContainer, models.SubsystemDocker, true},
	}

	for i, evt := range events {
		successStr := "true"
		if !evt.success {
			successStr = "false"
		}

		auditEvent := &models.AuditEvent{
			ID:           fmt.Sprintf("audit-%d", i),
			Timestamp:    now.Add(time.Duration(i) * time.Minute).UnixMilli(),
			Action:       evt.action,
			ResourceType: evt.resourceType,
			Resource: models.Resource{
				Type:       evt.resourceType,
				Identifier: fmt.Sprintf("resource-%d", i),
				Data: map[string]string{
					"cluster_id": clusterID,
					"success":    successStr,
				},
			},
			Subsystem: evt.subsystem,
			User: models.Principal{
				UID:      helper.AdminUser.ID.String(),
				Username: helper.AdminUser.Username,
				Type:     models.PrincipalTypeUser,
			},
			ClientIP:  "192.168.1.100",
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}
		err := helper.DB.Create(auditEvent).Error
		require.NoError(t, err)
	}

	t.Run("overall statistics include K8s events", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/audit/stats", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		// Total count should include all events
		totalCount, ok := stats["total_count"].(float64)
		require.True(t, ok, "total_count should be numeric")
		assert.GreaterOrEqual(t, totalCount, 14.0, "Should have at least 14 audit events")

		// Success rate calculation (12 successful out of 14)
		if successRate, ok := stats["success_rate"].(float64); ok {
			expectedRate := 12.0 / 14.0 * 100 // ~85.7%
			assert.InDelta(t, expectedRate, successRate, 1.0, "success_rate should be around 85.7%")
		}
	})

	t.Run("statistics by_action includes K8s actions", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/audit/stats", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		// Check by_action grouping
		byAction, ok := stats["by_action"].(map[string]interface{})
		require.True(t, ok, "by_action should be present")

		// CreateResource: 2 occurrences
		createResourceStats, ok := byAction["CreateResource"].(map[string]interface{})
		require.True(t, ok, "by_action.CreateResource should be present")
		assert.Equal(t, float64(2), createResourceStats["count"])

		// UpdateResource: 2 occurrences (1 success, 1 failure)
		updateResourceStats, ok := byAction["UpdateResource"].(map[string]interface{})
		require.True(t, ok, "by_action.UpdateResource should be present")
		assert.Equal(t, float64(2), updateResourceStats["count"])

		// DeleteResource: 1 occurrence
		deleteResourceStats, ok := byAction["DeleteResource"].(map[string]interface{})
		require.True(t, ok, "by_action.DeleteResource should be present")
		assert.Equal(t, float64(1), deleteResourceStats["count"])

		// ViewResource: 3 occurrences
		viewResourceStats, ok := byAction["ViewResource"].(map[string]interface{})
		require.True(t, ok, "by_action.ViewResource should be present")
		assert.Equal(t, float64(3), viewResourceStats["count"])

		// NodeTerminalAccess: 2 occurrences
		nodeTerminalStats, ok := byAction["NodeTerminalAccess"].(map[string]interface{})
		require.True(t, ok, "by_action.NodeTerminalAccess should be present")
		assert.Equal(t, float64(2), nodeTerminalStats["count"])

		// PodTerminalAccess: 2 occurrences (1 success, 1 failure)
		podTerminalStats, ok := byAction["PodTerminalAccess"].(map[string]interface{})
		require.True(t, ok, "by_action.PodTerminalAccess should be present")
		assert.Equal(t, float64(2), podTerminalStats["count"])
	})

	t.Run("statistics by_resource_type includes k8s_node and k8s_pod", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/audit/stats", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		// Check by_resource_type grouping
		byResourceType, ok := stats["by_resource_type"].(map[string]interface{})
		require.True(t, ok, "by_resource_type should be present")

		// k8s_node: 2 occurrences
		k8sNodeStats, ok := byResourceType["k8s_node"].(map[string]interface{})
		require.True(t, ok, "by_resource_type.k8s_node should be present")
		assert.Equal(t, float64(2), k8sNodeStats["count"])

		// k8s_pod: 2 occurrences
		k8sPodStats, ok := byResourceType["k8s_pod"].(map[string]interface{})
		require.True(t, ok, "by_resource_type.k8s_pod should be present")
		assert.Equal(t, float64(2), k8sPodStats["count"])

		// deployment: 4 occurrences
		deploymentStats, ok := byResourceType["deployment"].(map[string]interface{})
		require.True(t, ok, "by_resource_type.deployment should be present")
		assert.Equal(t, float64(4), deploymentStats["count"])

		// pod: 3 occurrences
		podStats, ok := byResourceType["pod"].(map[string]interface{})
		require.True(t, ok, "by_resource_type.pod should be present")
		assert.Equal(t, float64(3), podStats["count"])
	})

	t.Run("statistics filtered by subsystem=kubernetes", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/audit/stats?subsystem=kubernetes", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		// Should only include K8s events (12 total)
		totalCount := stats["total_count"].(float64)
		assert.Equal(t, 12.0, totalCount, "Should have 12 K8s audit events")

		// by_action should only show K8s actions
		byAction := stats["by_action"].(map[string]interface{})
		assert.Contains(t, byAction, "CreateResource")
		assert.Contains(t, byAction, "UpdateResource")
		assert.Contains(t, byAction, "DeleteResource")
		assert.Contains(t, byAction, "ViewResource")
		assert.Contains(t, byAction, "NodeTerminalAccess")
		assert.Contains(t, byAction, "PodTerminalAccess")

		// Should not contain Docker actions
		assert.NotContains(t, byAction, "created")
	})

	t.Run("statistics filtered by cluster_id", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/audit/stats?cluster_id=%s", clusterID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		// All test events belong to this cluster (14 total)
		totalCount := stats["total_count"].(float64)
		assert.Equal(t, 14.0, totalCount)
	})

	t.Run("success_rate calculation", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/audit/stats?subsystem=kubernetes", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		// K8s events: 10 successful + 2 failed = 12 total
		// Success rate: 10/12 = 83.33%
		successRate, ok := stats["success_rate"].(float64)
		require.True(t, ok, "success_rate should be present")
		assert.InDelta(t, 83.33, successRate, 1.0, "K8s success rate should be around 83.33%")
	})

	t.Run("statistics by time range", func(t *testing.T) {
		// Query last 10 minutes
		startTime := now.Add(-1 * time.Hour).UnixMilli()
		endTime := now.Add(15 * time.Minute).UnixMilli()

		path := fmt.Sprintf("/api/v1/audit/stats?start_time=%d&end_time=%d", startTime, endTime)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		// All events should be within this range
		totalCount := stats["total_count"].(float64)
		assert.GreaterOrEqual(t, totalCount, 14.0)
	})

	t.Run("statistics with combined filters", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/audit/stats?subsystem=kubernetes&action=NodeTerminalAccess")
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		// Should only have NodeTerminalAccess events (2 total)
		totalCount := stats["total_count"].(float64)
		assert.Equal(t, 2.0, totalCount)

		// by_action should only show NodeTerminalAccess
		byAction := stats["by_action"].(map[string]interface{})
		assert.Len(t, byAction, 1, "Should only have one action type")
		assert.Contains(t, byAction, "NodeTerminalAccess")
	})

	t.Run("verify by_action shows success counts", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/audit/stats", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		byAction := stats["by_action"].(map[string]interface{})

		// UpdateResource had 2 events: 1 success, 1 failure
		updateStats := byAction["UpdateResource"].(map[string]interface{})
		if successCount, ok := updateStats["success_count"].(float64); ok {
			assert.Equal(t, 1.0, successCount, "UpdateResource should have 1 successful event")
		}
		if failureCount, ok := updateStats["failure_count"].(float64); ok {
			assert.Equal(t, 1.0, failureCount, "UpdateResource should have 1 failed event")
		}

		// PodTerminalAccess had 2 events: 1 success, 1 failure
		podTerminalStats := byAction["PodTerminalAccess"].(map[string]interface{})
		if successCount, ok := podTerminalStats["success_count"].(float64); ok {
			assert.Equal(t, 1.0, successCount)
		}
		if failureCount, ok := podTerminalStats["failure_count"].(float64); ok {
			assert.Equal(t, 1.0, failureCount)
		}
	})

	t.Run("empty result when no events match filter", func(t *testing.T) {
		fakeClusterID := uuid.New().String()
		path := fmt.Sprintf("/api/v1/audit/stats?cluster_id=%s", fakeClusterID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		stats := data["data"].(map[string]interface{})

		totalCount := stats["total_count"].(float64)
		assert.Equal(t, 0.0, totalCount, "Should have 0 events for non-existent cluster")
	})
}
