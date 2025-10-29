package kube

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/ysicing/tiga/internal/services/recording"
)

// K8sTerminalSessionType represents the type of terminal session
type K8sTerminalSessionType string

const (
	// SessionTypeNodeTerminal represents a node SSH terminal session
	SessionTypeNodeTerminal K8sTerminalSessionType = "node_terminal"
	// SessionTypePodExec represents a pod container exec session
	SessionTypePodExec K8sTerminalSessionType = "pod_exec"
)

// K8sTerminalSession represents a K8s terminal session with recording
// Reference: 010-k8s-pod-009 T017, data-model.md Entity 3
type K8sTerminalSession struct {
	// Session identity
	SessionID uuid.UUID
	Type      K8sTerminalSessionType

	// K8s context
	ClusterID     string
	NodeName      string            // For node terminal
	Namespace     string            // For pod exec
	PodName       string            // For pod exec
	ContainerName string            // For pod exec
	Labels        map[string]string // Additional metadata

	// Recording
	RecordingID    *uuid.UUID
	RecordingState string // "active", "stopped", "timeout"
	Recorder       *recording.AsciinemaRecorder

	// Timing
	StartedAt time.Time
	Timer     *time.Timer // 2-hour timeout timer

	// Connection
	Conn  *websocket.Conn
	Mutex sync.Mutex
}

// NewK8sNodeTerminalSession creates a new node terminal session
func NewK8sNodeTerminalSession(sessionID uuid.UUID, clusterID, nodeName string, conn *websocket.Conn) *K8sTerminalSession {
	return &K8sTerminalSession{
		SessionID:      sessionID,
		Type:           SessionTypeNodeTerminal,
		ClusterID:      clusterID,
		NodeName:       nodeName,
		RecordingState: "inactive",
		StartedAt:      time.Now(),
		Conn:           conn,
	}
}

// NewK8sPodExecSession creates a new pod exec session
func NewK8sPodExecSession(sessionID uuid.UUID, clusterID, namespace, podName, containerName string, conn *websocket.Conn) *K8sTerminalSession {
	return &K8sTerminalSession{
		SessionID:      sessionID,
		Type:           SessionTypePodExec,
		ClusterID:      clusterID,
		Namespace:      namespace,
		PodName:        podName,
		ContainerName:  containerName,
		RecordingState: "inactive",
		StartedAt:      time.Now(),
		Conn:           conn,
	}
}

// StartRecording starts recording the terminal session
// recorder: AsciinemaRecorder instance
// recordingID: UUID of the TerminalRecording database record
func (s *K8sTerminalSession) StartRecording(recorder *recording.AsciinemaRecorder, recordingID uuid.UUID) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	s.Recorder = recorder
	s.RecordingID = &recordingID
	s.RecordingState = "active"

	// Start 2-hour timeout timer
	s.Timer = time.AfterFunc(2*time.Hour, func() {
		s.handleRecordingTimeout()
	})
}

// StopRecording stops the recording (called on normal disconnect)
func (s *K8sTerminalSession) StopRecording() error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	if s.RecordingState == "stopped" {
		return nil // Already stopped
	}

	// Stop timer if still running
	if s.Timer != nil {
		s.Timer.Stop()
		s.Timer = nil
	}

	// Stop recorder
	if s.Recorder != nil {
		if err := s.Recorder.Stop(); err != nil {
			return fmt.Errorf("failed to stop recorder: %w", err)
		}
	}

	s.RecordingState = "stopped"
	return nil
}

// handleRecordingTimeout handles the 2-hour recording timeout
// Stops recording but keeps terminal connection alive
func (s *K8sTerminalSession) handleRecordingTimeout() {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	if s.RecordingState != "active" {
		return // Already stopped
	}

	// Stop recording
	if s.Recorder != nil {
		s.Recorder.Stop()
	}

	s.RecordingState = "timeout"

	// Send WebSocket notification to client
	if s.Conn != nil {
		notification := map[string]interface{}{
			"type":    "recording_stopped",
			"reason":  "2_hour_limit_reached",
			"message": "Recording has been stopped after 2 hours. Terminal session continues.",
		}

		// Try to send notification (non-blocking, best effort)
		s.Conn.WriteJSON(notification)
	}
}

// WriteRecordingFrame writes a frame to the recording (if active)
// frameType: "o" for output, "i" for input
// data: terminal data
func (s *K8sTerminalSession) WriteRecordingFrame(frameType string, data []byte) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	if s.RecordingState != "active" || s.Recorder == nil {
		return nil // Recording not active, silently skip
	}

	return s.Recorder.WriteFrame(frameType, data)
}

// Close closes the terminal session and stops recording
func (s *K8sTerminalSession) Close() error {
	// Stop recording first
	if err := s.StopRecording(); err != nil {
		// Log error but continue closing
		return fmt.Errorf("error stopping recording during close: %w", err)
	}

	// Close WebSocket connection
	if s.Conn != nil {
		s.Conn.Close()
	}

	return nil
}

// GetDuration returns the session duration in seconds
func (s *K8sTerminalSession) GetDuration() int {
	return int(time.Since(s.StartedAt).Seconds())
}

// IsRecording returns whether recording is currently active
func (s *K8sTerminalSession) IsRecording() bool {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	return s.RecordingState == "active"
}

// GetRecordingDuration returns the current recording duration in seconds
func (s *K8sTerminalSession) GetRecordingDuration() int {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	if s.Recorder == nil {
		return 0
	}

	return s.Recorder.GetDuration()
}
