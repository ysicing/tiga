package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/pkg/common"
	"github.com/ysicing/tiga/pkg/rbac"

	authservices "github.com/ysicing/tiga/internal/services/auth"
)

// getConfigFromContext retrieves config from gin context, returns nil if not found
func getConfigFromContext(c *gin.Context) *config.Config {
	if cfg, exists := c.Get("config"); exists {
		if appCfg, ok := cfg.(*config.Config); ok {
			return appCfg
		}
	}
	return nil
}

// getCookieExpiration returns cookie expiration seconds from config or default
func getCookieExpiration(c *gin.Context) int {
	if cfg := getConfigFromContext(c); cfg != nil {
		return int(cfg.JWT.ExpiresIn) * 2 // Double JWT expiration for cookie
	}
	return 24 * 60 * 60 * 2 // Default: 48 hours (double of 24 hours JWT)
}

// isAnonymousEnabled returns anonymous user enabled flag from config or default
func isAnonymousEnabled(c *gin.Context) bool {
	if cfg := getConfigFromContext(c); cfg != nil {
		return cfg.Features.AnonymousUserEnabled
	}
	return false // Default: disabled
}

// AnonymousUser for backward compatibility
var AnonymousUser = models.User{
	ID:       uuid.Nil,
	Username: "anonymous",
	Provider: "Anonymous",
	Enabled:  true,
}

type AuthHandler struct {
	manager           *OAuthManager
	userRepo          *repository.UserRepository
	oauthProviderRepo *repository.OAuthProviderRepository
	db                *gorm.DB
}

func NewAuthHandler(db *gorm.DB, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		manager:           NewOAuthManager(jwtSecret),
		userRepo:          repository.NewUserRepository(db),
		oauthProviderRepo: repository.NewOAuthProviderRepository(db),
		db:                db,
	}
}

func (h *AuthHandler) GetProviders(c *gin.Context) {
	providers := h.manager.GetAvailableProviders()
	providers = append(providers, "password")
	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	provider := c.Query("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Provider parameter is required",
		})
		return
	}

	oauthProvider, err := h.manager.GetProvider(c, provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	state := h.manager.GenerateState()

	logrus.Debugf("OAuth Login - Provider: %s, State: %s", provider, state)

	// Store state and provider in cookies with SameSite=Lax and Secure when appropriate
	setCookieSecure(c, "oauth_state", state, 600)
	setCookieSecure(c, "oauth_provider", provider, 600)

	authURL := oauthProvider.GetAuthURL(state)
	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"provider": provider,
	})
}

