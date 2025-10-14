package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// AuditMiddleware handles audit logging
type AuditMiddleware struct {
	auditRepo *repository.AuditLogRepository
}

// NewAuditMiddleware creates a new audit middleware
func NewAuditMiddleware(auditRepo *repository.AuditLogRepository) *AuditMiddleware {
	return &AuditMiddleware{
		auditRepo: auditRepo,
	}
}

// AuditLog returns an audit logging middleware
func (m *AuditMiddleware) AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip audit logging for certain paths
		if shouldSkipAudit(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()

		// Capture request body if needed
		var requestBody []byte
		if c.Request.Body != nil && shouldLogBody(c.Request.Method) {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Process request
		c.Next()

		// Create audit log entry
		go func() {
			auditLog := m.buildAuditLog(c, start, requestBody)
			if auditLog != nil {
				if err := m.auditRepo.Create(c.Request.Context(), auditLog); err != nil {
					logrus.Errorf("Failed to create audit log: %v", err)
				}
			}
		}()
	}
}

// buildAuditLog builds an audit log entry
func (m *AuditMiddleware) buildAuditLog(c *gin.Context, start time.Time, requestBody []byte) *models.AuditLog {
	// Get user ID if authenticated
	var userID *uuid.UUID
	if uid, err := GetUserID(c); err == nil {
		userID = &uid
	}

	// Determine action from method and path
	action := determineAction(c.Request.Method, c.Request.URL.Path)

	// Determine resource type and ID from path
	resourceType, resourceID := extractResource(c.Request.URL.Path)

	// Determine status
	status := "success"
	if c.Writer.Status() >= 400 {
		status = "failure"
	}

	// Build changes object
	changes := make(map[string]interface{})
	if len(requestBody) > 0 && len(requestBody) < 10000 { // Limit size
		var bodyData map[string]interface{}
		if err := json.Unmarshal(requestBody, &bodyData); err == nil {
			changes["request"] = bodyData
		}
	}

	// Add response status
	changes["status_code"] = c.Writer.Status()

	// Add errors if any
	if len(c.Errors) > 0 {
		changes["errors"] = c.Errors.String()
	}

	auditLog := &models.AuditLog{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		Status:       status,
		Description:  buildDescription(c),
		Changes:      changes,
	}

	return auditLog
}

// shouldSkipAudit determines if audit logging should be skipped
func shouldSkipAudit(path string) bool {
	skipPaths := []string{
		"/health",
		"/ready",
		"/metrics",
		"/api/v1/auth/refresh",
	}

	for _, skip := range skipPaths {
		if path == skip {
			return true
		}
	}

	return false
}

// shouldLogBody determines if request body should be logged
func shouldLogBody(method string) bool {
	return method == "POST" || method == "PUT" || method == "PATCH"
}

// determineAction determines the action from method and path
func determineAction(method, path string) string {
	switch method {
	case "GET":
		return "read"
	case "POST":
		return "create"
	case "PUT", "PATCH":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return method
	}
}

// extractResource extracts resource type and ID from path
func extractResource(path string) (string, *uuid.UUID) {
	// This is a simplified implementation
	// In practice, you would parse the path to extract resource info
	// Example: /api/v1/dbs/uuid -> resourceType: dbs, resourceID: uuid

	parts := splitPath(path)
	if len(parts) >= 4 {
		resourceType := parts[3]
		if len(parts) >= 5 {
			if id, err := uuid.Parse(parts[4]); err == nil {
				return resourceType, &id
			}
		}
		return resourceType, nil
	}

	return "unknown", nil
}

// splitPath splits a path into parts
func splitPath(path string) []string {
	var parts []string
	current := ""
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(path[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// buildDescription builds a human-readable description
func buildDescription(c *gin.Context) string {
	method := c.Request.Method
	path := c.Request.URL.Path
	statusCode := c.Writer.Status()

	description := method + " " + path

	if statusCode >= 400 {
		description += " (failed)"
	}

	return description
}

// AuditCreate is a helper to log create actions
func AuditCreate(c *gin.Context, resourceType string, resourceID uuid.UUID, description string) {
	// This would be called from handlers to create audit logs
	// Implementation would use the audit repository directly
}

// AuditUpdate is a helper to log update actions
func AuditUpdate(c *gin.Context, resourceType string, resourceID uuid.UUID, changes map[string]interface{}, description string) {
	// This would be called from handlers to create audit logs
}

// AuditDelete is a helper to log delete actions
func AuditDelete(c *gin.Context, resourceType string, resourceID uuid.UUID, description string) {
	// This would be called from handlers to create audit logs
}

// Global audit middleware instance
var globalAuditMiddleware *AuditMiddleware

// InitAuditMiddleware initializes the global audit middleware
func InitAuditMiddleware(auditRepo *repository.AuditLogRepository) {
	globalAuditMiddleware = NewAuditMiddleware(auditRepo)
}

// AuditLog is the global audit logging middleware
func AuditLog() gin.HandlerFunc {
	if globalAuditMiddleware == nil {
		panic("audit middleware not initialized")
	}
	return globalAuditMiddleware.AuditLog()
}
