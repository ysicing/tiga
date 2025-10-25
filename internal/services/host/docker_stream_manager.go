package host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/proto"
)

// DockerStreamSession represents an active Docker streaming operation session
type DockerStreamSession struct {
	SessionID   string
	InstanceID  uuid.UUID
	Operation   string // exec_container, get_logs, get_stats, pull_image, get_events
	ContainerID string
	ImageName   string
	Params      map[string]string
	StartedAt   time.Time

	// gRPC stream from Agent
	AgentStream proto.HostMonitor_DockerStreamClient

	// Channels for data flow
	DataChan  chan *proto.DockerStreamData  // Agent → User
	ErrorChan chan *proto.DockerStreamError // Agent → User
	CloseChan chan *proto.DockerStreamClose // Agent → User
	InputChan chan *proto.DockerStreamMessage // Server → Agent (for stdin, resize, etc.)

	// Session control
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	closed bool
	ready  bool // Agent confirmed ready
}

// DockerStreamManager manages active Docker stream sessions
type DockerStreamManager struct {
	sessions sync.Map // SessionID -> *DockerStreamSession
}

// NewDockerStreamManager creates a new Docker stream manager
func NewDockerStreamManager() *DockerStreamManager {
	return &DockerStreamManager{}
}

// HandleDockerStream handles the DockerStream RPC from Agent
func (m *DockerStreamManager) HandleDockerStream(stream proto.HostMonitor_DockerStreamServer) error {
	ctx := stream.Context()

	// First message should be Init from Agent
	firstMsg, err := stream.Recv()
	if err != nil {
		logrus.Errorf("[DockerStreamMgr] Failed to receive init message: %v", err)
		return fmt.Errorf("failed to receive init message: %w", err)
	}

	// Extract init message
	init, ok := firstMsg.Message.(*proto.DockerStreamMessage_Init)
	if !ok {
		logrus.Errorf("[DockerStreamMgr] First message is not Init: %T", firstMsg.Message)
		return fmt.Errorf("first message is not Init")
	}

	sessionID := init.Init.SessionId
	if sessionID == "" {
		logrus.Errorf("[DockerStreamMgr] Empty SessionID")
		return fmt.Errorf("empty SessionID")
	}

	logrus.Infof("[DockerStreamMgr] Agent connecting with SessionID: %s", sessionID)

	// Find session
	sessionI, exists := m.sessions.Load(sessionID)
	if !exists {
		logrus.Errorf("[DockerStreamMgr] Session not found: %s", sessionID)
		// Send error to Agent
		stream.Send(&proto.DockerStreamMessage{
			Message: &proto.DockerStreamMessage_Error{
				Error: &proto.DockerStreamError{
					SessionId: sessionID,
					Error:     "session not found",
				},
			},
		})
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session := sessionI.(*DockerStreamSession)

	// Send ready confirmation to Agent
	if err := stream.Send(&proto.DockerStreamMessage{
		Message: &proto.DockerStreamMessage_Init{
			Init: &proto.DockerStreamInit{
				SessionId: sessionID,
				Ready:     true,
			},
		},
	}); err != nil {
		logrus.Errorf("[DockerStreamMgr] Failed to send ready confirmation: %v", err)
		return err
	}

	session.mu.Lock()
	session.ready = true
	session.mu.Unlock()

	logrus.Infof("[DockerStreamMgr] Agent ready for session: %s, operation: %s", sessionID, session.Operation)

	// Goroutine: Forward messages from Server (InputChan) to Agent stream
	go func() {
		for {
			select {
			case msg, ok := <-session.InputChan:
				if !ok {
					return // InputChan closed
				}
				if err := stream.Send(msg); err != nil {
					logrus.Errorf("[DockerStreamMgr] Failed to send to Agent: %v", err)
					return
				}
			case <-ctx.Done():
				return
			case <-session.ctx.Done():
				return
			}
		}
	}()

	// Goroutine: Forward messages from Agent to channels
	go func() {
		defer func() {
			session.mu.Lock()
			session.closed = true
			session.mu.Unlock()
			close(session.DataChan)
			close(session.ErrorChan)
			close(session.CloseChan)
		}()

		for {
			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					logrus.Infof("[DockerStreamMgr] Agent closed stream: %s", sessionID)
				} else {
					logrus.Errorf("[DockerStreamMgr] Stream recv error: %v", err)
					// Send error to user
					select {
					case session.ErrorChan <- &proto.DockerStreamError{
						SessionId: sessionID,
						Error:     err.Error(),
					}:
					default:
					}
				}
				return
			}

			// Route message to appropriate channel
			switch m := msg.Message.(type) {
			case *proto.DockerStreamMessage_Data:
				select {
				case session.DataChan <- m.Data:
				case <-ctx.Done():
					return
				case <-session.ctx.Done():
					return
				}

			case *proto.DockerStreamMessage_Error:
				select {
				case session.ErrorChan <- m.Error:
				case <-ctx.Done():
					return
				case <-session.ctx.Done():
					return
				}

			case *proto.DockerStreamMessage_Close:
				select {
				case session.CloseChan <- m.Close:
				case <-ctx.Done():
					return
				case <-session.ctx.Done():
					return
				}
				return // Stream closed

			default:
				logrus.Warnf("[DockerStreamMgr] Unknown message type: %T", m)
			}
		}
	}()

	// Wait for session context cancellation
	<-session.ctx.Done()
	logrus.Infof("[DockerStreamMgr] Session context done: %s", sessionID)
	return nil
}