func (h *AuthHandler) PasswordLogin(c *gin.Context) {
	var req common.PasswordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	user, err := h.userRepo.GetByUsername(context.Background(), req.Username)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"username": req.Username,
			"error":    err.Error(),
		}).Debug("User not found during password login")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	if authservices.VerifyPassword(req.Password, user.Password) != nil {
		logrus.WithFields(logrus.Fields{
			"username": req.Username,
		}).Debug("Password verification failed")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	if !user.Enabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "user disabled"})
		return
	}

	if err := h.userRepo.UpdateLastLogin(context.Background(), user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed login"})
		return
	}

	jwtToken, err := h.manager.GenerateJWT(user, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT"})
		return
	}

	setCookieSecure(c, "auth_token", jwtToken, getCookieExpiration(c))

	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	provider, err := c.Cookie("oauth_provider")
	if err != nil {
		logrus.Error("OAuth Callback - No provider found in cookie: ", err)
		c.Redirect(http.StatusFound, "/login?error=missing_provider&reason=no_provider_in_cookie")
		return
	}

	stateParam := c.Query("state")
	cookieState, stateErr := c.Cookie("oauth_state")

	logrus.Debugf("OAuth Callback - Using provider: %s\n", provider)

	// Validate state to protect against CSRF and authorization code injection
	if stateErr != nil || stateParam == "" || cookieState == "" || stateParam != cookieState {
		logrus.Warnf("OAuth Callback - state mismatch or missing (cookieState=%v, stateParam=%v, err=%v)", cookieState, stateParam, stateErr)
		// Clear oauth cookies
		setCookieSecure(c, "oauth_state", "", -1)
		setCookieSecure(c, "oauth_provider", "", -1)
		c.Redirect(http.StatusFound, "/login?error=invalid_state&reason=state_mismatch")
		return
	}

	// Clear oauth cookies now that state is validated
	setCookieSecure(c, "oauth_state", "", -1)
	setCookieSecure(c, "oauth_provider", "", -1)

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Authorization code not provided",
		})
		return
	}

	// Get the OAuth provider
	oauthProvider, err := h.manager.GetProvider(c, provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Provider not found: " + provider,
		})
		return
	}

	logrus.Debugf("OAuth Callback - Exchanging code for token with provider: %s", provider)
	// Exchange code for token
	tokenResp, err := oauthProvider.ExchangeCodeForToken(code)
	if err != nil {
		c.Redirect(http.StatusFound, "/login?error=token_exchange_failed&reason=token_exchange_failed&provider="+provider)
		return
	}

	logrus.Debugf("OAuth Callback - Getting user info with provider: %s", provider)
	// Get user info
	user, err := oauthProvider.GetUserInfo(tokenResp.AccessToken)
	if err != nil {
		c.Redirect(http.StatusFound, "/login?error=user_info_failed&reason=user_info_failed&provider="+provider)
		return
	}

	if user.Sub == "" {
		c.Redirect(http.StatusFound, "/login?error=user_info_failed&reason=user_info_failed&provider="+provider)
		return
	}

	if err := h.userRepo.FindWithSubOrUpsert(context.Background(), user); err != nil {
		c.Redirect(http.StatusFound, "/login?error=user_upsert_failed&reason=user_upsert_failed&provider="+provider)
		return
	}
	role := rbac.GetUserRoles(*user)
	if len(role) == 0 {
		logrus.Warnf("OAuth Callback - Access denied for user: %s (provider: %s)", user.Key(), provider)
		c.Redirect(http.StatusFound, "/login?error=insufficient_permissions&reason=insufficient_permissions&user="+user.Key()+"&provider="+provider)
		return
	}
	if !user.Enabled {
		c.Redirect(http.StatusFound, "/login?error=user_disabled&reason=user_disabled")
		return
	}

	// Generate JWT with refresh token support
	jwtToken, err := h.manager.GenerateJWT(user, tokenResp.RefreshToken)
	if err != nil {
		c.Redirect(http.StatusFound, "/login?error=jwt_generation_failed&reason=jwt_generation_failed&user="+user.Key()+"&provider="+provider)
		return
	}

	// Set JWT as HTTP-only cookie with secure/samesite settings
	setCookieSecure(c, "auth_token", jwtToken, getCookieExpiration(c))

	c.Redirect(http.StatusFound, "/dashboard")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	setCookieSecure(c, "auth_token", "", -1)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
	})
}

func (h *AuthHandler) GetUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Not authenticated",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func (h *AuthHandler) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isAnonymousEnabled(c) {
			c.Set("user", AnonymousUser)
			c.Next()
			return
		}

		// Try to read auth token cookie (if missing, tokenString will be empty)
		tokenString, _ := c.Cookie("auth_token")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Validate and potentially refresh the token
		claims, err := h.manager.ValidateJWT(tokenString)
		if err != nil {
			// Try to refresh the token if validation fails
			refreshedToken, refreshErr := h.manager.RefreshJWT(c, tokenString)
			if refreshErr != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid or expired token",
				})
				setCookieSecure(c, "auth_token", "", -1)
				c.Abort()
				return
			}
			setCookieSecure(c, "auth_token", refreshedToken, getCookieExpiration(c))
			// Validate the refreshed token
			claims, err = h.manager.ValidateJWT(refreshedToken)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Failed to validate refreshed token",
				})
				setCookieSecure(c, "auth_token", "", -1)
				c.Abort()
				return
			}
		}
		user, err := h.userRepo.GetByID(context.Background(), uuid.MustParse(claims.UserID))
		if err != nil || !user.Enabled {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			setCookieSecure(c, "auth_token", "", -1)
			c.Abort()
			return
		}
		// Simplified: removed RBAC system, using is_admin flag
		c.Set("user", *user)
		c.Next()
	}
}

