package host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ysicing/tiga/proto"
)

// TerminalSession represents an active terminal session
type TerminalSession struct {
	StreamID  string
	HostID    uuid.UUID
	UUID      string
	StartedAt time.Time

	// gRPC stream from Agent
	AgentStream proto.HostMonitor_IOStreamServer

	// Buffered channels for bidirectional communication
	ToAgent   chan []byte
	FromAgent chan []byte

	// Error tracking
	LastError error
	ErrorChan chan error // Non-blocking error notifications

	// Session control
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	closed bool
}

// TerminalManager manages active terminal sessions
type TerminalManager struct {
	sessions sync.Map // StreamID -> *TerminalSession
}

// NewTerminalManager creates a new terminal manager
func NewTerminalManager() *TerminalManager {
	return &TerminalManager{}
}

// HandleIOStream handles the IOStream RPC from Agent
func (m *TerminalManager) HandleIOStream(stream proto.HostMonitor_IOStreamServer) error {
	ctx := stream.Context()

	// First message should contain StreamID with magic header
	firstMsg, err := stream.Recv()
	if err != nil {
		logrus.Errorf("[TerminalMgr] Failed to receive StreamID: %v", err)
		return fmt.Errorf("failed to receive StreamID: %w", err)
	}

	// Parse StreamID (format: 0xff 0x05 0xff 0x05 + streamID)
	if len(firstMsg.Data) < 5 {
		logrus.Errorf("[TerminalMgr] Invalid StreamID message length: %d", len(firstMsg.Data))
		return fmt.Errorf("invalid StreamID message")
	}

	magicHeader := []byte{0xff, 0x05, 0xff, 0x05}
	if !bytesEqual(firstMsg.Data[:4], magicHeader) {
		logrus.Errorf("[TerminalMgr] Invalid magic header: %x", firstMsg.Data[:4])
		return fmt.Errorf("invalid magic header")
	}

	streamID := string(firstMsg.Data[4:])
	if streamID == "" {
		logrus.Errorf("[TerminalMgr] Empty StreamID")
		return fmt.Errorf("empty StreamID")
	}

	logrus.Infof("[TerminalMgr] Agent connecting with StreamID: %s", streamID)

	// List all sessions for debugging
	sessionCount := 0
	m.sessions.Range(func(key, value interface{}) bool {
		sessionCount++
		logrus.Debugf("[TerminalMgr] Existing session: %s", key.(string))
		return true
	})
	logrus.Debugf("[TerminalMgr] Total sessions: %d", sessionCount)

	// Check if session exists
	sessionI, exists := m.sessions.Load(streamID)
	if !exists {
		logrus.Errorf("[TerminalMgr] Session not found: %s (have %d sessions)", streamID, sessionCount)
		return fmt.Errorf("session not found: %s", streamID)
	}

	session := sessionI.(*TerminalSession)
	session.mu.Lock()
	session.AgentStream = stream
	session.mu.Unlock()

	logrus.Infof("[TerminalMgr] Agent connected to terminal session: %s", streamID)

	// Forward messages from Agent to WebSocket
	go func() {
		defer func() {
			session.mu.Lock()
			session.closed = true
			session.mu.Unlock()
			close(session.FromAgent)
		}()

		for {
			msg, err := stream.Recv()
			if err != nil {
				// Classify error type
				errorType := classifyStreamError(err)
				logrus.Errorf("[TerminalMgr] IOStream recv error (%s): %v", errorType, err)

				// Store error for diagnostics
				session.mu.Lock()
				session.LastError = err
				session.mu.Unlock()

				// Send error notification (non-blocking)
				select {
				case session.ErrorChan <- err:
				default:
				}

				return
			}

			select {
			case session.FromAgent <- msg.Data:
			case <-ctx.Done():
				return
			case <-session.ctx.Done():
				return
			}
		}
	}()

	// Forward messages from WebSocket to Agent
	for {
		select {
		case data := <-session.ToAgent:
			if err := stream.Send(&proto.IOStreamData{Data: data}); err != nil {
				logrus.Errorf("IOStream send error: %v", err)
				return err
			}

		case <-ctx.Done():
			return ctx.Err()

		case <-session.ctx.Done():
			return nil
		}
	}
}

