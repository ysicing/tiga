package host

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/proto"
)

// TerminalSession represents an active terminal session
type TerminalSession struct {
	StreamID  string
	HostID    uint
	UUID      string
	StartedAt time.Time

	// gRPC stream from Agent
	AgentStream proto.HostMonitor_IOStreamServer

	// Buffered channels for bidirectional communication
	ToAgent   chan []byte
	FromAgent chan []byte

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
		return fmt.Errorf("failed to receive StreamID: %w", err)
	}

	// Parse StreamID (format: 0xff 0x05 0xff 0x05 + streamID)
	if len(firstMsg.Data) < 5 {
		return fmt.Errorf("invalid StreamID message")
	}

	magicHeader := []byte{0xff, 0x05, 0xff, 0x05}
	if !bytesEqual(firstMsg.Data[:4], magicHeader) {
		return fmt.Errorf("invalid magic header")
	}

	streamID := string(firstMsg.Data[4:])
	if streamID == "" {
		return fmt.Errorf("empty StreamID")
	}

	// Check if session exists
	sessionI, exists := m.sessions.Load(streamID)
	if !exists {
		return fmt.Errorf("session not found: %s", streamID)
	}

	session := sessionI.(*TerminalSession)
	session.mu.Lock()
	session.AgentStream = stream
	session.mu.Unlock()

	logrus.Infof("Agent connected to terminal session: %s", streamID)

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
				if err != io.EOF {
					logrus.Errorf("IOStream recv error: %v", err)
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
func (m *TerminalManager) CreateSession(streamID string, hostID uint, uuid string) *TerminalSession {
	ctx, cancel := context.WithCancel(context.Background())

	session := &TerminalSession{
		StreamID:  streamID,
		HostID:    hostID,
		UUID:      uuid,
		StartedAt: time.Now(),
		ToAgent:   make(chan []byte, 100),
		FromAgent: make(chan []byte, 100),
		ctx:       ctx,
		cancel:    cancel,
	}

	m.sessions.Store(streamID, session)
	logrus.Infof("Created terminal session: %s for host %d", streamID, hostID)

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
