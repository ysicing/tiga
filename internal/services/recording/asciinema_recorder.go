package recording

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// AsciinemaRecorder implements Asciinema v2 format recording
// Reference: 010-k8s-pod-009 T016
//
// Format:
// Line 1 (Header): {"version": 2, "width": 120, "height": 40, "timestamp": 1234567890, "title": "..."}
// Line 2+ (Frames): [elapsed_seconds, "o", "data"]
type AsciinemaRecorder struct {
	file      *os.File
	startTime time.Time
	recording bool
	mutex     sync.Mutex
}

// AsciinemaHeader represents the Asciinema v2 header (first line)
type AsciinemaHeader struct {
	Version   int    `json:"version"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Timestamp int64  `json:"timestamp"`
	Title     string `json:"title,omitempty"`
}

// AsciinemaFrame represents a single frame [timestamp, type, data]
type AsciinemaFrame struct {
	Timestamp float64
	Type      string
	Data      string
}

// NewAsciinemaRecorder creates a new Asciinema v2 recorder
// filePath: absolute path to .cast file
// width, height: terminal dimensions
// title: recording title (optional)
func NewAsciinemaRecorder(filePath string, width, height int, title string) (*AsciinemaRecorder, error) {
	// Ensure parent directory exists
	dir := filePath[:len(filePath)-len(filePath[len(filePath)-1:])]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create recording directory: %w", err)
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create recording file: %w", err)
	}

	recorder := &AsciinemaRecorder{
		file:      file,
		startTime: time.Now(),
		recording: true,
	}

	// Write Asciinema v2 header (first line)
	header := AsciinemaHeader{
		Version:   2,
		Width:     width,
		Height:    height,
		Timestamp: recorder.startTime.Unix(),
		Title:     title,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to marshal header: %w", err)
	}

	if _, err := file.Write(append(headerJSON, '\n')); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	return recorder, nil
}

// WriteFrame writes a single frame to the recording
// frameType: "o" for output (from terminal), "i" for input (from user)
// data: raw terminal data
func (r *AsciinemaRecorder) WriteFrame(frameType string, data []byte) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.recording {
		return fmt.Errorf("recorder is stopped")
	}

	if len(data) == 0 {
		return nil // Skip empty frames
	}

	// Calculate elapsed time in seconds (with millisecond precision)
	elapsed := time.Since(r.startTime).Seconds()

	// Create frame: [timestamp, type, data]
	frame := []interface{}{
		elapsed,
		frameType,
		string(data),
	}

	frameJSON, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("failed to marshal frame: %w", err)
	}

	// Write frame as a new line
	if _, err := r.file.Write(append(frameJSON, '\n')); err != nil {
		return fmt.Errorf("failed to write frame: %w", err)
	}

	return nil
}

// Stop stops the recording and closes the file
func (r *AsciinemaRecorder) Stop() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.recording {
		return nil // Already stopped
	}

	r.recording = false

	if err := r.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	if err := r.file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	return nil
}

// IsRecording returns whether the recorder is currently recording
func (r *AsciinemaRecorder) IsRecording() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.recording
}

// GetDuration returns the current recording duration in seconds
func (r *AsciinemaRecorder) GetDuration() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return int(time.Since(r.startTime).Seconds())
}

// GetFilePath returns the file path of the recording
func (r *AsciinemaRecorder) GetFilePath() string {
	if r.file == nil {
		return ""
	}
	return r.file.Name()
}
