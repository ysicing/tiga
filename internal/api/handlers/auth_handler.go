package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/api/middleware"
	"github.com/ysicing/tiga/internal/services/auth"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	loginService   *auth.LoginService
	sessionService *auth.SessionService
	oauthManager   *auth.OAuthManager
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	loginService *auth.LoginService,
	sessionService *auth.SessionService,
	oauthManager *auth.OAuthManager,
) *AuthHandler {
	return &AuthHandler{
		loginService:   loginService,
		sessionService: sessionService,
		oauthManager:   oauthManager,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles user login (password-based)
// @Summary User login
// @Description Authenticate user with username and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} SuccessResponse
// @Success 204 "No Content - Login successful (cookie set)"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/auth/login/password [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if !BindJSON(c, &req) {
		return
	}

	// Get client info
	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	// Perform login
	loginReq := &auth.LoginRequest{
		Username:   req.Username,
		Password:   req.Password,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		DeviceType: "web",
	}

	response, err := h.loginService.Login(c.Request.Context(), loginReq)
	if err != nil {
		RespondUnauthorized(c, err)
		return
	}

	// Set JWT token as HTTP-only cookie
	// Calculate if we should use secure flag
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"auth_token",
		response.TokenPair.AccessToken,
		int(response.TokenPair.ExpiresAt.Sub(time.Now()).Seconds()),
		"/",
		"",
		secure,
		true, // httpOnly
	)

	// Return 204 No Content for compatibility with old frontend
	c.Status(http.StatusNoContent)
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	Token string `json:"token" binding:"required"`
}

// Logout handles user logout
// @Summary User logout
// @Description Invalidate user session
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get token from header
	token := c.GetHeader("Authorization")
	if token == "" {
		RespondUnauthorized(c, fmt.Errorf("authorization token required"))
		return
	}

	// Remove "Bearer " prefix
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// Logout
	if err := h.loginService.Logout(c.Request.Context(), token); err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccessWithMessage(c, nil, "logged out successfully")
}

// RefreshTokenRequest represents a refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Get a new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if !BindJSON(c, &req) {
		return
	}

	// Refresh token
	tokenPair, err := h.loginService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		RespondUnauthorized(c, err)
		return
	}

	RespondSuccess(c, tokenPair)
}

// ChangePasswordRequest represents a change password request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword handles password change
// @Summary Change password
// @Description Change current user's password
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "Password change request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if !BindJSON(c, &req) {
		return
	}

	// Get user ID from context
	userID, err := middleware.GetUserID(c)
	if err != nil {
		RespondUnauthorized(c, err)
		return
	}

	// Change password
	if err := h.loginService.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		RespondBadRequest(c, err)
		return
	}

	RespondSuccessWithMessage(c, nil, "password changed successfully")
}

// GetProfile returns the current user's profile
// @Summary Get user profile
// @Description Get current authenticated user's profile
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		RespondUnauthorized(c, err)
		return
	}

	username, _ := middleware.GetUsername(c)
	email, _ := middleware.GetEmail(c)
	roles, _ := middleware.GetRoles(c)

	profile := gin.H{
		"user_id":  userID,
		"username": username,
		"email":    email,
		"roles":    roles,
	}

	RespondSuccess(c, profile)
}

// OAuthLoginRequest represents an OAuth login request
type OAuthLoginRequest struct {
	Provider string `json:"provider" binding:"required"` // google, github, oidc
	Code     string `json:"code" binding:"required"`
	State    string `json:"state" binding:"required"`
}