// CreateSession creates a new terminal session
func (m *TerminalManager) CreateSession(streamID string, hostID uuid.UUID, uuid string) *TerminalSession {
	ctx, cancel := context.WithCancel(context.Background())

	session := &TerminalSession{
		StreamID:  streamID,
		HostID:    hostID,
		UUID:      uuid,
		StartedAt: time.Now(),
		ToAgent:   make(chan []byte, 100),
		FromAgent: make(chan []byte, 100),
		ErrorChan: make(chan error, 10), // Buffered for non-blocking sends
		ctx:       ctx,
		cancel:    cancel,
	}

	m.sessions.Store(streamID, session)
	logrus.Infof("[TerminalMgr] Created terminal session: %s for host %s (agent: %s)", streamID, hostID, uuid)

	// Verify it was stored
	if _, exists := m.sessions.Load(streamID); exists {
		logrus.Debugf("[TerminalMgr] Session %s successfully stored and verified", streamID)
	} else {
		logrus.Errorf("[TerminalMgr] Session %s FAILED to store!", streamID)
	}

	return session
}

// GetSession retrieves a session by StreamID
func (m *TerminalManager) GetSession(streamID string) (*TerminalSession, bool) {
	sessionI, exists := m.sessions.Load(streamID)
	if !exists {
		return nil, false
	}
	return sessionI.(*TerminalSession), true
}

// CloseSession closes and removes a terminal session
func (m *TerminalManager) CloseSession(streamID string) error {
	sessionI, exists := m.sessions.Load(streamID)
	if !exists {
		return fmt.Errorf("session not found: %s", streamID)
	}

	session := sessionI.(*TerminalSession)
	session.mu.Lock()
	defer session.mu.Unlock()

	if !session.closed {
		session.cancel()
		close(session.ToAgent)
		session.closed = true
	}

	m.sessions.Delete(streamID)
	logrus.Infof("Closed terminal session: %s", streamID)

	return nil
}

// SendToAgent sends data to the Agent
func (s *TerminalSession) SendToAgent(data []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return fmt.Errorf("session closed")
	}

	select {
	case s.ToAgent <- data:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("send timeout")
	}
}

// ReceiveFromAgent receives data from the Agent
func (s *TerminalSession) ReceiveFromAgent() ([]byte, error) {
	select {
	case data, ok := <-s.FromAgent:
		if !ok {
			return nil, io.EOF
		}
		return data, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

// bytesEqual compares two byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// GetLastError returns the last error that occurred
func (s *TerminalSession) GetLastError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastError
}

// classifyStreamError classifies stream errors for better diagnostics
func classifyStreamError(err error) string {
	if err == nil {
		return "no-error"
	}

	// Check for io.EOF
	if errors.Is(err, io.EOF) {
		return "eof"
	}

	// Check for context errors
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline-exceeded"
	}

	// Check for gRPC status errors
	if st, ok := status.FromError(err); ok {
		code := st.Code()
		switch code {
		case codes.Canceled:
			return "grpc-canceled"
		case codes.DeadlineExceeded:
			return "grpc-deadline-exceeded"
		case codes.Unavailable:
			return "grpc-unavailable"
		case codes.ResourceExhausted:
			return "grpc-resource-exhausted"
		case codes.Aborted:
			return "grpc-aborted"
		case codes.Internal:
			return "grpc-internal"
		case codes.Unknown:
			return "grpc-unknown"
		default:
			return fmt.Sprintf("grpc-%s", code.String())
		}
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "network-timeout"
		}
		if netErr.Temporary() {
			return "network-temporary"
		}
		return "network-error"
	}

	// Generic error
	return "unknown-error"
}

// IsRecoverableError checks if an error is potentially recoverable
func IsRecoverableError(err error) bool {
	if err == nil {
		return false
	}

	errorType := classifyStreamError(err)

	// These error types might be recoverable with retry
	recoverableTypes := map[string]bool{
		"network-timeout":         true,
		"network-temporary":       true,
		"grpc-unavailable":        true,
		"grpc-deadline-exceeded":  true,
		"grpc-resource-exhausted": true,
		"grpc-aborted":            true,
	}

	return recoverableTypes[errorType]
}
