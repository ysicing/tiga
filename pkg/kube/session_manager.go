package kube

import (
	"sync"

	"github.com/google/uuid"
)

// SessionManager manages all active K8s terminal sessions
// Thread-safe using sync.Map
// Reference: 010-k8s-pod-009 T018
type SessionManager struct {
	sessions sync.Map // map[uuid.UUID]*K8sTerminalSession
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{}
}

// AddSession adds a new session to the manager
func (m *SessionManager) AddSession(session *K8sTerminalSession) {
	m.sessions.Store(session.SessionID, session)
}

// GetSession retrieves a session by ID
// Returns (session, true) if found, (nil, false) if not found
func (m *SessionManager) GetSession(sessionID uuid.UUID) (*K8sTerminalSession, bool) {
	value, ok := m.sessions.Load(sessionID)
	if !ok {
		return nil, false
	}

	session, ok := value.(*K8sTerminalSession)
	return session, ok
}

// RemoveSession removes a session from the manager
func (m *SessionManager) RemoveSession(sessionID uuid.UUID) {
	m.sessions.Delete(sessionID)
}

// GetAllSessions returns all active sessions
// Useful for monitoring and cleanup operations
func (m *SessionManager) GetAllSessions() []*K8sTerminalSession {
	sessions := make([]*K8sTerminalSession, 0)

	m.sessions.Range(func(key, value interface{}) bool {
		if session, ok := value.(*K8sTerminalSession); ok {
			sessions = append(sessions, session)
		}
		return true // Continue iteration
	})

	return sessions
}

// Count returns the number of active sessions
func (m *SessionManager) Count() int {
	count := 0
	m.sessions.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// GetSessionsByCluster returns all sessions for a specific cluster
func (m *SessionManager) GetSessionsByCluster(clusterID string) []*K8sTerminalSession {
	sessions := make([]*K8sTerminalSession, 0)

	m.sessions.Range(func(key, value interface{}) bool {
		if session, ok := value.(*K8sTerminalSession); ok {
			if session.ClusterID == clusterID {
				sessions = append(sessions, session)
			}
		}
		return true
	})

	return sessions
}

// CloseAllSessions closes all active sessions
// Used during graceful shutdown
func (m *SessionManager) CloseAllSessions() []error {
	errors := make([]error, 0)

	m.sessions.Range(func(key, value interface{}) bool {
		if session, ok := value.(*K8sTerminalSession); ok {
			if err := session.Close(); err != nil {
				errors = append(errors, err)
			}
			m.sessions.Delete(key)
		}
		return true
	})

	return errors
}
