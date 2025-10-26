package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// AuditHelper provides helper functions for Docker operation audit logging
// T036-T037: Migrated to use unified AuditEvent model
type AuditHelper struct {
	auditEventRepo repository.AuditEventRepository
}

// NewAuditHelper creates a new AuditHelper using unified audit system
func NewAuditHelper(auditEventRepo repository.AuditEventRepository) *AuditHelper {
	return &AuditHelper{
		auditEventRepo: auditEventRepo,
	}
}

// AuditParams contains parameters for creating an audit log
type AuditParams struct {
	Action       string                 // Docker action (use models.DockerAction* constants)
	ResourceType string                 // Resource type (use models.DockerResourceType* constants)
	ResourceID   uuid.UUID              // Resource ID (instance/container/image ID)
	ResourceName string                 // Resource name
	InstanceID   uuid.UUID              // Docker instance ID
	InstanceName string                 // Docker instance name
	AgentID      *uuid.UUID             // Agent ID (optional)
	ExtraData    map[string]interface{} // Additional data (optional)
	Error        error                  // Error if operation failed (optional)
	Duration     int64                  // Operation duration in milliseconds (optional)
}

// LogDockerOperation logs a Docker operation to the unified audit system
// This is a non-blocking operation that runs in a goroutine
func (h *AuditHelper) LogDockerOperation(c *gin.Context, params AuditParams) {
	// Extract user info from context
	userID, username := h.extractUserInfo(c)

	// Map Docker operation to generic Action
	genericAction := mapDockerOperationToAction(params.Action)

	// Build metadata map for Docker-specific data
	metadata := make(map[string]string)
	// Store the detailed Docker operation in metadata
	metadata["operation"] = params.Action

	if params.InstanceID != uuid.Nil {
		metadata["instance_id"] = params.InstanceID.String()
	}
	if params.InstanceName != "" {
		metadata["instance_name"] = params.InstanceName
	}
	if params.AgentID != nil {
		metadata["agent_id"] = params.AgentID.String()
	}
	if params.Duration > 0 {
		metadata["duration_ms"] = fmt.Sprintf("%d", params.Duration)
	}

	// Add extra data to metadata
	if params.ExtraData != nil {
		for k, v := range params.ExtraData {
			if strVal, ok := v.(string); ok {
				metadata[k] = strVal
			} else {
				// Convert non-string values to JSON
				if jsonBytes, err := json.Marshal(v); err == nil {
					metadata[k] = string(jsonBytes)
				}
			}
		}
	}

	// Create unified audit event
	event := &models.AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UnixMilli(),

		// Action and Resource (use generic action, store detailed operation in metadata)
		Action:       genericAction,
		ResourceType: models.ResourceType(params.ResourceType),
		Resource: models.Resource{
			Type:       models.ResourceType(params.ResourceType),
			Identifier: params.ResourceID.String(),
			Data:       map[string]string{"resource_name": params.ResourceName},
		},

		// Subsystem
		Subsystem: models.SubsystemDocker,

		// User
		User: models.Principal{
			UID:      userID.String(),
			Username: username,
			Type:     models.PrincipalTypeUser,
		},

		// Client info
		ClientIP:      c.ClientIP(),
		UserAgent:     c.Request.UserAgent(),
		RequestMethod: c.Request.Method,
		RequestID:     c.GetString("request_id"),

		// Metadata
		Data: metadata,
	}

	// Add error if present
	if params.Error != nil {
		// Store error in metadata
		event.Data["error"] = params.Error.Error()
		event.Data["success"] = "false"
	} else {
		event.Data["success"] = "true"
	}

	// Log asynchronously to avoid blocking the request
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := h.auditEventRepo.Create(ctx, event); err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"action":        genericAction,
				"operation":     params.Action,
				"resource_type": params.ResourceType,
				"resource_id":   params.ResourceID,
				"user_id":       userID,
			}).Error("Docker: failed to write audit log")
		}
	}()
}

