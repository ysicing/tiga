package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/models"
)

// RBACMiddleware provides simplified K8s resource access control based on is_admin flag
// Admin users: Full access to all K8s resources
// Regular users: Read-only access (GET, LIST operations)
func RBACMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(models.User)

		// Admin users have full access
		if user.IsAdmin {
			c.Next()
			return
		}

		// Regular users can only perform read operations
		verb := method2verb(c.Request.Method)
		if verb == "get" || verb == "list" {
			c.Next()
			return
		}

		// Deny write operations for non-admin users
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Admin access required for this operation",
		})
	}
}

func method2verb(method string) string {
	switch method {
	case http.MethodGet:
		return "get"
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return strings.ToLower(method)
	}
}