// CreateSession creates a new Docker stream session and triggers Agent to connect
func (m *DockerStreamManager) CreateSession(instanceID uuid.UUID, operation string, containerID, imageName string, params map[string]string) (*DockerStreamSession, error) {
	sessionID := uuid.New().String()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // 5 min timeout

	session := &DockerStreamSession{
		SessionID:   sessionID,
		InstanceID:  instanceID,
		Operation:   operation,
		ContainerID: containerID,
		ImageName:   imageName,
		Params:      params,
		StartedAt:   time.Now(),
		DataChan:    make(chan *proto.DockerStreamData, 100),
		ErrorChan:   make(chan *proto.DockerStreamError, 10),
		CloseChan:   make(chan *proto.DockerStreamClose, 1),
		InputChan:   make(chan *proto.DockerStreamMessage, 100), // Server → Agent
		ctx:         ctx,
		cancel:      cancel,
	}

	m.sessions.Store(sessionID, session)
	logrus.Infof("[DockerStreamMgr] Created session: %s, instance: %s, operation: %s", sessionID, instanceID, operation)

	// TODO: Send task to Agent to initiate DockerStream connection
	// This will be handled by queueing a task through AgentManager

	return session, nil
}

// GetSession retrieves a session by SessionID
func (m *DockerStreamManager) GetSession(sessionID string) (*DockerStreamSession, bool) {
	sessionI, exists := m.sessions.Load(sessionID)
	if !exists {
		return nil, false
	}
	return sessionI.(*DockerStreamSession), true
}

// CloseSession closes and removes a Docker stream session
func (m *DockerStreamManager) CloseSession(sessionID string) error {
	sessionI, exists := m.sessions.Load(sessionID)
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session := sessionI.(*DockerStreamSession)
	session.mu.Lock()
	defer session.mu.Unlock()

	if !session.closed {
		session.cancel()
		session.closed = true
		close(session.InputChan) // Close input channel
	}

	m.sessions.Delete(sessionID)
	logrus.Infof("[DockerStreamMgr] Closed session: %s", sessionID)

	return nil
}

// WaitForReady waits for Agent to connect and confirm ready
func (s *DockerStreamSession) WaitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s.mu.RLock()
		ready := s.ready
		s.mu.RUnlock()

		if ready {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return errors.New("timeout waiting for Agent to be ready")
}

// ReceiveData receives data from the Docker stream
func (s *DockerStreamSession) ReceiveData() (*proto.DockerStreamData, error) {
	select {
	case data, ok := <-s.DataChan:
		if !ok {
			return nil, io.EOF
		}
		return data, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

// ReceiveError receives error from the Docker stream
func (s *DockerStreamSession) ReceiveError() (*proto.DockerStreamError, error) {
	select {
	case err, ok := <-s.ErrorChan:
		if !ok {
			return nil, io.EOF
		}
		return err, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

// ReceiveClose receives close message from the Docker stream
func (s *DockerStreamSession) ReceiveClose() (*proto.DockerStreamClose, error) {
	select {
	case close, ok := <-s.CloseChan:
		if !ok {
			return nil, io.EOF
		}
		return close, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

// IsClosed returns whether the session is closed
func (s *DockerStreamSession) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// IsReady returns whether Agent is ready
func (s *DockerStreamSession) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready
}
