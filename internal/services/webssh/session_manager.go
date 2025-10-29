package webssh

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// SessionManager manages WebSSH sessions
type SessionManager struct {
	db                 *gorm.DB
	activeSessions     sync.Map // map[string]*models.WebSSHSession
	userSessions       sync.Map // map[uuid.UUID]map[string]bool - tracks sessions per user
	recorders          sync.Map // map[string]*SessionRecorder
	sessionTimeout     time.Duration
	maxSessions        int
	maxSessionsPerUser int // Maximum sessions per user
	recordingDir       string
	mu                 sync.RWMutex // Protects metrics
	metrics            ConnectionMetrics
}

// ConnectionMetrics tracks connection pool metrics
type ConnectionMetrics struct {
	TotalActiveSessions int
	UserSessionCounts   map[uuid.UUID]int
	TotalSessionsOpened int64
	TotalSessionsClosed int64
}

// NewSessionManager creates a new session manager
func NewSessionManager(db *gorm.DB, recordingDir string) *SessionManager {
	if recordingDir == "" {
		recordingDir = "./data/recordings"
	}

	sm := &SessionManager{
		db:                 db,
		sessionTimeout:     30 * time.Minute,
		maxSessions:        100,
		maxSessionsPerUser: 5, // Default: 5 sessions per user
		recordingDir:       recordingDir,
		metrics: ConnectionMetrics{
			UserSessionCounts: make(map[uuid.UUID]int),
		},
	}

	// Start cleanup goroutine
	go sm.cleanupStaleSessions()

	return sm
}

// CreateSession creates a new WebSSH session.
// If sessionID is empty, a new UUID-based identifier will be generated.
func (m *SessionManager) CreateSession(ctx context.Context, sessionID string, userID, hostID uuid.UUID, cols, rows int, clientIP string, recordingEnabled bool) (*models.WebSSHSession, error) {
	// Check global max sessions limit
	count := 0
	m.activeSessions.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	if count >= m.maxSessions {
		return nil, fmt.Errorf("maximum session limit reached (%d)", m.maxSessions)
	}

	// Check per-user max sessions limit
	userSessionCount := m.GetUserSessionCount(userID)
	if userSessionCount >= m.maxSessionsPerUser {
		return nil, fmt.Errorf("maximum sessions per user reached (%d)", m.maxSessionsPerUser)
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

	// Track user session
	m.addUserSession(userID, sessionID)

	// Update metrics
	m.updateMetricsOnCreate(userID)

	return session, nil
}

// GetSession retrieves a session by ID
func (m *SessionManager) GetSession(sessionID string) (*models.WebSSHSession, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	if session, ok := m.activeSessions.Load(sessionID); ok {
		return session.(*models.WebSSHSession), nil
	}

	// Fallback to database
	var session models.WebSSHSession
	if err := m.db.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, err
	}
	return &session, nil
}

// UpdateActivity updates the last activity time for a session
func (m *SessionManager) UpdateActivity(sessionID string) {
	// Update database asynchronously to avoid blocking hot path
	// Don't modify the in-memory session object to avoid data race
	go func(sid string) {
		now := time.Now()
		_ = m.db.Model(&models.WebSSHSession{}).
			Where("session_id = ?", sid).
			Update("last_active", now).Error
	}(sessionID)
}

