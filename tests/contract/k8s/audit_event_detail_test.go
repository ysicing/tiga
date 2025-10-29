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

// TestGetAuditEventDetailK8s tests GET /api/v1/audit/events/:id with K8s event details
// Reference: 010-k8s-pod-009 T009 - Audit event detail API contract test
// MUST FAIL until K8s audit event detail serialization is implemented
func TestGetAuditEventDetailK8s(t *testing.T) {
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

	t.Run("resource creation event detail", func(t *testing.T) {
		event := &models.AuditEvent{
			ID:           "audit-create-1",
			Timestamp:    now.UnixMilli(),
			Action:       models.ActionCreateResource,
			ResourceType: models.ResourceTypeDeployment,
			Resource: models.Resource{
				Type:       models.ResourceTypeDeployment,
				Identifier: "nginx-deployment",
				Data: map[string]string{
					"cluster_id":    clusterID,
					"namespace":     "production",
					"resource_name": "nginx-deployment",
					"replicas":      "3",
					"image":         "nginx:1.21",
					"success":       "true",
				},
			},
			Subsystem: models.SubsystemKubernetes,
			User: models.Principal{
				UID:      helper.AdminUser.ID.String(),
				Username: helper.AdminUser.Username,
				Type:     models.PrincipalTypeUser,
			},
			ClientIP:  "192.168.1.100",
			CreatedAt: now,
		}
		err := helper.DB.Create(event).Error
		require.NoError(t, err)

		// Request detail
		path := fmt.Sprintf("/api/v1/audit/events/%s", event.ID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		auditEvent := data["data"].(map[string]interface{})

		// Verify basic structure
		assert.Equal(t, event.ID, auditEvent["id"])
		assert.Equal(t, string(models.ActionCreateResource), auditEvent["action"])
		assert.Equal(t, string(models.ResourceTypeDeployment), auditEvent["resource_type"])
		assert.Equal(t, string(models.SubsystemKubernetes), auditEvent["subsystem"])

		// Verify resource.data structure
		resource := auditEvent["resource"].(map[string]interface{})
		resourceData := resource["data"].(map[string]interface{})

		assert.Equal(t, clusterID, resourceData["cluster_id"])
		assert.Equal(t, "production", resourceData["namespace"])
		assert.Equal(t, "nginx-deployment", resourceData["resource_name"])
		assert.Equal(t, "3", resourceData["replicas"])
		assert.Equal(t, "nginx:1.21", resourceData["image"])
		assert.Equal(t, "true", resourceData["success"])
	})

	t.Run("resource update event with diff_object", func(t *testing.T) {
		oldDeployment := map[string]interface{}{
			"replicas": 3,
			"image":    "nginx:1.20",
		}
		newDeployment := map[string]interface{}{
			"replicas": 5,
			"image":    "nginx:1.21",
		}

		event := &models.AuditEvent{
			ID:           "audit-update-1",
			Timestamp:    now.UnixMilli(),
			Action:       models.ActionUpdateResource,
			ResourceType: models.ResourceTypeDeployment,
			Resource: models.Resource{
				Type:       models.ResourceTypeDeployment,
				Identifier: "api-deployment",
				Data: map[string]string{
					"cluster_id":     clusterID,
					"namespace":      "production",
					"resource_name":  "api-deployment",
					"change_summary": "Updated replicas from 3 to 5, image from nginx:1.20 to nginx:1.21",
					"success":        "true",
				},
			},
			Subsystem: models.SubsystemKubernetes,
			User: models.Principal{
				UID:      helper.AdminUser.ID.String(),
				Username: helper.AdminUser.Username,
				Type:     models.PrincipalTypeUser,
			},
			ClientIP:  "192.168.1.100",
			CreatedAt: now,
		}

		// Set diff_object
		err := event.MarshalOldObject(oldDeployment)
		require.NoError(t, err)
		err = event.MarshalNewObject(newDeployment)
		require.NoError(t, err)

		err = helper.DB.Create(event).Error
		require.NoError(t, err)

		// Request detail
		path := fmt.Sprintf("/api/v1/audit/events/%s", event.ID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		auditEvent := data["data"].(map[string]interface{})

		// Verify diff_object is present
		diffObject, ok := auditEvent["diff_object"].(map[string]interface{})
		require.True(t, ok, "diff_object should be present for update events")

		assert.NotEmpty(t, diffObject["old_object"], "old_object should be present")
		assert.NotEmpty(t, diffObject["new_object"], "new_object should be present")

		// Verify change_summary
		resource := auditEvent["resource"].(map[string]interface{})
		resourceData := resource["data"].(map[string]interface{})
		assert.Contains(t, resourceData["change_summary"], "Updated replicas")
	})

	t.Run("terminal access event with recording_id", func(t *testing.T) {
		recordingID := uuid.New().String()

		event := &models.AuditEvent{
			ID:           "audit-terminal-1",
			Timestamp:    now.UnixMilli(),
			Action:       models.ActionNodeTerminalAccess,
			ResourceType: models.ResourceTypeK8sNode,
			Resource: models.Resource{
				Type:       models.ResourceTypeK8sNode,
				Identifier: "worker-node-1",
				Data: map[string]string{
					"cluster_id":    clusterID,
					"resource_name": "worker-node-1",
					"recording_id":  recordingID,
					"duration":      "180",
					"success":       "true",
				},
			},
			Subsystem: models.SubsystemKubernetes,
			User: models.Principal{
				UID:      helper.AdminUser.ID.String(),
				Username: helper.AdminUser.Username,
				Type:     models.PrincipalTypeUser,
			},
			ClientIP:  "192.168.1.100",
			CreatedAt: now,
		}
		err := helper.DB.Create(event).Error
		require.NoError(t, err)

		// Request detail
		path := fmt.Sprintf("/api/v1/audit/events/%s", event.ID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		auditEvent := data["data"].(map[string]interface{})

		// Verify action and resource_type
		assert.Equal(t, string(models.ActionNodeTerminalAccess), auditEvent["action"])
		assert.Equal(t, string(models.ResourceTypeK8sNode), auditEvent["resource_type"])

		// Verify recording_id is present
		resource := auditEvent["resource"].(map[string]interface{})
		resourceData := resource["data"].(map[string]interface{})
		assert.Equal(t, recordingID, resourceData["recording_id"])
		assert.Equal(t, "180", resourceData["duration"])
	})

	t.Run("pod terminal access event with full metadata", func(t *testing.T) {
		recordingID := uuid.New().String()

		event := &models.AuditEvent{
			ID:           "audit-pod-terminal-1",
			Timestamp:    now.UnixMilli(),
			Action:       models.ActionPodTerminalAccess,
			ResourceType: models.ResourceTypeK8sPod,
			Resource: models.Resource{
				Type:       models.ResourceTypeK8sPod,
				Identifier: "api-pod-abc",
				Data: map[string]string{
					"cluster_id":     clusterID,
					"namespace":      "production",
					"pod_name":       "api-pod-abc",
					"container_name": "api",
					"recording_id":   recordingID,
					"duration":       "600",
					"success":        "true",
				},
			},
			Subsystem: models.SubsystemKubernetes,
			User: models.Principal{
				UID:      helper.AdminUser.ID.String(),
				Username: helper.AdminUser.Username,
				Type:     models.PrincipalTypeUser,
			},
			ClientIP:  "192.168.1.105",
			CreatedAt: now,
		}
		err := helper.DB.Create(event).Error
		require.NoError(t, err)

		// Request detail
		path := fmt.Sprintf("/api/v1/audit/events/%s", event.ID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		auditEvent := data["data"].(map[string]interface{})

		// Verify full pod metadata
		resource := auditEvent["resource"].(map[string]interface{})
		resourceData := resource["data"].(map[string]interface{})

		assert.Equal(t, clusterID, resourceData["cluster_id"])
		assert.Equal(t, "production", resourceData["namespace"])
		assert.Equal(t, "api-pod-abc", resourceData["pod_name"])
		assert.Equal(t, "api", resourceData["container_name"])
		assert.Equal(t, recordingID, resourceData["recording_id"])
		assert.Equal(t, "600", resourceData["duration"])
	})

	t.Run("view resource event (read-only operation)", func(t *testing.T) {
		event := &models.AuditEvent{
			ID:           "audit-view-1",
			Timestamp:    now.UnixMilli(),
			Action:       models.ActionViewResource,
			ResourceType: models.ResourceTypePod,
			Resource: models.Resource{
				Type:       models.ResourceTypePod,
				Identifier: "nginx-pod",
				Data: map[string]string{
					"cluster_id":    clusterID,
					"namespace":     "default",
					"resource_name": "nginx-pod",
					"operation":     "get",
					"success":       "true",
				},
			},
			Subsystem: models.SubsystemKubernetes,
			User: models.Principal{
				UID:      helper.AdminUser.ID.String(),
				Username: "developer",
				Type:     models.PrincipalTypeUser,
			},
			ClientIP:  "192.168.1.110",
			CreatedAt: now,
		}
		err := helper.DB.Create(event).Error
		require.NoError(t, err)

		// Request detail
		path := fmt.Sprintf("/api/v1/audit/events/%s", event.ID)
		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		data := helper.AssertSuccessResponse(t, resp)
		auditEvent := data["data"].(map[string]interface{})

		// Verify this is a read-only operation
		assert.Equal(t, string(models.ActionViewResource), auditEvent["action"])

		// Verify no diff_object for read operations
		if diffObj, exists := auditEvent["diff_object"]; exists {
			diffObject := diffObj.(map[string]interface{})
			// Should be empty or null for read operations
			assert.Empty(t, diffObject["old_object"])
			assert.Empty(t, diffObject["new_object"])
		}
	})

	t.Run("nonexistent event returns 404", func(t *testing.T) {
		fakeID := "nonexistent-audit-id"
		path := fmt.Sprintf("/api/v1/audit/events/%s", fakeID)

		resp, err := helper.MakeRequest(http.MethodGet, path, nil)
		require.NoError(t, err)

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
