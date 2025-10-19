package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ysicing/tiga/internal/config"
)

// ReadonlyMode is a middleware that blocks write operations when readonly mode is enabled
// It allows GET, HEAD, OPTIONS requests and blocks POST, PUT, PATCH, DELETE requests
func ReadonlyMode(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if readonly mode is not enabled
		if !cfg.Features.ReadonlyMode {
			c.Next()
			return
		}

		// Allow read-only HTTP methods
		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			c.Next()
			return
		}

		// Allow specific read-only API endpoints even if they use POST
		// (e.g., query endpoints that use POST for complex filters)
		path := c.Request.URL.Path
		if isReadonlyEndpoint(path) {
			c.Next()
			return
		}

		// Block all other write operations
		c.JSON(http.StatusForbidden, gin.H{
			"code":    http.StatusForbidden,
			"message": "Write operations are disabled in readonly mode",
		})
		c.Abort()
	}
}

// isReadonlyEndpoint checks if an endpoint is considered read-only
// even if it uses POST method (e.g., for complex queries)
func isReadonlyEndpoint(path string) bool {
	readonlyPaths := []string{
		"/api/v1/auth/login",        // Allow login
		"/api/v1/auth/refresh",      // Allow token refresh
		"/api/v1/database/query",    // Query execution (read-only queries still allowed)
		"/api/v1/prometheus/query",  // Prometheus query
		"/api/v1/k8s/clusters/test", // Test cluster connection
	}

	for _, readonlyPath := range readonlyPaths {
		if strings.HasPrefix(path, readonlyPath) {
			return true
		}
	}

	return false
}
