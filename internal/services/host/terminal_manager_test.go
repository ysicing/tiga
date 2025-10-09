package host

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestNewTerminalManager tests terminal manager creation
func TestNewTerminalManager(t *testing.T) {
	mgr := NewTerminalManager()
	assert.NotNil(t, mgr)
}

// TestCreateSession tests session creation
func TestCreateSession(t *testing.T) {
	mgr := NewTerminalManager()
	streamID := "test-stream-123"
	hostID := uuid.New()
	agentUUID := "agent-uuid-456"

	session := mgr.CreateSession(streamID, hostID, agentUUID)

	require.NotNil(t, session)
	assert.Equal(t, streamID, session.StreamID)
	assert.Equal(t, hostID, session.HostID)
	assert.Equal(t, agentUUID, session.UUID)
	assert.NotNil(t, session.ToAgent)
	assert.NotNil(t, session.FromAgent)
	assert.NotNil(t, session.ErrorChan)
	assert.NotNil(t, session.ctx)
	assert.NotNil(t, session.cancel)
	assert.False(t, session.StartedAt.IsZero())
	assert.False(t, session.closed)

	// Verify session can be retrieved
	retrieved, exists := mgr.GetSession(streamID)
	require.True(t, exists)
	assert.Equal(t, streamID, retrieved.StreamID)

	// Clean up
	mgr.CloseSession(streamID)
}

// TestGetSession tests session retrieval
func TestGetSession(t *testing.T) {
	mgr := NewTerminalManager()
	streamID := "test-stream-789"
	hostID := uuid.New()

	t.Run("session exists", func(t *testing.T) {
		mgr.CreateSession(streamID, hostID, "agent-uuid")
		session, exists := mgr.GetSession(streamID)
		assert.True(t, exists)
		assert.NotNil(t, session)
		assert.Equal(t, streamID, session.StreamID)
		mgr.CloseSession(streamID)
	})

	t.Run("session does not exist", func(t *testing.T) {
		session, exists := mgr.GetSession("nonexistent-stream")
		assert.False(t, exists)
		assert.Nil(t, session)
	})
}

// TestCloseSession tests session closure
func TestCloseSession(t *testing.T) {
	mgr := NewTerminalManager()
	streamID := "test-close-session"
	hostID := uuid.New()

	t.Run("close existing session", func(t *testing.T) {
		session := mgr.CreateSession(streamID, hostID, "agent-uuid")
		require.NotNil(t, session)

		// Close the session
		err := mgr.CloseSession(streamID)
		assert.NoError(t, err)

		// Verify session is closed
		session.mu.RLock()
		closed := session.closed
		session.mu.RUnlock()
		assert.True(t, closed)

		// Verify session is removed from manager
		_, exists := mgr.GetSession(streamID)
		assert.False(t, exists)

		// Verify context is canceled
		select {
		case <-session.ctx.Done():
			// Expected: context is done
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Context should be canceled")
		}
	})

	t.Run("close non-existent session", func(t *testing.T) {
		err := mgr.CloseSession("nonexistent-stream")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})

	t.Run("double close session", func(t *testing.T) {
		streamID := "test-double-close"
		session := mgr.CreateSession(streamID, hostID, "agent-uuid")
		require.NotNil(t, session)

		// First close
		err := mgr.CloseSession(streamID)
		assert.NoError(t, err)

		// Second close should fail (session already removed)
		err = mgr.CloseSession(streamID)
		assert.Error(t, err)
	})
}