func (h *AuthHandler) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Not authenticated",
			})
			c.Abort()
			return
		}

		u := user.(models.User)
		if !rbac.UserHasRole(u, "admin") {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Admin role required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// Get token from cookie
	tokenString, err := c.Cookie("auth_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "No token found",
		})
		return
	}

	// Refresh the token
	newToken, err := h.manager.RefreshJWT(c, tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Failed to refresh token",
		})
		return
	}

	// Update the cookie with the new token
	setCookieSecure(c, "auth_token", newToken, getCookieExpiration(c))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Token refreshed successfully",
	})
}

// OAuth Provider Management APIs

func (h *AuthHandler) ListOAuthProviders(c *gin.Context) {
	providers, err := h.oauthProviderRepo.List(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve OAuth providers",
		})
		return
	}

	// Don't expose client secrets in the response
	for i := range providers {
		providers[i].ClientSecret = "***"
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
	})
}

func (h *AuthHandler) CreateOAuthProvider(c *gin.Context) {
	var provider models.OAuthProvider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request payload: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if provider.Name == "" || provider.ClientID == "" || string(provider.ClientSecret) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Name, ClientID, and ClientSecret are required",
		})
		return
	}

	if err := h.oauthProviderRepo.Create(context.Background(), &provider); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create OAuth provider: " + err.Error(),
		})
		return
	}

	// Note: Providers are now loaded dynamically from database, no reload needed

	// Don't expose client secret in response
	provider.ClientSecret = "***"
	c.JSON(http.StatusCreated, gin.H{
		"provider": provider,
	})
}

func (h *AuthHandler) UpdateOAuthProvider(c *gin.Context) {
	id := c.Param("id")
	var provider models.OAuthProvider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request payload: " + err.Error(),
		})
		return
	}

	// Parse ID and set it (convert to UUID)
	providerID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid provider ID",
		})
		return
	}
	provider.ID = providerID

	// Validate required fields
	if provider.Name == "" || provider.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Name and ClientID are required",
		})
		return
	}

	updates := map[string]interface{}{
		"name":          provider.Name,
		"client_id":     provider.ClientID,
		"auth_url":      provider.AuthURL,
		"token_url":     provider.TokenURL,
		"user_info_url": provider.UserInfoURL,
		"scopes":        provider.Scopes,
		"issuer":        provider.Issuer,
		"enabled":       provider.Enabled,
	}
	if provider.ClientSecret != "" {
		updates["client_secret"] = provider.ClientSecret
	}

	if err := h.oauthProviderRepo.UpdateFields(context.Background(), provider.ID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update OAuth provider: " + err.Error(),
		})
		return
	}
	// Don't expose client secret in response
	provider.ClientSecret = "***"
	c.JSON(http.StatusOK, gin.H{
		"provider": provider,
	})
}

func (h *AuthHandler) DeleteOAuthProvider(c *gin.Context) {
	id := c.Param("id")

	providerID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid provider ID",
		})
		return
	}

	if err := h.oauthProviderRepo.Delete(context.Background(), providerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete OAuth provider: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "OAuth provider deleted successfully",
	})
}

func (h *AuthHandler) GetOAuthProvider(c *gin.Context) {
	id := c.Param("id")
	providerID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid provider ID",
		})
		return
	}

	provider, err := h.oauthProviderRepo.GetByID(context.Background(), providerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "OAuth provider not found",
		})
		return
	}

	provider.ClientSecret = "***"
	c.JSON(http.StatusOK, gin.H{
		"provider": provider,
	})
}

// setCookieSecure sets a cookie with SameSite=Lax and HttpOnly=true. It marks Secure=true
// when the request is over TLS or X-Forwarded-Proto indicates https.
func setCookieSecure(c *gin.Context, name, value string, maxAge int) {
	// Determine if secure should be set based on TLS or X-Forwarded-Proto header
	secure := c.Request != nil && (c.Request.TLS != nil || strings.EqualFold(c.Request.Header.Get("X-Forwarded-Proto"), "https"))

	// Set SameSite to Lax for OAuth flows while still providing CSRF protection
	c.SetSameSite(http.SameSiteLaxMode)
	// The SetCookie function signature is (name, value string, maxAge int, path, domain string, secure, httpOnly bool)
	c.SetCookie(name, value, maxAge+60*60, "/", "", secure, true)
}
