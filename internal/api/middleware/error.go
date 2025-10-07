package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string      `json:"error"`
	Message string      `json:"message,omitempty"`
	Code    string      `json:"code,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// Recovery returns a middleware that recovers from panics
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				logrus.Errorf("Panic recovered: %v\n%s", err, debug.Stack())

				// Return error response
				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Error:   "internal server error",
					Message: "an unexpected error occurred",
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}

// ErrorHandler handles errors and provides standardized error responses
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// Determine status code
			statusCode := c.Writer.Status()
			if statusCode == http.StatusOK {
				statusCode = http.StatusInternalServerError
			}

			// Build error response
			response := ErrorResponse{
				Error:   err.Error(),
				Message: err.Meta.(string),
			}

			// Log error
			logrus.Errorf("Request error: %v", err)

			c.JSON(statusCode, response)
		}
	}
}

// AbortWithError aborts the request with an error
func AbortWithError(c *gin.Context, statusCode int, err error) {
	c.JSON(statusCode, ErrorResponse{
		Error: err.Error(),
	})
	c.Abort()
}

// AbortWithMessage aborts the request with a message
func AbortWithMessage(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, ErrorResponse{
		Error: message,
	})
	c.Abort()
}

// AbortWithErrorResponse aborts the request with a detailed error response
func AbortWithErrorResponse(c *gin.Context, statusCode int, errResp ErrorResponse) {
	c.JSON(statusCode, errResp)
	c.Abort()
}

// HandleError is a helper to handle errors in handlers
func HandleError(c *gin.Context, err error) {
	statusCode := http.StatusInternalServerError
	message := err.Error()

	// Map specific errors to status codes
	switch {
	case isNotFoundError(err):
		statusCode = http.StatusNotFound
	case isValidationError(err):
		statusCode = http.StatusBadRequest
	case isUnauthorizedError(err):
		statusCode = http.StatusUnauthorized
	case isForbiddenError(err):
		statusCode = http.StatusForbidden
	case isConflictError(err):
		statusCode = http.StatusConflict
	}

	logrus.Errorf("Handler error: %v", err)

	c.JSON(statusCode, ErrorResponse{
		Error:   message,
		Message: message,
	})
}

// Error type checkers
func isNotFoundError(err error) bool {
	return contains(err.Error(), "not found")
}

func isValidationError(err error) bool {
	return contains(err.Error(), "invalid", "validation", "required")
}

func isUnauthorizedError(err error) bool {
	return contains(err.Error(), "unauthorized", "authentication", "token")
}

func isForbiddenError(err error) bool {
	return contains(err.Error(), "forbidden", "permission", "access denied")
}

func isConflictError(err error) bool {
	return contains(err.Error(), "already exists", "conflict", "duplicate")
}

func contains(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) &&
			(s == substr || containsSubstring(s, substr)) {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// NewValidationError creates a new validation error response
func NewValidationError(field, message string) ErrorResponse {
	return ErrorResponse{
		Error:   "validation error",
		Message: fmt.Sprintf("field '%s': %s", field, message),
		Details: ValidationError{
			Field:   field,
			Message: message,
		},
	}
}

// MultiValidationErrors represents multiple validation errors
func MultiValidationErrors(errors []ValidationError) ErrorResponse {
	return ErrorResponse{
		Error:   "validation errors",
		Message: fmt.Sprintf("%d validation error(s) occurred", len(errors)),
		Details: errors,
	}
}
