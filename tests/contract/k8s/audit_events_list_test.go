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

// TestListAuditEventsK8sFilter tests GET /api/v1/audit/events with K8s subsystem filtering
// Reference: 010-k8s-pod-009 T008 - Audit events list API contract test
// MUST FAIL until K8s audit event filtering is implemented
func TestListAuditEventsK8sFilter(t *testing.T) {
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

	// Create test audit events
	events := []struct {
		subsystem    models.SubsystemType
		action       models.Action
		resourceType models.ResourceType
		resourceData map[string]string
	}{
		{
			subsystem:    models.SubsystemKubernetes,
			action:       models.ActionCreateResource,
			resourceType: models.ResourceTypeDeployment,
			resourceData: map[string]string{
				"cluster_id":    clusterID,
				"namespace":     "default",
				"resource_name": "nginx-deployment",
				"success":       "true",
			},
		},
		{
			subsystem:    models.SubsystemKubernetes,
			action:       models.ActionUpdateResource,
			resourceType: models.ResourceTypeDeployment,
			resourceData: map[string]string{
				"cluster_id":     clusterID,
				"namespace":      "default",
				"resource_name":  "nginx-deployment",
				"change_summary": "Updated replicas from 3 to 5",
				"success":        "true",
			},
		},
		{
			subsystem:    models.SubsystemKubernetes,
			action:       models.ActionDeleteResource,
			resourceType: models.ResourceTypeDeployment,
			resourceData: map[string]string{
				"cluster_id":    clusterID,
				"namespace":     "default",
				"resource_name": "nginx-deployment",
				"success":       "true",
			},
		},
		{
			subsystem:    models.SubsystemKubernetes,
			action:       models.ActionNodeTerminalAccess,
			resourceType: models.ResourceTypeK8sNode,
			resourceData: map[string]string{
				"cluster_id":    clusterID,
				"resource_name": "worker-node-1",
				"recording_id":  uuid.New().String(),
				"success":       "true",
			},
		},
		{
			subsystem:    models.SubsystemKubernetes,
			action:       models.ActionPodTerminalAccess,
			resourceType: models.ResourceTypeK8sPod,
			resourceData: map[string]string{
				"cluster_id":     clusterID,
				"namespace":      "kube-system",
				"pod_name":       "coredns-abc",
				"container_name": "coredns",
				"recording_id":   uuid.New().String(),
				"success":        "true",
			},
		},
		{
			subsystem:    models.SubsystemKubernetes,
			action:       models.ActionViewResource,
			resourceType: models.ResourceTypePod,
			resourceData: map[string]string{
				"cluster_id":    clusterID,
				"namespace":     "default",
				"resource_name": "nginx-pod",
				"success":       "true",
			},
		},
		// Non-K8s event for comparison
		{
			subsystem:    models.SubsystemDocker,
			action:       models.ActionCreated,
			resourceType: models.ResourceTypeDockerContainer,
			resourceData: map[string]string{
				"instance_id":    "docker-001",
				"container_name": "test-container",
				"success":        "true",
			},
		},
	}

	for i, evt := range events {
		auditEvent := &models.AuditEvent{
			ID:           fmt.Sprintf("audit-%d", i),
			Timestamp:    now.Add(time.Duration(i) * time.Minute).UnixMilli(),
			Action:       evt.action,
			ResourceType: evt.resourceType,
			Resource: models.Resource{
				Type:       evt.resourceType,
				Identifier: fmt.Sprintf("resource-%d", i),
				Data:       evt.resourceData,
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

	t.Run("filter by subsystem=kubernetes", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/audit/events?subsystem=kubernetes", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		events := responseData["events"].([]interface{})

		// Should find 6 K8s events (not the Docker event)
		assert.GreaterOrEqual(t, len(events), 6, "Should have at least 6 K8s audit events")

		// Verify all events are K8s subsystem
		for _, e := range events {
			event := e.(map[string]interface{})
			assert.Equal(t, string(models.SubsystemKubernetes), event["subsystem"])
		}
	})

	t.Run("filter by action=CreateResource", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/audit/events?action=CreateResource", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		events := responseData["events"].([]interface{})

		assert.GreaterOrEqual(t, len(events), 1, "Should have at least 1 CreateResource event")

		for _, e := range events {
			event := e.(map[string]interface{})
			assert.Equal(t, string(models.ActionCreateResource), event["action"])
		}
	})

	t.Run("filter by action=NodeTerminalAccess", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/audit/events?action=NodeTerminalAccess", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		events := responseData["events"].([]interface{})

		assert.GreaterOrEqual(t, len(events), 1, "Should have at least 1 NodeTerminalAccess event")

		for _, e := range events {
			event := e.(map[string]interface{})
			assert.Equal(t, string(models.ActionNodeTerminalAccess), event["action"])

			// Verify resource_type is k8s_node
			assert.Equal(t, string(models.ResourceTypeK8sNode), event["resource_type"])

			// Verify resource.data contains recording_id
			resource := event["resource"].(map[string]interface{})
			resourceData := resource["data"].(map[string]interface{})
			assert.NotEmpty(t, resourceData["recording_id"], "recording_id should be present")
		}
	})

	t.Run("filter by resource_type=k8s_pod", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/audit/events?resource_type=k8s_pod", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		events := responseData["events"].([]interface{})

		assert.GreaterOrEqual(t, len(events), 1, "Should have at least 1 k8s_pod event")

		for _, e := range events {
			event := e.(map[string]interface{})
			assert.Equal(t, string(models.ResourceTypeK8sPod), event["resource_type"])

			// Verify resource.data structure
			resource := event["resource"].(map[string]interface{})
			resourceData := resource["data"].(map[string]interface{})
			assert.NotEmpty(t, resourceData["cluster_id"])
			assert.NotEmpty(t, resourceData["namespace"])
			assert.NotEmpty(t, resourceData["pod_name"])
			assert.NotEmpty(t, resourceData["container_name"])
		}
	})

	t.Run("filter by cluster_id", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/audit/events?cluster_id=%s", clusterID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		events := responseData["events"].([]interface{})

		// Should find all 6 K8s events for this cluster
		assert.GreaterOrEqual(t, len(events), 6, "Should have at least 6 events for cluster")

		// Verify all events belong to this cluster
		for _, e := range events {
			event := e.(map[string]interface{})
			resource := event["resource"].(map[string]interface{})
			resourceData := resource["data"].(map[string]interface{})
			assert.Equal(t, clusterID, resourceData["cluster_id"])
		}
	})

	t.Run("combined filters - subsystem and action", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/audit/events?subsystem=kubernetes&action=UpdateResource", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		events := responseData["events"].([]interface{})

		assert.GreaterOrEqual(t, len(events), 1, "Should have UpdateResource events")

		for _, e := range events {
			event := e.(map[string]interface{})
			assert.Equal(t, string(models.SubsystemKubernetes), event["subsystem"])
			assert.Equal(t, string(models.ActionUpdateResource), event["action"])

			// Verify change_summary is present for update events
			resource := event["resource"].(map[string]interface{})
			resourceData := resource["data"].(map[string]interface{})
			if changeSum, ok := resourceData["change_summary"]; ok {
				assert.NotEmpty(t, changeSum)
			}
		}
	})

	t.Run("combined filters - cluster, namespace and action", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/audit/events?cluster_id=%s&namespace=default&action=CreateResource", clusterID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		events := responseData["events"].([]interface{})

		assert.GreaterOrEqual(t, len(events), 1, "Should have CreateResource events in default namespace")

		for _, e := range events {
			event := e.(map[string]interface{})
			resource := event["resource"].(map[string]interface{})
			resourceData := resource["data"].(map[string]interface{})

			assert.Equal(t, clusterID, resourceData["cluster_id"])
			assert.Equal(t, "default", resourceData["namespace"])
			assert.Equal(t, string(models.ActionCreateResource), event["action"])
		}
	})

	t.Run("pagination with K8s filters", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/audit/events?subsystem=kubernetes&page=1&limit=10", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})

		// Verify pagination structure
		pagination, ok := responseData["pagination"].(map[string]interface{})
		require.True(t, ok, "pagination should be present")
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(10), pagination["limit"])
	})

	t.Run("verify response structure", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/audit/events?subsystem=kubernetes&limit=1", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		events := responseData["events"].([]interface{})

		require.GreaterOrEqual(t, len(events), 1, "Should have at least 1 event")

		event := events[0].(map[string]interface{})

		// Verify event structure
		assert.NotEmpty(t, event["id"])
		assert.NotEmpty(t, event["timestamp"])
		assert.NotEmpty(t, event["action"])
		assert.NotEmpty(t, event["resource_type"])
		assert.NotEmpty(t, event["subsystem"])
		assert.NotEmpty(t, event["client_ip"])
		assert.NotEmpty(t, event["created_at"])

		// Verify resource structure
		resource, ok := event["resource"].(map[string]interface{})
		require.True(t, ok, "resource should be present")
		assert.NotEmpty(t, resource["type"])
		assert.NotEmpty(t, resource["identifier"])
		assert.NotNil(t, resource["data"])

		// Verify user structure
		user, ok := event["user"].(map[string]interface{})
		require.True(t, ok, "user should be present")
		assert.NotEmpty(t, user["uid"])
		assert.NotEmpty(t, user["username"])
		assert.NotEmpty(t, user["type"])
	})

	t.Run("sort by timestamp descending (newest first)", func(t *testing.T) {
		resp, err := helper.MakeRequest(http.MethodGet, "/api/v1/audit/events?subsystem=kubernetes&order=desc", nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		responseData := data["data"].(map[string]interface{})
		events := responseData["events"].([]interface{})

		require.GreaterOrEqual(t, len(events), 2, "Should have at least 2 events")

		// Verify events are sorted by timestamp descending
		for i := 0; i < len(events)-1; i++ {
			event1 := events[i].(map[string]interface{})
			event2 := events[i+1].(map[string]interface{})

			timestamp1 := event1["timestamp"].(float64)
			timestamp2 := event2["timestamp"].(float64)

			assert.GreaterOrEqual(t, timestamp1, timestamp2, "Events should be sorted by timestamp descending")
		}
	})
}
