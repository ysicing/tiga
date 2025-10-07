package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SessionCache manages user session caching
type SessionCache struct {
	manager *CacheManager
}

// NewSessionCache creates a new session cache
func NewSessionCache(manager *CacheManager) *SessionCache {
	return &SessionCache{
		manager: manager,
	}
}

// SessionData represents cached session data
type SessionData struct {
	UserID       uuid.UUID              `json:"user_id"`
	Username     string                 `json:"username"`
	Email        string                 `json:"email"`
	Roles        []string               `json:"roles"`
	Permissions  []string               `json:"permissions"`
	ExpiresAt    time.Time              `json:"expires_at"`
	LastActivity time.Time              `json:"last_activity"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// GetSession retrieves session data from cache
func (sc *SessionCache) GetSession(ctx context.Context, sessionID string) (*SessionData, error) {
	if !sc.manager.IsEnabled() {
		return nil, ErrCacheMiss
	}

	key := sc.buildSessionKey(sessionID)
	var session SessionData

	if err := sc.manager.client.GetJSON(ctx, key, &session); err != nil {
		return nil, err
	}

	// Check if session has expired
	if time.Now().After(session.ExpiresAt) {
		sc.DeleteSession(ctx, sessionID)
		return nil, ErrCacheMiss
	}

	return &session, nil
}

// SetSession stores session data in cache
func (sc *SessionCache) SetSession(ctx context.Context, sessionID string, session *SessionData) error {
	if !sc.manager.IsEnabled() {
		return nil
	}

	key := sc.buildSessionKey(sessionID)

	// Calculate TTL based on expiration
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = sc.manager.config.SessionTTL
	}

	return sc.manager.client.SetJSON(ctx, key, session, ttl)
}

// DeleteSession removes session data from cache
func (sc *SessionCache) DeleteSession(ctx context.Context, sessionID string) error {
	if !sc.manager.IsEnabled() {
		return nil
	}

	key := sc.buildSessionKey(sessionID)
	return sc.manager.client.Delete(ctx, key)
}

// UpdateSessionActivity updates the last activity timestamp
func (sc *SessionCache) UpdateSessionActivity(ctx context.Context, sessionID string) error {
	if !sc.manager.IsEnabled() {
		return nil
	}

	session, err := sc.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	session.LastActivity = time.Now()
	return sc.SetSession(ctx, sessionID, session)
}

// InvalidateUserSessions invalidates all sessions for a user
func (sc *SessionCache) InvalidateUserSessions(ctx context.Context, userID uuid.UUID) error {
	if !sc.manager.IsEnabled() {
		return nil
	}

	pattern := sc.manager.CacheKey(sc.manager.config.SessionPrefix, "*")
	return sc.manager.client.DeletePattern(ctx, pattern)
}

// buildSessionKey builds a cache key for a session
func (sc *SessionCache) buildSessionKey(sessionID string) string {
	return sc.manager.CacheKey(sc.manager.config.SessionPrefix, sessionID)
}

// SetSessionWithTTL stores session with custom TTL
func (sc *SessionCache) SetSessionWithTTL(ctx context.Context, sessionID string, session *SessionData, ttl time.Duration) error {
	if !sc.manager.IsEnabled() {
		return nil
	}

	key := sc.buildSessionKey(sessionID)
	return sc.manager.client.SetJSON(ctx, key, session, ttl)
}

// ExtendSession extends session expiration
func (sc *SessionCache) ExtendSession(ctx context.Context, sessionID string, extension time.Duration) error {
	if !sc.manager.IsEnabled() {
		return nil
	}

	session, err := sc.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	session.ExpiresAt = session.ExpiresAt.Add(extension)
	return sc.SetSession(ctx, sessionID, session)
}

// IsSessionValid checks if a session exists and is valid
func (sc *SessionCache) IsSessionValid(ctx context.Context, sessionID string) (bool, error) {
	session, err := sc.GetSession(ctx, sessionID)
	if err != nil {
		if err == ErrCacheMiss {
			return false, nil
		}
		return false, err
	}

	return time.Now().Before(session.ExpiresAt), nil
}

// GetUserActiveSessions retrieves all active session IDs for a user
// Note: This is a simplified implementation. In production, you'd maintain a user->sessions index
func (sc *SessionCache) GetUserActiveSessions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	if !sc.manager.IsEnabled() {
		return nil, ErrCacheMiss
	}

	// In a real implementation, you'd maintain a separate index
	// For now, return empty slice
	return []string{}, nil
}

// RateLimitCache manages rate limiting data
type RateLimitCache struct {
	manager *CacheManager
}

// NewRateLimitCache creates a new rate limit cache
func NewRateLimitCache(manager *CacheManager) *RateLimitCache {
	return &RateLimitCache{
		manager: manager,
	}
}

// IncrementRequest increments request count for rate limiting
func (rlc *RateLimitCache) IncrementRequest(ctx context.Context, key string, window time.Duration) (int64, error) {
	if !rlc.manager.IsEnabled() {
		return 0, ErrCacheDisabled
	}

	cacheKey := rlc.buildRateLimitKey(key)

	// Check if key exists
	exists, err := rlc.manager.client.Exists(ctx, cacheKey)
	if err != nil {
		return 0, err
	}

	// Increment counter
	count, err := rlc.manager.client.Increment(ctx, cacheKey)
	if err != nil {
		return 0, err
	}

	// Set expiration on first request
	if !exists {
		if err := rlc.manager.client.Expire(ctx, cacheKey, window); err != nil {
			return count, err
		}
	}

	return count, nil
}

// GetRequestCount retrieves current request count
func (rlc *RateLimitCache) GetRequestCount(ctx context.Context, key string) (int64, error) {
	if !rlc.manager.IsEnabled() {
		return 0, ErrCacheMiss
	}

	cacheKey := rlc.buildRateLimitKey(key)
	val, err := rlc.manager.client.Get(ctx, cacheKey)
	if err != nil {
		if err == ErrCacheMiss {
			return 0, nil
		}
		return 0, err
	}

	var count int64
	if _, err := fmt.Sscanf(val, "%d", &count); err != nil {
		return 0, fmt.Errorf("failed to parse count: %w", err)
	}

	return count, nil
}

// ResetRequestCount resets request count for a key
func (rlc *RateLimitCache) ResetRequestCount(ctx context.Context, key string) error {
	if !rlc.manager.IsEnabled() {
		return nil
	}

	cacheKey := rlc.buildRateLimitKey(key)
	return rlc.manager.client.Delete(ctx, cacheKey)
}

// buildRateLimitKey builds a cache key for rate limiting
func (rlc *RateLimitCache) buildRateLimitKey(key string) string {
	return rlc.manager.CacheKey("ratelimit", key)
}

// GetTimeToReset retrieves time until rate limit resets
func (rlc *RateLimitCache) GetTimeToReset(ctx context.Context, key string) (time.Duration, error) {
	if !rlc.manager.IsEnabled() {
		return 0, ErrCacheDisabled
	}

	cacheKey := rlc.buildRateLimitKey(key)
	return rlc.manager.client.GetTTL(ctx, cacheKey)
}