// CloseSession closes a WebSSH session
func (m *SessionManager) CloseSession(ctx context.Context, sessionID, reason string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	// Check if session is already closed
	if session.Status == "closed" {
		return fmt.Errorf("session already closed: %s", sessionID)
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

	// Save to unified terminal_recordings table if recording was enabled
	if session.RecordingEnabled && session.RecordingPath != "" {
		if err := m.saveToUnifiedRecordingTable(ctx, session); err != nil {
			// Log error but don't fail session closure
			fmt.Printf("Failed to save to unified recording table: %v\n", err)
		}
	}

	// Remove from user sessions tracking
	m.removeUserSession(session.UserID, sessionID)

	// Update metrics
	m.updateMetricsOnClose(session.UserID)

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

// addUserSession adds a session to user's session tracking
func (m *SessionManager) addUserSession(userID uuid.UUID, sessionID string) {
	// Load or create user session set
	sessionsInterface, _ := m.userSessions.LoadOrStore(userID, &sync.Map{})
	sessions := sessionsInterface.(*sync.Map)
	sessions.Store(sessionID, true)
}

// removeUserSession removes a session from user's session tracking
func (m *SessionManager) removeUserSession(userID uuid.UUID, sessionID string) {
	if sessionsInterface, ok := m.userSessions.Load(userID); ok {
		sessions := sessionsInterface.(*sync.Map)
		sessions.Delete(sessionID)
	}
}

// GetUserSessionCount returns the number of active sessions for a user
func (m *SessionManager) GetUserSessionCount(userID uuid.UUID) int {
	count := 0
	if sessionsInterface, ok := m.userSessions.Load(userID); ok {
		sessions := sessionsInterface.(*sync.Map)
		sessions.Range(func(key, value interface{}) bool {
			count++
			return true
		})
	}
	return count
}

// GetUserSessions returns all active sessions for a user
func (m *SessionManager) GetUserSessions(userID uuid.UUID) []*models.WebSSHSession {
	var userSessions []*models.WebSSHSession

	if sessionsInterface, ok := m.userSessions.Load(userID); ok {
		sessions := sessionsInterface.(*sync.Map)
		sessions.Range(func(key, value interface{}) bool {
			sessionID := key.(string)
			if session, err := m.GetSession(sessionID); err == nil {
				userSessions = append(userSessions, session)
			}
			return true
		})
	}

	return userSessions
}

// updateMetricsOnCreate updates metrics when a session is created
func (m *SessionManager) updateMetricsOnCreate(userID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics.TotalActiveSessions++
	m.metrics.TotalSessionsOpened++
	m.metrics.UserSessionCounts[userID] = m.GetUserSessionCount(userID)
}

// updateMetricsOnClose updates metrics when a session is closed
func (m *SessionManager) updateMetricsOnClose(userID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics.TotalActiveSessions--
	m.metrics.TotalSessionsClosed++

	userCount := m.GetUserSessionCount(userID)
	if userCount > 0 {
		m.metrics.UserSessionCounts[userID] = userCount
	} else {
		delete(m.metrics.UserSessionCounts, userID)
	}
}

// GetMetrics returns current connection pool metrics
func (m *SessionManager) GetMetrics() ConnectionMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy to avoid race conditions
	metricsCopy := ConnectionMetrics{
		TotalActiveSessions: m.metrics.TotalActiveSessions,
		TotalSessionsOpened: m.metrics.TotalSessionsOpened,
		TotalSessionsClosed: m.metrics.TotalSessionsClosed,
		UserSessionCounts:   make(map[uuid.UUID]int),
	}

	for userID, count := range m.metrics.UserSessionCounts {
		metricsCopy.UserSessionCounts[userID] = count
	}

	return metricsCopy
}

// SetMaxSessionsPerUser allows configuring the per-user session limit
func (m *SessionManager) SetMaxSessionsPerUser(max int) {
	if max > 0 {
		m.maxSessionsPerUser = max
	}
}

// SetSessionTimeout allows configuring the session timeout duration
func (m *SessionManager) SetSessionTimeout(timeout time.Duration) {
	if timeout > 0 {
		m.sessionTimeout = timeout
	}
}

// saveToUnifiedRecordingTable saves a WebSSH recording to the unified terminal_recordings table
func (m *SessionManager) saveToUnifiedRecordingTable(ctx context.Context, session *models.WebSSHSession) error {
	// Parse session ID to UUID
	sessionUUID, err := uuid.Parse(session.SessionID)
	if err != nil {
		return fmt.Errorf("invalid session ID format: %w", err)
	}

	// Fetch username from User table
	var user models.User
	if err := m.db.WithContext(ctx).Where("id = ?", session.UserID).First(&user).Error; err != nil {
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	// Fetch host information for TypeMetadata
	var hostNode models.HostNode
	if err := m.db.WithContext(ctx).Where("id = ?", session.HostNodeID).First(&hostNode).Error; err != nil {
		return fmt.Errorf("failed to fetch host node: %w", err)
	}

	// Fetch host info
	var hostInfo models.HostInfo
	if err := m.db.WithContext(ctx).Where("host_node_id = ?", session.HostNodeID).First(&hostInfo).Error; err == nil {
		// HostInfo found, will include in metadata
	} else {
		// HostInfo not found, create empty one
		hostInfo = models.HostInfo{}
	}

	// Calculate duration
	var duration int
	if session.EndTime != nil && !session.StartTime.IsZero() {
		duration = int(session.EndTime.Sub(session.StartTime).Seconds())
	}

	// Construct TypeMetadata JSONB
	typeMetadataJSON := map[string]interface{}{
		"session_id": session.SessionID,
		"host_id":    session.HostNodeID.String(),
		"host":       hostNode.Name,
	}

	// Convert to JSON string
	typeMetadataStr, err := json.Marshal(typeMetadataJSON)
	if err != nil {
		return fmt.Errorf("failed to marshal type metadata: %w", err)
	}

	// Create TerminalRecording entry
	recording := &models.TerminalRecording{
		SessionID:     sessionUUID,
		UserID:        &session.UserID, // Convert to pointer for optional foreign key
		Username:      user.Username,
		RecordingType: "webssh",
		TypeMetadata:  datatypes.JSON(typeMetadataStr),
		StorageType:   "local",
		StoragePath:   session.RecordingPath,
		FileSize:      session.RecordingSize,
		Format:        "asciinema",
		StartedAt:     session.StartTime,
		EndedAt:       session.EndTime,
		Duration:      duration,
		Rows:          session.Rows,
		Cols:          session.Cols,
		Shell:         "/bin/bash",
		ClientIP:      session.ClientIP,
	}

	if err := m.db.WithContext(ctx).Create(recording).Error; err != nil {
		return fmt.Errorf("failed to create unified recording: %w", err)
	}

	return nil
}