// TestSendToAgent tests sending data to agent
func TestSendToAgent(t *testing.T) {
	mgr := NewTerminalManager()
	streamID := "test-send-to-agent"
	hostID := uuid.New()

	t.Run("send successfully", func(t *testing.T) {
		session := mgr.CreateSession(streamID, hostID, "agent-uuid")
		defer mgr.CloseSession(streamID)

		testData := []byte("test data")
		err := session.SendToAgent(testData)
		assert.NoError(t, err)

		// Verify data is in channel
		select {
		case received := <-session.ToAgent:
			assert.Equal(t, testData, received)
		case <-time.After(1 * time.Second):
			t.Fatal("Data not received in channel")
		}
	})

	t.Run("send to closed session", func(t *testing.T) {
		session := mgr.CreateSession("test-send-closed", hostID, "agent-uuid")
		mgr.CloseSession("test-send-closed")

		// Wait for close to complete
		time.Sleep(50 * time.Millisecond)

		testData := []byte("test data")
		err := session.SendToAgent(testData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session closed")
	})
}

// TestReceiveFromAgent tests receiving data from agent
func TestReceiveFromAgent(t *testing.T) {
	mgr := NewTerminalManager()
	streamID := "test-receive-from-agent"
	hostID := uuid.New()

	t.Run("receive successfully", func(t *testing.T) {
		session := mgr.CreateSession(streamID, hostID, "agent-uuid")
		defer mgr.CloseSession(streamID)

		testData := []byte("test data from agent")

		// Send data to FromAgent channel
		go func() {
			session.FromAgent <- testData
		}()

		// Receive data
		received, err := session.ReceiveFromAgent()
		assert.NoError(t, err)
		assert.Equal(t, testData, received)
	})

	t.Run("receive from closed channel", func(t *testing.T) {
		session := mgr.CreateSession("test-receive-closed", hostID, "agent-uuid")

		// Close FromAgent channel manually to simulate agent disconnect
		close(session.FromAgent)

		// Attempt to receive should return EOF
		_, err := session.ReceiveFromAgent()
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("receive with context cancellation", func(t *testing.T) {
		session := mgr.CreateSession("test-receive-canceled", hostID, "agent-uuid")

		// Cancel context immediately
		session.cancel()

		// Attempt to receive should return context error
		_, err := session.ReceiveFromAgent()
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

// TestGetLastError tests error tracking
func TestGetLastError(t *testing.T) {
	mgr := NewTerminalManager()
	streamID := "test-get-last-error"
	hostID := uuid.New()

	session := mgr.CreateSession(streamID, hostID, "agent-uuid")
	defer mgr.CloseSession(streamID)

	t.Run("no error initially", func(t *testing.T) {
		err := session.GetLastError()
		assert.NoError(t, err)
	})

	t.Run("error after setting", func(t *testing.T) {
		testError := errors.New("test error")
		session.mu.Lock()
		session.LastError = testError
		session.mu.Unlock()

		err := session.GetLastError()
		assert.Error(t, err)
		assert.Equal(t, testError, err)
	})
}

// TestBytesEqual tests byte slice comparison helper
func TestBytesEqual(t *testing.T) {
	t.Run("equal slices", func(t *testing.T) {
		a := []byte{0xff, 0x05, 0xff, 0x05}
		b := []byte{0xff, 0x05, 0xff, 0x05}
		assert.True(t, bytesEqual(a, b))
	})

	t.Run("different slices", func(t *testing.T) {
		a := []byte{0xff, 0x05, 0xff, 0x05}
		b := []byte{0xff, 0x05, 0xff, 0x06}
		assert.False(t, bytesEqual(a, b))
	})

	t.Run("different lengths", func(t *testing.T) {
		a := []byte{0xff, 0x05}
		b := []byte{0xff, 0x05, 0xff}
		assert.False(t, bytesEqual(a, b))
	})

	t.Run("empty slices", func(t *testing.T) {
		a := []byte{}
		b := []byte{}
		assert.True(t, bytesEqual(a, b))
	})
}

// TestClassifyStreamError tests error classification
func TestClassifyStreamError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedType   string
	}{
		{
			name:         "no error",
			err:          nil,
			expectedType: "no-error",
		},
		{
			name:         "io.EOF",
			err:          io.EOF,
			expectedType: "eof",
		},
		{
			name:         "context canceled",
			err:          context.Canceled,
			expectedType: "canceled",
		},
		{
			name:         "context deadline exceeded",
			err:          context.DeadlineExceeded,
			expectedType: "deadline-exceeded",
		},
		{
			name:         "grpc canceled",
			err:          status.Error(codes.Canceled, "canceled"),
			expectedType: "grpc-canceled",
		},
		{
			name:         "grpc deadline exceeded",
			err:          status.Error(codes.DeadlineExceeded, "deadline"),
			expectedType: "grpc-deadline-exceeded",
		},
		{
			name:         "grpc unavailable",
			err:          status.Error(codes.Unavailable, "unavailable"),
			expectedType: "grpc-unavailable",
		},
		{
			name:         "grpc resource exhausted",
			err:          status.Error(codes.ResourceExhausted, "exhausted"),
			expectedType: "grpc-resource-exhausted",
		},
		{
			name:         "grpc aborted",
			err:          status.Error(codes.Aborted, "aborted"),
			expectedType: "grpc-aborted",
		},
		{
			name:         "grpc internal",
			err:          status.Error(codes.Internal, "internal"),
			expectedType: "grpc-internal",
		},
		{
			name:         "grpc unknown",
			err:          status.Error(codes.Unknown, "unknown"),
			expectedType: "grpc-unknown",
		},
		{
			name:         "generic error",
			err:          errors.New("some error"),
			expectedType: "unknown-error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorType := classifyStreamError(tt.err)
			assert.Equal(t, tt.expectedType, errorType)
		})
	}
}

// TestIsRecoverableError tests error recoverability check
func TestIsRecoverableError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		recoverable  bool
	}{
		{
			name:        "nil error",
			err:         nil,
			recoverable: false,
		},
		{
			name:        "io.EOF - not recoverable",
			err:         io.EOF,
			recoverable: false,
		},
		{
			name:        "context canceled - not recoverable",
			err:         context.Canceled,
			recoverable: false,
		},
		{
			name:        "grpc unavailable - recoverable",
			err:         status.Error(codes.Unavailable, "unavailable"),
			recoverable: true,
		},
		{
			name:        "grpc deadline exceeded - recoverable",
			err:         status.Error(codes.DeadlineExceeded, "deadline"),
			recoverable: true,
		},
		{
			name:        "grpc resource exhausted - recoverable",
			err:         status.Error(codes.ResourceExhausted, "exhausted"),
			recoverable: true,
		},
		{
			name:        "grpc aborted - recoverable",
			err:         status.Error(codes.Aborted, "aborted"),
			recoverable: true,
		},
		{
			name:        "grpc internal - not recoverable",
			err:         status.Error(codes.Internal, "internal"),
			recoverable: false,
		},
		{
			name:        "generic error - not recoverable",
			err:         errors.New("some error"),
			recoverable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recoverable := IsRecoverableError(tt.err)
			assert.Equal(t, tt.recoverable, recoverable)
		})
	}
}

