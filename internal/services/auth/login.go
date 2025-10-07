package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// LoginService handles user authentication
type LoginService struct {
	db             *gorm.DB
	passwordHasher *PasswordHasher
	jwtManager     *JWTManager
	sessionService *SessionService
}

// NewLoginService creates a new LoginService
func NewLoginService(db *gorm.DB, jwtManager *JWTManager, sessionService *SessionService) *LoginService {
	return &LoginService{
		db:             db,
		passwordHasher: NewPasswordHasher(),
		jwtManager:     jwtManager,
		sessionService: sessionService,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	IPAddress  string `json:"ip_address,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	DeviceType string `json:"device_type,omitempty"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User      *UserInfo    `json:"user"`
	TokenPair *TokenPair   `json:"token"`
	Session   *SessionInfo `json:"session"`
	ExpiresAt time.Time    `json:"expires_at"`
}

// UserInfo represents user information in login response
type UserInfo struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
	Avatar   string    `json:"avatar_url,omitempty"`
	Roles    []string  `json:"roles"`
}

// SessionInfo represents session information
type SessionInfo struct {
	ID        uuid.UUID `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Login authenticates a user with username and password
func (s *LoginService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// Find user by username or email
	var user models.User
	err := s.db.WithContext(ctx).
		Where("username = ? OR email = ?", req.Username, req.Username).
		Where("status = ?", "active").
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid username or password")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Verify password
	if err := s.passwordHasher.Verify(req.Password, user.Password); err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	// Check if account is suspended or deleted
	if user.Status != "active" {
		return nil, fmt.Errorf("account is not active")
	}

	// Generate simple role based on is_admin flag
	var roleNames []string
	if user.IsAdmin {
		roleNames = []string{"admin"}
	} else {
		roleNames = []string{"user"}
	}

	// Generate JWT token pair
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Username, user.Email, roleNames)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session
	session, err := s.sessionService.CreateSession(ctx, &CreateSessionRequest{
		UserID:       user.ID,
		Token:        tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		DeviceType:   req.DeviceType,
		ExpiresAt:    tokenPair.ExpiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Update last login time
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.db.WithContext(ctx).Model(&user).Update("last_login_at", now).Error; err != nil {
		// Log error but don't fail the login
		fmt.Printf("Failed to update last login time: %v\n", err)
	}

	return &LoginResponse{
		User: &UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			FullName: user.FullName,
			Avatar:   user.AvatarURL,
			Roles:    roleNames,
		},
		TokenPair: tokenPair,
		Session: &SessionInfo{
			ID:        session.ID,
			ExpiresAt: session.ExpiresAt,
		},
		ExpiresAt: tokenPair.ExpiresAt,
	}, nil
}

// Logout logs out a user by invalidating their session
func (s *LoginService) Logout(ctx context.Context, token string) error {
	// Extract user ID from token
	userID, err := s.jwtManager.ExtractUserID(token)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// Invalidate session
	if err := s.sessionService.InvalidateSessionByToken(ctx, token); err != nil {
		return fmt.Errorf("failed to invalidate session: %w", err)
	}

	// Could also add to token blacklist here if needed

	fmt.Printf("User %s logged out successfully\n", userID)
	return nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *LoginService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Validate refresh token
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user to fetch current roles
	var user models.User
	err = s.db.WithContext(ctx).
		Where("id = ? AND status = ?", claims.UserID, "active").
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found or inactive")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Generate simple role based on is_admin flag
	var roleNames []string
	if user.IsAdmin {
		roleNames = []string{"admin"}
	} else {
		roleNames = []string{"user"}
	}

	// Generate new access token
	tokenPair, err := s.jwtManager.RefreshAccessToken(refreshToken, roleNames)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update session with new access token
	if err := s.sessionService.UpdateSessionToken(ctx, refreshToken, tokenPair.AccessToken); err != nil {
		// Log error but don't fail the refresh
		fmt.Printf("Failed to update session token: %v\n", err)
	}

	return tokenPair, nil
}

// ValidateCredentials validates username and password without creating a session
func (s *LoginService) ValidateCredentials(ctx context.Context, username, password string) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).
		Where("username = ? OR email = ?", username, username).
		Where("status = ?", "active").
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if err := s.passwordHasher.Verify(password, user.Password); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return &user, nil
}

// ChangePassword changes a user's password
func (s *LoginService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	// Validate new password strength
	if err := ValidatePasswordStrength(newPassword); err != nil {
		return fmt.Errorf("weak password: %w", err)
	}

	// Get user
	var user models.User
	err := s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Verify old password
	if err := s.passwordHasher.Verify(oldPassword, user.Password); err != nil {
		return fmt.Errorf("invalid old password")
	}

	// Hash new password
	newHash, err := s.passwordHasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	if err := s.db.WithContext(ctx).Model(&user).Update("password", newHash).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Invalidate all existing sessions for security
	if err := s.sessionService.InvalidateAllUserSessions(ctx, userID); err != nil {
		fmt.Printf("Failed to invalidate user sessions: %v\n", err)
	}

	return nil
}

// ResetPassword resets a user's password (admin or password reset flow)
func (s *LoginService) ResetPassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	// Validate new password strength
	if err := ValidatePasswordStrength(newPassword); err != nil {
		return fmt.Errorf("weak password: %w", err)
	}

	// Hash new password
	newHash, err := s.passwordHasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	err = s.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("password", newHash).Error

	if err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	// Invalidate all existing sessions
	if err := s.sessionService.InvalidateAllUserSessions(ctx, userID); err != nil {
		fmt.Printf("Failed to invalidate user sessions: %v\n", err)
	}

	return nil
}
