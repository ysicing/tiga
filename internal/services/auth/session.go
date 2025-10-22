package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// SessionService manages user sessions
type SessionService struct {
	db                 *gorm.DB
	maxSessionsPerUser int // Maximum concurrent sessions per user (0 = unlimited)
}

// NewSessionService creates a new SessionService
func NewSessionService(db *gorm.DB) *SessionService {
	return &SessionService{
		db:                 db,
		maxSessionsPerUser: 5, // Default: 5 concurrent sessions per user
	}
}

// SetMaxSessionsPerUser sets the maximum concurrent sessions per user
func (s *SessionService) SetMaxSessionsPerUser(max int) {
	s.maxSessionsPerUser = max
}

// CreateSessionRequest represents a session creation request
type CreateSessionRequest struct {
	UserID       uuid.UUID `json:"user_id"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	DeviceType   string    `json:"device_type,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// CreateSession creates a new session
func (s *SessionService) CreateSession(ctx context.Context, req *CreateSessionRequest) (*models.Session, error) {
	// Enforce maximum concurrent sessions per user (if configured)
	if s.maxSessionsPerUser > 0 {
		// Count current active sessions for this user
		activeCount, err := s.CountActiveSessions(ctx, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to count active sessions: %w", err)
		}

		// If at or over limit, invalidate oldest session(s)
		if activeCount >= int64(s.maxSessionsPerUser) {
			sessionsToRemove := activeCount - int64(s.maxSessionsPerUser) + 1
			if err := s.invalidateOldestSessions(ctx, req.UserID, int(sessionsToRemove)); err != nil {
				// Log error but don't fail the login - cleanup is best effort
				fmt.Printf("Warning: failed to cleanup old sessions for user %s: %v\n", req.UserID, err)
			}
		}
	}

	session := &models.Session{
		UserID:         req.UserID,
		Token:          req.Token,
		RefreshToken:   req.RefreshToken,
		IPAddress:      req.IPAddress,
		UserAgent:      req.UserAgent,
		DeviceType:     req.DeviceType,
		ExpiresAt:      req.ExpiresAt,
		IsActive:       true,
		LastActivityAt: time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// invalidateOldestSessions invalidates the N oldest active sessions for a user
func (s *SessionService) invalidateOldestSessions(ctx context.Context, userID uuid.UUID, count int) error {
	// Get oldest active sessions
	var oldSessions []models.Session
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND is_active = ? AND expires_at > ?", userID, true, time.Now()).
		Order("last_activity_at ASC"). // Oldest first
		Limit(count).
		Find(&oldSessions).Error

	if err != nil {
		return fmt.Errorf("failed to query old sessions: %w", err)
	}

	// Extract session IDs
	var sessionIDs []uuid.UUID
	for _, session := range oldSessions {
		sessionIDs = append(sessionIDs, session.ID)
	}

	if len(sessionIDs) == 0 {
		return nil // Nothing to invalidate
	}

	// Batch invalidate
	err = s.db.WithContext(ctx).
		Model(&models.Session{}).
		Where("id IN ?", sessionIDs).
		Update("is_active", false).Error

	if err != nil {
		return fmt.Errorf("failed to invalidate old sessions: %w", err)
	}

	fmt.Printf("Invalidated %d old session(s) for user %s (concurrent session limit enforcement)\n", len(sessionIDs), userID)
	return nil
}

// GetSession retrieves a session by ID
func (s *SessionService) GetSession(ctx context.Context, sessionID uuid.UUID) (*models.Session, error) {
	var session models.Session
	err := s.db.WithContext(ctx).
		Preload("User").
		Where("id = ?", sessionID).
		First(&session).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

// GetSessionByToken retrieves a session by token
func (s *SessionService) GetSessionByToken(ctx context.Context, token string) (*models.Session, error) {
	var session models.Session
	err := s.db.WithContext(ctx).
		Preload("User").
		Where("token = ?", token).
		First(&session).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

// ValidateSession validates a session by token
func (s *SessionService) ValidateSession(ctx context.Context, token string) (*models.Session, error) {
	session, err := s.GetSessionByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check if session is valid
	if !session.IsValid() {
		return nil, fmt.Errorf("session is invalid or expired")
	}

	return session, nil
}

// RefreshSession refreshes a session's activity time
func (s *SessionService) RefreshSession(ctx context.Context, sessionID uuid.UUID) error {
	now := time.Now()
	err := s.db.WithContext(ctx).
		Model(&models.Session{}).
		Where("id = ?", sessionID).
		Update("last_activity_at", now).Error

	if err != nil {
		return fmt.Errorf("failed to refresh session: %w", err)
	}

	return nil
}

// RefreshSessionByToken refreshes a session by token
func (s *SessionService) RefreshSessionByToken(ctx context.Context, token string) error {
	now := time.Now()
	err := s.db.WithContext(ctx).
		Model(&models.Session{}).
		Where("token = ?", token).
		Update("last_activity_at", now).Error

	if err != nil {
		return fmt.Errorf("failed to refresh session: %w", err)
	}

	return nil
}

// UpdateSessionToken updates the session's access token
func (s *SessionService) UpdateSessionToken(ctx context.Context, refreshToken, newAccessToken string) error {
	err := s.db.WithContext(ctx).
		Model(&models.Session{}).
		Where("refresh_token = ?", refreshToken).
		Updates(map[string]interface{}{
			"token":            newAccessToken,
			"last_activity_at": time.Now(),
		}).Error

	if err != nil {
		return fmt.Errorf("failed to update session token: %w", err)
	}

	return nil
}

// InvalidateSession invalidates a session
func (s *SessionService) InvalidateSession(ctx context.Context, sessionID uuid.UUID) error {
	err := s.db.WithContext(ctx).
		Model(&models.Session{}).
		Where("id = ?", sessionID).
		Update("is_active", false).Error

	if err != nil {
		return fmt.Errorf("failed to invalidate session: %w", err)
	}

	return nil
}

// InvalidateSessionByToken invalidates a session by token
func (s *SessionService) InvalidateSessionByToken(ctx context.Context, token string) error {
	err := s.db.WithContext(ctx).
		Model(&models.Session{}).
		Where("token = ?", token).
		Update("is_active", false).Error

	if err != nil {
		return fmt.Errorf("failed to invalidate session: %w", err)
	}

	return nil
}

// InvalidateAllUserSessions invalidates all sessions for a user
func (s *SessionService) InvalidateAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	err := s.db.WithContext(ctx).
		Model(&models.Session{}).
		Where("user_id = ?", userID).
		Update("is_active", false).Error

	if err != nil {
		return fmt.Errorf("failed to invalidate user sessions: %w", err)
	}

	return nil
}

// GetUserSessions retrieves all sessions for a user
func (s *SessionService) GetUserSessions(ctx context.Context, userID uuid.UUID, activeOnly bool) ([]*models.Session, error) {
	var sessions []*models.Session

	query := s.db.WithContext(ctx).Where("user_id = ?", userID)
	if activeOnly {
		query = query.Where("is_active = ? AND expires_at > ?", true, time.Now())
	}

	err := query.Order("last_activity_at DESC").Find(&sessions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	return sessions, nil
}

// GetActiveSessions retrieves all active sessions
func (s *SessionService) GetActiveSessions(ctx context.Context, limit int) ([]*models.Session, error) {
	var sessions []*models.Session

	query := s.db.WithContext(ctx).
		Preload("User").
		Where("is_active = ? AND expires_at > ?", true, time.Now()).
		Order("last_activity_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&sessions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}

	return sessions, nil
}

// CleanupExpiredSessions removes expired sessions from the database
func (s *SessionService) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	result := s.db.WithContext(ctx).
		Where("expires_at < ? OR (is_active = ? AND last_activity_at < ?)",
			time.Now(),
			true,
			time.Now().Add(-7*24*time.Hour), // Clean sessions inactive for 7 days
		).
		Delete(&models.Session{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// CountActiveSessions counts active sessions for a user
func (s *SessionService) CountActiveSessions(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&models.Session{}).
		Where("user_id = ? AND is_active = ? AND expires_at > ?", userID, true, time.Now()).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count active sessions: %w", err)
	}

	return count, nil
}

// GetSessionsByIPAddress retrieves sessions by IP address
func (s *SessionService) GetSessionsByIPAddress(ctx context.Context, ipAddress string, activeOnly bool) ([]*models.Session, error) {
	var sessions []*models.Session

	query := s.db.WithContext(ctx).
		Preload("User").
		Where("ip_address = ?", ipAddress)

	if activeOnly {
		query = query.Where("is_active = ? AND expires_at > ?", true, time.Now())
	}

	err := query.Order("created_at DESC").Find(&sessions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by IP: %w", err)
	}

	return sessions, nil
}

// ExtendSession extends a session's expiration time
func (s *SessionService) ExtendSession(ctx context.Context, sessionID uuid.UUID, duration time.Duration) error {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	newExpiresAt := time.Now().Add(duration)
	err = s.db.WithContext(ctx).
		Model(&models.Session{}).
		Where("id = ?", sessionID).
		Updates(map[string]interface{}{
			"expires_at":       newExpiresAt,
			"last_activity_at": time.Now(),
		}).Error

	if err != nil {
		return fmt.Errorf("failed to extend session: %w", err)
	}

	fmt.Printf("Extended session %s until %s\n", session.ID, newExpiresAt)
	return nil
}

// SessionStatistics represents session statistics
type SessionStatistics struct {
	TotalSessions   int64 `json:"total_sessions"`
	ActiveSessions  int64 `json:"active_sessions"`
	ExpiredSessions int64 `json:"expired_sessions"`
	UniqueUsers     int64 `json:"unique_users"`
}

// GetSessionStatistics retrieves session statistics
func (s *SessionService) GetSessionStatistics(ctx context.Context) (*SessionStatistics, error) {
	stats := &SessionStatistics{}

	// Total sessions
	if err := s.db.WithContext(ctx).Model(&models.Session{}).Count(&stats.TotalSessions).Error; err != nil {
		return nil, fmt.Errorf("failed to count total sessions: %w", err)
	}

	// Active sessions
	if err := s.db.WithContext(ctx).
		Model(&models.Session{}).
		Where("is_active = ? AND expires_at > ?", true, time.Now()).
		Count(&stats.ActiveSessions).Error; err != nil {
		return nil, fmt.Errorf("failed to count active sessions: %w", err)
	}

	// Expired sessions
	if err := s.db.WithContext(ctx).
		Model(&models.Session{}).
		Where("expires_at <= ?", time.Now()).
		Count(&stats.ExpiredSessions).Error; err != nil {
		return nil, fmt.Errorf("failed to count expired sessions: %w", err)
	}

	// Unique users with active sessions
	if err := s.db.WithContext(ctx).
		Model(&models.Session{}).
		Where("is_active = ? AND expires_at > ?", true, time.Now()).
		Distinct("user_id").
		Count(&stats.UniqueUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to count unique users: %w", err)
	}

	return stats, nil
}