// mapDockerOperationToAction maps detailed Docker operations to generic actions
// Following the unified audit system pattern: use generic actions (created/updated/deleted/read)
// and store detailed operation type in metadata["operation"]
func mapDockerOperationToAction(dockerOperation string) models.Action {
	// List and Get operations -> read
	if isReadOperation(dockerOperation) {
		return models.ActionRead
	}

	// Create operations -> created
	if isCreateOperation(dockerOperation) {
		return models.ActionCreated
	}

	// Delete operations -> deleted
	if isDeleteOperation(dockerOperation) {
		return models.ActionDeleted
	}

	// All other operations (start, stop, restart, etc.) -> updated
	return models.ActionUpdated
}

// isReadOperation checks if the operation is a read-only operation
func isReadOperation(operation string) bool {
	readOps := []string{
		"list_instances", "get_instance",
		"list_containers", "get_container", "get_container_logs", "get_container_stats",
		"list_images", "get_image",
		"list_volumes", "get_volume",
		"list_networks", "get_network",
		"get_system_info", "get_docker_info", "get_version", "get_disk_usage", "ping",
		"list_recordings", "get_recording", "get_recording_playback", "get_recording_statistics",
		"test_connection", // Connection test is also read-only
	}

	for _, op := range readOps {
		if operation == op {
			return true
		}
	}
	return false
}

// isCreateOperation checks if the operation is a create operation
func isCreateOperation(operation string) bool {
	createOps := []string{
		"create_instance",
		"pull_image", // Pulling image creates a new local image
		"create_volume",
		"create_network",
	}

	for _, op := range createOps {
		if operation == op {
			return true
		}
	}
	return false
}

// isDeleteOperation checks if the operation is a delete operation
func isDeleteOperation(operation string) bool {
	deleteOps := []string{
		"delete_instance",
		"delete_container",
		"delete_image",
		"delete_volume", "prune_volumes",
		"delete_network",
		"delete_recording",
	}

	for _, op := range deleteOps {
		if operation == op {
			return true
		}
	}
	return false
}

// LogListOperation is a convenience method for logging list operations (read-only)
func (h *AuditHelper) LogListOperation(c *gin.Context, action, resourceType string, instanceID uuid.UUID, instanceName string, count int, err error) {
	extraData := map[string]interface{}{
		"count": count,
	}

	// For list operations, determine scope
	scope := "instance" // Default: listing resources within an instance
	if instanceID == uuid.Nil {
		scope = "global" // Listing all instances globally
		extraData["scope"] = scope
	}

	h.LogDockerOperation(c, AuditParams{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   instanceID, // uuid.Nil for global list, instance ID for scoped list
		ResourceName: instanceName,
		InstanceID:   instanceID,
		InstanceName: instanceName,
		ExtraData:    extraData,
		Error:        err,
	})
}

// LogGetOperation is a convenience method for logging get operations (read-only)
func (h *AuditHelper) LogGetOperation(c *gin.Context, action, resourceType string, resourceID uuid.UUID, resourceName string, instanceID uuid.UUID, instanceName string, err error) {
	h.LogDockerOperation(c, AuditParams{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		InstanceID:   instanceID,
		InstanceName: instanceName,
		Error:        err,
	})
}

// LogMutationOperation is a convenience method for logging mutation operations (create/update/delete)
func (h *AuditHelper) LogMutationOperation(c *gin.Context, action, resourceType string, resourceID uuid.UUID, resourceName string, instanceID uuid.UUID, instanceName string, duration int64, err error) {
	h.LogDockerOperation(c, AuditParams{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		InstanceID:   instanceID,
		InstanceName: instanceName,
		Duration:     duration,
		Error:        err,
	})
}

// extractUserInfo extracts user ID and username from Gin context
// Returns zero UUID and "anonymous" if user is not authenticated
func (h *AuditHelper) extractUserInfo(c *gin.Context) (uuid.UUID, string) {
	// Try to get user ID from context (set by auth middleware)
	userIDValue, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, "anonymous"
	}

	// Parse user ID
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		// Try to parse as string
		if userIDStr, ok := userIDValue.(string); ok {
			parsed, err := uuid.Parse(userIDStr)
			if err == nil {
				userID = parsed
			}
		}
	}

	// Get username from context
	username := "unknown"
	if usernameValue, exists := c.Get("username"); exists {
		if usernameStr, ok := usernameValue.(string); ok {
			username = usernameStr
		}
	}

	return userID, username
}
