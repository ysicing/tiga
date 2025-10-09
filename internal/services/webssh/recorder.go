package webssh

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// AsciinemaHeader represents the header of an asciinema recording (v2 format)
type AsciinemaHeader struct {
	Version   int               `json:"version"`
	Width     int               `json:"width"`
	Height    int               `json:"height"`
	Timestamp int64             `json:"timestamp"`
	Env       map[string]string `json:"env,omitempty"`
}

// AsciinemaEvent represents a single event in the recording
type AsciinemaEvent struct {
	Time      float64 `json:"time"`      // Time offset from start in seconds
	EventType string  `json:"eventType"` // "o" for output, "i" for input
	Data      string  `json:"data"`      // The actual data
}

// SessionRecorder records terminal session in asciicast v2 format
type SessionRecorder struct {
	sessionID    string
	file         *os.File
	writer       *bufio.Writer
	startTime    time.Time
	recordDir    string
	cols         int
	rows         int
	mu           sync.Mutex
	closed       bool
	bytesWritten int64
}

// NewSessionRecorder creates a new session recorder
func NewSessionRecorder(sessionID string, cols, rows int, recordDir string) (*SessionRecorder, error) {
	// Ensure recording directory exists
	if err := os.MkdirAll(recordDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create recording directory: %w", err)
	}

	// Create recording file
	filename := filepath.Join(recordDir, fmt.Sprintf("%s.cast", sessionID))
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create recording file: %w", err)
	}

	recorder := &SessionRecorder{
		sessionID: sessionID,
		file:      file,
		writer:    bufio.NewWriter(file),
		startTime: time.Now(),
		recordDir: recordDir,
		cols:      cols,
		rows:      rows,
	}

	// Write asciicast v2 header
	header := AsciinemaHeader{
		Version:   2,
		Width:     cols,
		Height:    rows,
		Timestamp: time.Now().Unix(),
		Env: map[string]string{
			"SHELL": "/bin/bash",
			"TERM":  "xterm-256color",
		},
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to marshal header: %w", err)
	}

	if _, err := recorder.writer.Write(append(headerBytes, '\n')); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	if err := recorder.writer.Flush(); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to flush header: %w", err)
	}

	logrus.Infof("Created recording for session %s at %s", sessionID, filename)
	return recorder, nil
}

// RecordOutput records terminal output
func (r *SessionRecorder) RecordOutput(data []byte) error {
	return r.recordEvent("o", data)
}

// RecordInput records terminal input
func (r *SessionRecorder) RecordInput(data []byte) error {
	return r.recordEvent("i", data)
}

// recordEvent writes a single event to the recording file
func (r *SessionRecorder) recordEvent(eventType string, data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return fmt.Errorf("recorder is closed")
	}

	// Calculate time offset from start
	elapsed := time.Since(r.startTime).Seconds()

	// Create event in asciicast v2 format: [time, "event_type", "data"]
	// Format: [timestamp, "o" or "i", data]
	event := []interface{}{
		elapsed,
		eventType,
		string(data),
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Write event line
	n, err := r.writer.Write(append(eventBytes, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	r.bytesWritten += int64(n)

	// Flush periodically to ensure data is persisted
	if r.bytesWritten%4096 == 0 {
		if err := r.writer.Flush(); err != nil {
			return fmt.Errorf("failed to flush: %w", err)
		}
	}

	return nil
}

// Resize updates the terminal size (not part of asciicast v2, but useful for logging)
func (r *SessionRecorder) Resize(cols, rows int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cols = cols
	r.rows = rows

	// In asciicast v2, resize events are not standard, but we can log them
	logrus.Debugf("Session %s resized to %dx%d", r.sessionID, cols, rows)
}

// Close finalizes and closes the recording
func (r *SessionRecorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	r.closed = true

	// Flush any remaining data
	if err := r.writer.Flush(); err != nil {
		logrus.Errorf("Failed to flush recorder: %v", err)
	}

	// Close file
	if err := r.file.Close(); err != nil {
		return fmt.Errorf("failed to close recording file: %w", err)
	}

	logrus.Infof("Closed recording for session %s (%d bytes)", r.sessionID, r.bytesWritten)
	return nil
}

// GetFilePath returns the path to the recording file
func (r *SessionRecorder) GetFilePath() string {
	return filepath.Join(r.recordDir, fmt.Sprintf("%s.cast", r.sessionID))
}

// GetBytesWritten returns the total bytes written
func (r *SessionRecorder) GetBytesWritten() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.bytesWritten
}

// GetDuration returns the recording duration
func (r *SessionRecorder) GetDuration() time.Duration {
	return time.Since(r.startTime)
}
