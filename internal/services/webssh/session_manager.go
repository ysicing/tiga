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
	db             *gorm.DB
	activeSessions sync.Map // map[string]*models.WebSSHSession
	recorders      sync.Map // map[string]*SessionRecorder
	sessionTimeout time.Duration
	maxSessions    int
	recordingDir   string
}

// NewSessionManager creates a new session manager
func NewSessionManager(db *gorm.DB, recordingDir string) *SessionManager {
	if recordingDir == "" {
		recordingDir = "./data/recordings"
	}

	sm := &SessionManager{
		db:             db,
		sessionTimeout: 30 * time.Minute,
		maxSessions:    100,
		recordingDir:   recordingDir,
	}

	// Start cleanup goroutine
	go sm.cleanupStaleSessions()

	return sm
}

// CreateSession creates a new WebSSH session.
// If sessionID is empty, a new UUID-based identifier will be generated.
func (m *SessionManager) CreateSession(ctx context.Context, sessionID string, userID, hostID uuid.UUID, cols, rows int, clientIP string, recordingEnabled bool) (*models.WebSSHSession, error) {
	// Check max sessions limit
	count := 0
	m.activeSessions.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	if count >= m.maxSessions {
		return nil, fmt.Errorf("maximum session limit reached")
	}

	if sessionID == "" {
		sessionID = "sess_" + uuid.New().String()
	}

	session := &models.WebSSHSession{
		SessionID:        sessionID,
		UserID:           userID,
		HostNodeID:       hostID,
		ClientIP:         clientIP,
		Cols:             cols,
		Rows:             rows,
		Status:           "active",
		RecordingEnabled: recordingEnabled,
		RecordingFormat:  "asciicast",
	}

	// Create recorder if recording is enabled
	if recordingEnabled {
		recorder, err := NewSessionRecorder(sessionID, cols, rows, m.recordingDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create recorder: %w", err)
		}
		session.RecordingPath = recorder.GetFilePath()
		m.recorders.Store(sessionID, recorder)
	}

	// Save to database
	if err := m.db.WithContext(ctx).Create(session).Error; err != nil {
		// Cleanup recorder if database save fails
		if recordingEnabled {
			if rec, ok := m.recorders.Load(sessionID); ok {
				rec.(*SessionRecorder).Close()
				m.recorders.Delete(sessionID)
			}
		}
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
		// Update database asynchronously to avoid blocking hot path
		go func(s *models.WebSSHSession) {
			_ = m.db.Save(s).Error
		}(s)
	}
}

// CloseSession closes a WebSSH session
func (m *SessionManager) CloseSession(ctx context.Context, sessionID, reason string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.Close(reason)

	// Close recorder if exists
	if rec, ok := m.recorders.Load(sessionID); ok {
		recorder := rec.(*SessionRecorder)
		if err := recorder.Close(); err != nil {
			fmt.Printf("Failed to close recorder: %v\n", err)
		}
		// Update recording size
		session.RecordingSize = recorder.GetBytesWritten()
		m.recorders.Delete(sessionID)
	}

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

// RecordOutput records terminal output
func (m *SessionManager) RecordOutput(sessionID string, data []byte) error {
	if rec, ok := m.recorders.Load(sessionID); ok {
		return rec.(*SessionRecorder).RecordOutput(data)
	}
	return nil // Recording not enabled
}

// RecordInput records terminal input
func (m *SessionManager) RecordInput(sessionID string, data []byte) error {
	if rec, ok := m.recorders.Load(sessionID); ok {
		return rec.(*SessionRecorder).RecordInput(data)
	}
	return nil // Recording not enabled
}

// ResizeRecorder updates terminal size in recorder
func (m *SessionManager) ResizeRecorder(sessionID string, cols, rows int) {
	if rec, ok := m.recorders.Load(sessionID); ok {
		rec.(*SessionRecorder).Resize(cols, rows)
	}
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
