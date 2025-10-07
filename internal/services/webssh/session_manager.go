package webssh

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ysicing/tiga/internal/models"
	"gorm.io/gorm"
)

// SessionManager manages WebSSH sessions
type SessionManager struct {
	db              *gorm.DB
	activeSessions  sync.Map // map[string]*models.WebSSHSession
	sessionTimeout  time.Duration
	maxSessions     int
}

// NewSessionManager creates a new session manager
func NewSessionManager(db *gorm.DB) *SessionManager {
	sm := &SessionManager{
		db:             db,
		sessionTimeout: 30 * time.Minute,
		maxSessions:    100,
	}

	// Start cleanup goroutine
	go sm.cleanupStaleSessions()

	return sm
}

// CreateSession creates a new WebSSH session
func (m *SessionManager) CreateSession(ctx context.Context, userID, hostID uint, cols, rows int, clientIP string) (*models.WebSSHSession, error) {
	// Check max sessions limit
	count := 0
	m.activeSessions.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	if count >= m.maxSessions {
		return nil, fmt.Errorf("maximum session limit reached")
	}

	session := &models.WebSSHSession{
		SessionID:  "sess_" + uuid.New().String(),
		UserID:     userID,
		HostNodeID: hostID,
		ClientIP:   clientIP,
		Cols:       cols,
		Rows:       rows,
		Status:     "active",
	}

	// Save to database
	if err := m.db.WithContext(ctx).Create(session).Error; err != nil {
		return nil, err
	}

	// Add to active sessions
	m.activeSessions.Store(session.SessionID, session)

	return session, nil
}

// GetSession retrieves a session by ID
func (m *SessionManager) GetSession(sessionID string) (*models.WebSSHSession, error) {
	if session, ok := m.activeSessions.Load(sessionID); ok {
		return session.(*models.WebSSHSession), nil
	}

	// Fallback to database
	var session models.WebSSHSession
	if err := m.db.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// UpdateActivity updates the last activity time for a session
func (m *SessionManager) UpdateActivity(sessionID string) {
	if session, ok := m.activeSessions.Load(sessionID); ok {
		s := session.(*models.WebSSHSession)
		s.LastActive = time.Now()
		m.db.Save(s)
	}
}

// CloseSession closes a WebSSH session
func (m *SessionManager) CloseSession(ctx context.Context, sessionID, reason string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.Close(reason)

	// Update database
	if err := m.db.WithContext(ctx).Save(session).Error; err != nil {
		return err
	}

	// Remove from active sessions
	m.activeSessions.Delete(sessionID)

	return nil
}

// ListActiveSessions returns all active sessions
func (m *SessionManager) ListActiveSessions() []*models.WebSSHSession {
	var sessions []*models.WebSSHSession
	m.activeSessions.Range(func(key, value interface{}) bool {
		sessions = append(sessions, value.(*models.WebSSHSession))
		return true
	})
	return sessions
}

// cleanupStaleSessions removes stale sessions periodically
func (m *SessionManager) cleanupStaleSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.activeSessions.Range(func(key, value interface{}) bool {
			session := value.(*models.WebSSHSession)
			if time.Since(session.LastActive) > m.sessionTimeout {
				m.CloseSession(context.Background(), session.SessionID, "timeout")
			}
			return true
		})
	}
}
