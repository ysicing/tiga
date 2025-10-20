package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
	auditservice "github.com/ysicing/tiga/internal/services/audit"
)

// AuditMiddleware handles unified audit logging
// T017: Enhanced to use AsyncLogger and unified AuditEvent model
//
// Reference: .claude/specs/006-gitness-tiga/tasks.md T017
//           .claude/specs/006-gitness-tiga/audit-unification.md Stage 2
type AuditMiddleware struct {
	asyncLogger *auditservice.AsyncLogger[*models.AuditEvent]
}

// NewAuditMiddleware creates a new unified audit middleware
// T017: Uses AsyncLogger instead of simple goroutine
func NewAuditMiddleware(asyncLogger *auditservice.AsyncLogger[*models.AuditEvent]) *AuditMiddleware {
	return &AuditMiddleware{
		asyncLogger: asyncLogger,
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

		// Capture response body using custom response writer
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Process request
		c.Next()

		// Build and log audit event asynchronously
		// T017: Use AsyncLogger for non-blocking writes
		auditEvent := m.buildAuditEvent(c, start, requestBody, blw.body.Bytes())
		if auditEvent != nil {
			if err := m.asyncLogger.Enqueue(auditEvent); err != nil {
				logrus.Errorf("Failed to enqueue audit event: %v", err)
			}
		}
	}
}

// buildAuditEvent builds a unified AuditEvent
// T017: Constructs AuditEvent with OldObject/NewObject extraction
func (m *AuditMiddleware) buildAuditEvent(c *gin.Context, start time.Time, requestBody, responseBody []byte) *models.AuditEvent {
	// Get user information
	var userUID, username string
	var principalType models.PrincipalType

	if uid, err := GetUserID(c); err == nil {
		userUID = uid.String()
		if name, err := GetUsername(c); err == nil {
			username = name
		} else {
			username = "unknown"
		}
		principalType = models.PrincipalTypeUser
	} else {
		userUID = "anonymous"
		username = "anonymous"
		principalType = models.PrincipalTypeAnonymous
	}

	// Determine action from method
	action := mapHTTPMethodToAction(c.Request.Method)

	// Extract resource information from path
	resourceType, resourceID := extractResourceFromPath(c.Request.URL.Path)

	// T017: Extract OldObject/NewObject from request/response bodies
	var oldObject, newObject string
	var diffObject models.DiffObject

	// For UPDATE/DELETE operations: response contains old object
	// For CREATE operations: response contains new object
	// For READ operations: no diff needed
	switch action {
	case models.ActionCreated:
		// New object from response body
		if len(responseBody) > 0 && len(responseBody) < 100*1024 {
			newObject = extractJSONFromResponse(responseBody)
		}

	case models.ActionUpdated:
		// Old object from request body (or would need to fetch from DB)
		// New object from response body
		if len(requestBody) > 0 && len(requestBody) < 100*1024 {
			oldObject = extractJSONFromRequest(requestBody)
		}
		if len(responseBody) > 0 && len(responseBody) < 100*1024 {
			newObject = extractJSONFromResponse(responseBody)
		}

	case models.ActionDeleted:
		// Old object from response body (if API returns deleted object)
		if len(responseBody) > 0 && len(responseBody) < 100*1024 {
			oldObject = extractJSONFromResponse(responseBody)
		}
	}

	// T017: Apply truncation if objects exceed 64KB
	if oldObject != "" {
		if result, err := auditservice.TruncateObject(json.RawMessage(oldObject)); err == nil {
			diffObject.OldObject = result.TruncatedJSON
			diffObject.OldObjectTruncated = result.WasTruncated
			if result.WasTruncated {
				diffObject.TruncatedFields = append(diffObject.TruncatedFields, result.TruncatedFields...)
			}
		}
	}

	if newObject != "" {
		if result, err := auditservice.TruncateObject(json.RawMessage(newObject)); err == nil {
			diffObject.NewObject = result.TruncatedJSON
			diffObject.NewObjectTruncated = result.WasTruncated
			if result.WasTruncated {
				diffObject.TruncatedFields = append(diffObject.TruncatedFields, result.TruncatedFields...)
			}
		}
	}

	// Extract client IP (with X-Forwarded-For support)
	clientIP := extractClientIP(c)

	// Get or generate request ID
	requestID := c.GetString("RequestID")
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Build resource metadata
	resourceData := make(map[string]string)
	if resourceID != "" {
		resourceData["resourceId"] = resourceID
	}
	resourceData["method"] = c.Request.Method
	resourceData["path"] = c.Request.URL.Path
	resourceData["statusCode"] = string(rune(c.Writer.Status()))

	// Create unified AuditEvent
	auditEvent := &models.AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UnixMilli(),
		Action:    action,
		ResourceType: resourceType,
		Resource: models.Resource{
			Type:       resourceType,
			Identifier: resourceID,
			Data:       resourceData,
		},
		User: models.Principal{
			UID:      userUID,
			Username: username,
			Type:     principalType,
		},
		DiffObject:    diffObject,
		ClientIP:      clientIP,
		UserAgent:     c.Request.UserAgent(),
		RequestMethod: c.Request.Method,
		RequestID:     requestID,
		CreatedAt:     time.Now(),
	}

	// Validate before logging
	if err := auditEvent.Validate(); err != nil {
		logrus.Errorf("Invalid audit event: %v", err)
		return nil
	}

	return auditEvent
}