// OAuthLogin handles OAuth login
// @Summary OAuth login
// @Description Complete OAuth login flow
// @Tags auth
// @Accept json
// @Produce json
// @Param request body OAuthLoginRequest true "OAuth login request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/oauth/login [post]
func (h *AuthHandler) OAuthLogin(c *gin.Context) {
	var req OAuthLoginRequest
	if !BindJSON(c, &req) {
		return
	}

	// Exchange code for token
	token, err := h.oauthManager.ExchangeCode(c.Request.Context(), req.Provider, req.Code)
	if err != nil {
		RespondUnauthorized(c, fmt.Errorf("failed to exchange code: %w", err))
		return
	}

	// Get user info
	userInfo, err := h.oauthManager.GetUserInfo(c.Request.Context(), req.Provider, token)
	if err != nil {
		RespondUnauthorized(c, fmt.Errorf("failed to get user info: %w", err))
		return
	}

	// TODO: Create or update user in database
	// TODO: Generate JWT tokens
	// TODO: Create session

	RespondSuccess(c, gin.H{
		"user_info": userInfo,
		"message":   "OAuth login successful (implementation pending)",
	})
}

// OAuthAuthURLRequest represents a request for OAuth authorization URL
type OAuthAuthURLRequest struct {
	Provider string `form:"provider" binding:"required"`
}

// GetOAuthAuthURL returns the OAuth authorization URL
// @Summary Get OAuth authorization URL
// @Description Get the URL to redirect user for OAuth authorization
// @Tags auth
// @Produce json
// @Param provider query string true "OAuth provider (google, github, oidc)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/auth/oauth/url [get]
func (h *AuthHandler) GetOAuthAuthURL(c *gin.Context) {
	var req OAuthAuthURLRequest
	if !BindQuery(c, &req) {
		return
	}

	// Generate state (random UUID)
	state := uuid.New().String()

	// Get auth URL
	authURL, err := h.oauthManager.GetAuthURL(req.Provider, state)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	RespondSuccess(c, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

// GetActiveSessions returns active sessions for the current user
// @Summary Get active sessions
// @Description Get all active sessions for the current user
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/sessions [get]
func (h *AuthHandler) GetActiveSessions(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		RespondUnauthorized(c, err)
		return
	}

	sessions, err := h.sessionService.GetUserSessions(c.Request.Context(), userID, true)
	if err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccess(c, sessions)
}

// InvalidateSessionRequest represents a request to invalidate a session
type InvalidateSessionRequest struct {
	SessionID string `uri:"session_id" binding:"required,uuid"`
}

// InvalidateSession invalidates a specific session
// @Summary Invalidate session
// @Description Invalidate a specific session by ID
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Param session_id path string true "Session ID (UUID)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/sessions/{session_id} [delete]
func (h *AuthHandler) InvalidateSession(c *gin.Context) {
	var req InvalidateSessionRequest
	if !BindURI(c, &req) {
		return
	}

	// Parse session ID
	sessionID, err := ParseUUID(req.SessionID)
	if err != nil {
		RespondBadRequest(c, err)
		return
	}

	// Invalidate session
	if err := h.sessionService.InvalidateSession(c.Request.Context(), sessionID); err != nil {
		RespondInternalError(c, err)
		return
	}

	RespondSuccessWithMessage(c, nil, "session invalidated successfully")
}

// GetCurrentUser gets the currently authenticated user
// @Summary Get current user
// @Description Get information about the currently authenticated user
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} ErrorResponse
// @Router /api/auth/user [get]
// @Security BearerAuth
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		RespondUnauthorized(c, fmt.Errorf("user not authenticated"))
		return
	}

	RespondSuccess(c, gin.H{"user": user})
}

// GetOAuthProviders gets list of available OAuth providers
// @Summary Get OAuth providers
// @Description Get list of available OAuth authentication providers
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/auth/providers [get]
func (h *AuthHandler) GetOAuthProviders(c *gin.Context) {
	// Get configured OAuth providers from the OAuth manager
	providers := []string{}

	// Always include password authentication as an available option
	providers = append(providers, "password")

	// Add OAuth providers if available
	if h.oauthManager != nil {
		oauthProviders := h.oauthManager.ListProviders()
		providers = append(providers, oauthProviders...)
	}

	RespondSuccess(c, gin.H{"providers": providers})
}