// TestNetworkErrorClassification tests network error classification
func TestNetworkErrorClassification(t *testing.T) {
	t.Run("network timeout error", func(t *testing.T) {
		// Create a mock timeout error
		err := &mockNetError{timeout: true, temporary: false}
		errorType := classifyStreamError(err)
		assert.Equal(t, "network-timeout", errorType)
		assert.True(t, IsRecoverableError(err))
	})

	t.Run("network temporary error", func(t *testing.T) {
		// Create a mock temporary error
		err := &mockNetError{timeout: false, temporary: true}
		errorType := classifyStreamError(err)
		assert.Equal(t, "network-temporary", errorType)
		assert.True(t, IsRecoverableError(err))
	})

	t.Run("network error - not timeout or temporary", func(t *testing.T) {
		err := &mockNetError{timeout: false, temporary: false}
		errorType := classifyStreamError(err)
		assert.Equal(t, "network-error", errorType)
		assert.False(t, IsRecoverableError(err))
	})
}

// TestMultipleSessions tests managing multiple sessions
func TestMultipleSessions(t *testing.T) {
	mgr := NewTerminalManager()
	hostID := uuid.New()

	// Create multiple sessions
	sessions := make([]string, 5)
	for i := 0; i < 5; i++ {
		streamID := uuid.New().String()
		sessions[i] = streamID
		session := mgr.CreateSession(streamID, hostID, "agent-uuid")
		require.NotNil(t, session)
	}

	// Verify all sessions exist
	for _, streamID := range sessions {
		session, exists := mgr.GetSession(streamID)
		assert.True(t, exists)
		assert.NotNil(t, session)
	}

	// Close all sessions
	for _, streamID := range sessions {
		err := mgr.CloseSession(streamID)
		assert.NoError(t, err)
	}

	// Verify all sessions are removed
	for _, streamID := range sessions {
		_, exists := mgr.GetSession(streamID)
		assert.False(t, exists)
	}
}

// TestSessionChannelBuffering tests channel buffering
func TestSessionChannelBuffering(t *testing.T) {
	mgr := NewTerminalManager()
	streamID := "test-channel-buffering"
	hostID := uuid.New()

	session := mgr.CreateSession(streamID, hostID, "agent-uuid")
	defer mgr.CloseSession(streamID)

	t.Run("ToAgent channel buffer", func(t *testing.T) {
		// Send multiple messages without receiving
		for i := 0; i < 10; i++ {
			err := session.SendToAgent([]byte{byte(i)})
			assert.NoError(t, err)
		}

		// Verify all messages are in channel
		for i := 0; i < 10; i++ {
			select {
			case data := <-session.ToAgent:
				assert.Equal(t, byte(i), data[0])
			case <-time.After(1 * time.Second):
				t.Fatalf("Expected message %d not received", i)
			}
		}
	})
}

// mockNetError is a mock implementation of net.Error for testing
type mockNetError struct {
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return "mock network error" }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

// Ensure mockNetError implements net.Error
var _ net.Error = (*mockNetError)(nil)
