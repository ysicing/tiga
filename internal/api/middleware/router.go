package middleware

import (
	"net/http"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RouterConfig represents router configuration
type RouterConfig struct {
	DebugMode       bool
	EnableGzip      bool
	EnableCORS      bool
	EnableRateLimit bool
	EnableSwagger   bool
	TrustedProxies  []string
}

// NewRouter creates and configures a new Gin router
func NewRouter(config *RouterConfig) *gin.Engine {
	// Set Gin mode
	if config.DebugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Set trusted proxies
	if len(config.TrustedProxies) > 0 {
		_ = router.SetTrustedProxies(config.TrustedProxies)
	} else {
		_ = router.SetTrustedProxies(nil)
	}

	// Global middlewares
	router.Use(Recovery())      // Panic recovery
	router.Use(RequestLogger()) // Request logging

	// Optional middlewares
	if config.EnableGzip {
		router.Use(gzip.Gzip(gzip.DefaultCompression))
	}

	if config.EnableCORS {
		router.Use(CORS())
	}

	// Health check endpoints (no auth required)
	router.GET("/health", healthCheck)
	router.GET("/ready", readinessCheck)

	// Metrics endpoint (Prometheus)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Swagger documentation (if enabled)
	if config.EnableSwagger {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Note: All actual routes are registered via api.SetupRoutes()
	// No placeholder routes needed here

	return router
}

// healthCheck handles health check requests
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Unix(),
	})
}

// readinessCheck handles readiness check requests
func readinessCheck(c *gin.Context) {
	// TODO: Add database connectivity check, etc.
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"time":   time.Now().Unix(),
	})
}

// SetupRoutes sets up all application routes
func SetupRoutes(router *gin.Engine, routeHandlers map[string]gin.HandlerFunc) {
	// This is a placeholder for setting up routes with actual handlers
	// In practice, you would register your handlers here
	for path, handler := range routeHandlers {
		router.GET(path, handler)
	}
}

// NoRoute handles 404 errors
func NoRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "route not found",
			"path":  c.Request.URL.Path,
		})
	}
}

// NoMethod handles 405 errors
func NoMethod() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error":  "method not allowed",
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
		})
	}
}
