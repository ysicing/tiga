package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/repository"

	authservices "github.com/ysicing/tiga/internal/services/auth"
)

// ContextKey represents context keys
type ContextKey string

const (
	UserIDKey   ContextKey = "user_id"
	UsernameKey ContextKey = "username"
	EmailKey    ContextKey = "email"
	RolesKey    ContextKey = "roles"
)

// JWTAuthMiddleware handles JWT authentication using the new JWTManager
type JWTAuthMiddleware struct {
	jwtManager *authservices.JWTManager
	userRepo   *repository.UserRepository
}

// NewJWTAuthMiddleware creates a new JWT auth middleware
func NewJWTAuthMiddleware(jwtManager *authservices.JWTManager, db *gorm.DB) *JWTAuthMiddleware {
	return &JWTAuthMiddleware{
		jwtManager: jwtManager,
		userRepo:   repository.NewUserRepository(db),
	}
}

// AuthRequired is a middleware that requires authentication
func (m *JWTAuthMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// Try to get token from Authorization header first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Check Bearer token format
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}

		// Fall back to cookie if header is not present
		if token == "" {
			cookieToken, err := c.Cookie("auth_token")
			if err != nil || cookieToken == "" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "authorization header or auth_token cookie required",
				})
				c.Abort()
				return
			}
			token = cookieToken
		}

		// Validate token using JWTManager
		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			c.Abort()
			return
		}

		// Get full user from database
		user, err := m.userRepo.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "user not found",
			})
			c.Abort()
			return
		}

		// Check if user is enabled
		if !user.Enabled {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "user account is disabled",
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user", *user)
		c.Set(string(UserIDKey), claims.UserID.String())
		c.Set(string(UsernameKey), claims.Username)
		c.Set(string(EmailKey), claims.Email)
		c.Set(string(RolesKey), claims.Roles)

		c.Next()
	}
}

// Global middleware instance (for backward compatibility)
var globalJWTMiddleware *JWTAuthMiddleware

// InitJWTAuthMiddleware initializes the global JWT auth middleware
func InitJWTAuthMiddleware(jwtManager *authservices.JWTManager, db *gorm.DB) {
	globalJWTMiddleware = NewJWTAuthMiddleware(jwtManager, db)
}

// AuthRequired returns the global auth middleware handler
func AuthRequired() gin.HandlerFunc {
	if globalJWTMiddleware == nil {
		panic("JWT auth middleware not initialized")
	}
	return globalJWTMiddleware.AuthRequired()
}

// GetUserID retrieves user ID from context
func GetUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get(string(UserIDKey))
	if !exists {
		return uuid.Nil, fmt.Errorf("user not authenticated")
	}

	// Handle both string and UUID types
	switch v := userIDStr.(type) {
	case string:
		return uuid.Parse(v)
	case uuid.UUID:
		return v, nil
	default:
		return uuid.Nil, fmt.Errorf("invalid user ID format")
	}
}

// GetUsername retrieves username from context
func GetUsername(c *gin.Context) (string, error) {
	username, exists := c.Get(string(UsernameKey))
	if !exists {
		return "", fmt.Errorf("user not authenticated")
	}

	name, ok := username.(string)
	if !ok {
		return "", fmt.Errorf("invalid username format")
	}

	return name, nil
}

// GetEmail retrieves email from context
func GetEmail(c *gin.Context) (string, error) {
	email, exists := c.Get(string(EmailKey))
	if !exists {
		return "", fmt.Errorf("user not authenticated")
	}

	mail, ok := email.(string)
	if !ok {
		return "", fmt.Errorf("invalid email format")
	}

	return mail, nil
}

// GetRoles retrieves roles from context
func GetRoles(c *gin.Context) ([]string, error) {
	roles, exists := c.Get(string(RolesKey))
	if !exists {
		return nil, fmt.Errorf("user not authenticated")
	}

	roleList, ok := roles.([]string)
	if !ok {
		return nil, fmt.Errorf("invalid roles format")
	}

	return roleList, nil
}

// MustGetUserID retrieves user ID from context or panics
func MustGetUserID(c *gin.Context) uuid.UUID {
	userID, err := GetUserID(c)
	if err != nil {
		panic(err)
	}
	return userID
}

// MustGetUsername retrieves username from context or panics
func MustGetUsername(c *gin.Context) string {
	username, err := GetUsername(c)
	if err != nil {
		panic(err)
	}
	return username
}
