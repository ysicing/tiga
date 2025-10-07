package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LoggerConfig represents logger middleware configuration
type LoggerConfig struct {
	SkipPaths      []string // Paths to skip logging
	LogRequestBody bool     // Whether to log request body
	LogResponse    bool     // Whether to log response
}

// RequestLogger returns a gin middleware for request logging
func RequestLogger() gin.HandlerFunc {
	return RequestLoggerWithConfig(&LoggerConfig{
		SkipPaths: []string{"/health", "/ready", "/metrics"},
	})
}

// RequestLoggerWithConfig returns a gin middleware with custom configuration
func RequestLoggerWithConfig(config *LoggerConfig) gin.HandlerFunc {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip logging for specified paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Get user info if available
		username := ""
		if user, exists := c.Get(string(UsernameKey)); exists {
			username = fmt.Sprintf("%v", user)
		}

		// Build log message
		logMsg := fmt.Sprintf("[GIN] %s | %3d | %13v | %15s | %-7s %s",
			start.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)

		if query != "" {
			logMsg += "?" + query
		}

		if username != "" {
			logMsg += fmt.Sprintf(" | user=%s", username)
		}

		// Log errors if any
		if len(c.Errors) > 0 {
			logMsg += fmt.Sprintf(" | errors=%s", c.Errors.String())
		}

		// Log based on status code
		if statusCode >= 500 {
			logrus.Errorf("%s | UA=%s", logMsg, userAgent)
		} else if statusCode >= 400 {
			logrus.Warnf("%s | UA=%s", logMsg, userAgent)
		} else {
			logrus.Infof("%s", logMsg)
		}
	}
}

// AccessLogger logs detailed access information
type AccessLogger struct {
	Path       string        `json:"path"`
	Method     string        `json:"method"`
	StatusCode int           `json:"status_code"`
	Latency    time.Duration `json:"latency"`
	ClientIP   string        `json:"client_ip"`
	UserAgent  string        `json:"user_agent"`
	Username   string        `json:"username,omitempty"`
	Errors     string        `json:"errors,omitempty"`
	Timestamp  time.Time     `json:"timestamp"`
}

// DetailedLogger returns a detailed structured logger middleware
func DetailedLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		log := AccessLogger{
			Path:       c.Request.URL.Path,
			Method:     c.Request.Method,
			StatusCode: c.Writer.Status(),
			Latency:    time.Since(start),
			ClientIP:   c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			Timestamp:  start,
		}

		// Get username if available
		if username, exists := c.Get(string(UsernameKey)); exists {
			log.Username = fmt.Sprintf("%v", username)
		}

		// Get errors if any
		if len(c.Errors) > 0 {
			log.Errors = c.Errors.String()
		}

		logrus.WithFields(logrus.Fields{
			"path":      log.Path,
			"method":    log.Method,
			"status":    log.StatusCode,
			"latency":   log.Latency,
			"client_ip": log.ClientIP,
			"username":  log.Username,
		}).Info("access")
	}
}

// MetricsLogger logs metrics for Prometheus
func MetricsLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		duration := time.Since(start).Seconds()
		statusCode := c.Writer.Status()

		// Record metrics (placeholder - integrate with Prometheus)
		logrus.WithFields(logrus.Fields{
			"path":     path,
			"method":   method,
			"status":   statusCode,
			"duration": duration,
		}).Debug("metrics")
	}
}
