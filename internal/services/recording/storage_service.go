package recording

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/config"
)

// StorageServiceInterface defines the interface for recording storage operations
type StorageServiceInterface interface {
	// WriteRecording writes recording data to storage and returns the storage path
	WriteRecording(recordingID uuid.UUID, startedAt time.Time, data io.Reader) (string, int64, error)

	// ReadRecording reads recording data from storage
	ReadRecording(storagePath string) (io.ReadCloser, error)

	// DeleteRecording deletes a recording file from storage
	DeleteRecording(storagePath string) error

	// GetRecordingPath generates the storage path for a recording
	GetRecordingPath(recordingID uuid.UUID, startedAt time.Time) string

	// EnsureBaseDir ensures the base directory exists
	EnsureBaseDir() error
}

// LocalStorageService implements StorageServiceInterface for local filesystem
type LocalStorageService struct {
	basePath string // Base directory for recordings
}

// NewLocalStorageService creates a new local filesystem storage service
func NewLocalStorageService(cfg *config.Config) *LocalStorageService {
	basePath := cfg.Recording.BasePath
	if basePath == "" {
		basePath = "./data/recordings" // Default path
	}

	return &LocalStorageService{
		basePath: basePath,
	}
}

// EnsureBaseDir ensures the base directory exists
func (s *LocalStorageService) EnsureBaseDir() error {
	return os.MkdirAll(s.basePath, 0755)
}

// GetRecordingPath generates the storage path for a recording
// Path format: {basePath}/{YYYY-MM-DD}/{recordingID}.cast
func (s *LocalStorageService) GetRecordingPath(recordingID uuid.UUID, startedAt time.Time) string {
	dateDir := startedAt.Format("2006-01-02")
	filename := fmt.Sprintf("%s.cast", recordingID.String())
	return filepath.Join(s.basePath, dateDir, filename)
}

// WriteRecording writes recording data to local filesystem
func (s *LocalStorageService) WriteRecording(recordingID uuid.UUID, startedAt time.Time, data io.Reader) (string, int64, error) {
	// Generate storage path
	storagePath := s.GetRecordingPath(recordingID, startedAt)

	// Ensure directory exists
	dir := filepath.Dir(storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create file
	file, err := os.Create(storagePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create file %s: %w", storagePath, err)
	}
	defer file.Close()

	// Write data
	written, err := io.Copy(file, data)
	if err != nil {
		return "", 0, fmt.Errorf("failed to write recording data: %w", err)
	}

	logrus.Debugf("Wrote recording %s to %s (%d bytes)", recordingID, storagePath, written)
	return storagePath, written, nil
}

// ReadRecording reads recording data from local filesystem
func (s *LocalStorageService) ReadRecording(storagePath string) (io.ReadCloser, error) {
	// Open file
	file, err := os.Open(storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("recording file not found: %s", storagePath)
		}
		return nil, fmt.Errorf("failed to open recording file: %w", err)
	}

	return file, nil
}

// DeleteRecording deletes a recording file from local filesystem
func (s *LocalStorageService) DeleteRecording(storagePath string) error {
	// Check if file exists
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		// File doesn't exist, consider it already deleted (idempotent)
		logrus.Debugf("Recording file already deleted or doesn't exist: %s", storagePath)
		return nil
	}

	// Delete file
	if err := os.Remove(storagePath); err != nil {
		return fmt.Errorf("failed to delete recording file %s: %w", storagePath, err)
	}

	logrus.Debugf("Deleted recording file: %s", storagePath)

	// Try to remove empty date directory (best effort, ignore errors)
	dir := filepath.Dir(storagePath)
	if err := os.Remove(dir); err == nil {
		logrus.Debugf("Removed empty date directory: %s", dir)
	}

	return nil
}

// BulkDelete deletes multiple recording files (used by cleanup service)
func (s *LocalStorageService) BulkDelete(storagePaths []string) error {
	var firstError error
	successCount := 0
	failCount := 0

	for _, path := range storagePaths {
		if err := s.DeleteRecording(path); err != nil {
			if firstError == nil {
				firstError = err
			}
			failCount++
			logrus.Warnf("Failed to delete recording file %s: %v", path, err)
		} else {
			successCount++
		}
	}

	if firstError != nil {
		logrus.Warnf("Bulk delete completed with errors: %d succeeded, %d failed", successCount, failCount)
		return fmt.Errorf("bulk delete partially failed: %w", firstError)
	}

	logrus.Debugf("Bulk delete completed successfully: %d files deleted", successCount)
	return nil
}