// bodyLogWriter wraps gin.ResponseWriter to capture response body
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Helper functions

func shouldSkipAudit(path string) bool {
	skipPaths := []string{
		"/health",
		"/ready",
		"/metrics",
		"/api/v1/auth/refresh",
		"/swagger",
	}

	for _, skip := range skipPaths {
		if strings.HasPrefix(path, skip) {
			return true
		}
	}

	return false
}

func shouldLogBody(method string) bool {
	return method == "POST" || method == "PUT" || method == "PATCH"
}

func mapHTTPMethodToAction(method string) models.Action {
	switch method {
	case "GET", "HEAD":
		return models.ActionRead
	case "POST":
		return models.ActionCreated
	case "PUT", "PATCH":
		return models.ActionUpdated
	case "DELETE":
		return models.ActionDeleted
	default:
		return models.ActionRead
	}
}

func extractResourceFromPath(path string) (models.ResourceType, string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Pattern: /api/v1/{resource_type}/{resource_id}
	if len(parts) >= 3 {
		resourceTypeStr := parts[2]

		// Map API paths to ResourceType enum
		var resourceType models.ResourceType
		switch {
		case strings.Contains(resourceTypeStr, "cluster"):
			resourceType = models.ResourceTypeCluster
		case strings.Contains(resourceTypeStr, "database"):
			resourceType = models.ResourceTypeDatabase
		case strings.Contains(resourceTypeStr, "pod"):
			resourceType = models.ResourceTypePod
		case strings.Contains(resourceTypeStr, "deployment"):
			resourceType = models.ResourceTypeDeployment
		case strings.Contains(resourceTypeStr, "service"):
			resourceType = models.ResourceTypeService
		case strings.Contains(resourceTypeStr, "user"):
			resourceType = models.ResourceTypeUser
		case strings.Contains(resourceTypeStr, "role"):
			resourceType = models.ResourceTypeRole
		case strings.Contains(resourceTypeStr, "scheduler"), strings.Contains(resourceTypeStr, "task"):
			resourceType = models.ResourceTypeScheduledTask
		default:
			resourceType = models.ResourceType(resourceTypeStr)
		}

		// Extract resource ID if present
		if len(parts) >= 4 {
			return resourceType, parts[3]
		}

		return resourceType, ""
	}

	return models.ResourceType("unknown"), ""
}

func extractClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header first
	if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
		// Take the first IP in the list
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	return c.ClientIP()
}

func extractJSONFromRequest(body []byte) string {
	// Attempt to parse as JSON
	var data interface{}
	if err := json.Unmarshal(body, &data); err == nil {
		return string(body)
	}
	return ""
}

func extractJSONFromResponse(body []byte) string {
	// Attempt to extract JSON from response
	// Response might be wrapped in {data: ...}
	var wrapper map[string]interface{}
	if err := json.Unmarshal(body, &wrapper); err == nil {
		if data, ok := wrapper["data"]; ok {
			if jsonData, err := json.Marshal(data); err == nil {
				return string(jsonData)
			}
		}
		// Return whole response if not wrapped
		return string(body)
	}
	return ""
}

// Global audit middleware instance
var globalAuditMiddleware *AuditMiddleware

// InitAuditMiddleware initializes the global audit middleware
// T017: Updated to use AsyncLogger
func InitAuditMiddleware(asyncLogger *auditservice.AsyncLogger[*models.AuditEvent]) {
	globalAuditMiddleware = NewAuditMiddleware(asyncLogger)
}

// AuditLog is the global audit logging middleware
func AuditLog() gin.HandlerFunc {
	if globalAuditMiddleware == nil {
		panic("audit middleware not initialized")
	}
	return globalAuditMiddleware.AuditLog()
}
