package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	k8sservice "github.com/ysicing/tiga/internal/services/k8s"
)

// K8sAuditContext key for storing audit context in gin.Context
const K8sAuditContext = "k8s_audit_context"

// K8sAudit creates a middleware for K8s resource operation auditing
// Reference: 010-k8s-pod-009 T022
//
// Intercepts /api/v1/k8s/* routes
// Records audit logs for POST/PUT/PATCH/DELETE operations
// GET operations are audited by handlers internally
func K8sAudit(auditService *k8sservice.K8sAuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only audit K8s routes
		if !strings.HasPrefix(c.Request.URL.Path, "/api/v1/k8s/") {
			c.Next()
			return
		}

		// Extract user from context (set by auth middleware)
		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")

		// Prepare audit context
		auditCtx := map[string]interface{}{
			"user_id":    userID,
			"username":   username,
			"client_ip":  c.ClientIP(),
			"request_id": uuid.New().String(),
			"start_time": time.Now(),
		}

		// Store audit context
		c.Set(K8sAuditContext, auditCtx)

		// Execute request
		c.Next()

		// Only audit modification operations (not GET)
		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			return // Read operations audited by handlers
		}

		// Map HTTP method to Action
		var action models.Action
		switch method {
		case "POST":
			action = models.ActionCreateResource
		case "PUT", "PATCH":
			action = models.ActionUpdateResource
		case "DELETE":
			action = models.ActionDeleteResource
		default:
			return // Unknown method, skip audit
		}

		// Extract resource information from response or context
		clusterID, _ := c.Get("cluster_id")
		namespace, _ := c.Get("namespace")
		resourceType, _ := c.Get("resource_type")
		resourceName, _ := c.Get("resource_name")

		// Check if operation succeeded (status < 400)
		success := c.Writer.Status() < 400

		// Create audit log
		log := &k8sservice.ResourceOperationLog{
			ClusterID:    toString(clusterID),
			Namespace:    toString(namespace),
			ResourceType: toResourceType(resourceType),
			ResourceName: toString(resourceName),
			Action:       action,
			UserID:       toString(userID),
			Username:     toString(username),
			ClientIP:     c.ClientIP(),
			RequestID:    toString(auditCtx["request_id"]),
			Success:      success,
		}

		// Log audit asynchronously (fire and forget)
		go auditService.LogResourceOperation(c.Request.Context(), log)
	}
}

// Helper functions

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func toResourceType(v interface{}) models.ResourceType {
	if v == nil {
		return ""
	}
	if rt, ok := v.(models.ResourceType); ok {
		return rt
	}
	if s, ok := v.(string); ok {
		return models.ResourceType(s)
	}
	return ""
}
