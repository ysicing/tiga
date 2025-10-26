package recording

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// ManagerService provides business logic for terminal recording management
// Handles CRUD operations, search, statistics, and playback
type ManagerService struct {
	repo           repository.RecordingRepositoryInterface
	storageService StorageServiceInterface
}

// NewManagerService creates a new manager service instance
func NewManagerService(
	repo repository.RecordingRepositoryInterface,
	storageService StorageServiceInterface,
) *ManagerService {
	return &ManagerService{
		repo:           repo,
		storageService: storageService,
	}
}

// ListRecordings retrieves recordings with filtering, pagination, and sorting
func (s *ManagerService) ListRecordings(
	ctx context.Context,
	filters repository.RecordingFilters,
	page, limit int,
) ([]*models.TerminalRecording, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20 // Default limit
	}

	recordings, total, err := s.repo.List(ctx, filters, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list recordings: %w", err)
	}

	return recordings, total, nil
}

// GetRecording retrieves a recording by ID
func (s *ManagerService) GetRecording(ctx context.Context, id uuid.UUID) (*models.TerminalRecording, error) {
	recording, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("recording not found: %w", err)
	}

	return recording, nil
}

// GetRecordingBySessionID retrieves a recording by session ID
func (s *ManagerService) GetRecordingBySessionID(ctx context.Context, sessionID uuid.UUID) (*models.TerminalRecording, error) {
	recording, err := s.repo.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("recording not found: %w", err)
	}

	return recording, nil
}

// DeleteRecording deletes a recording and its associated file
// Includes RBAC permission check (should be enforced by handler)
func (s *ManagerService) DeleteRecording(ctx context.Context, id uuid.UUID) error {
	// Get recording first
	recording, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("recording not found: %w", err)
	}

	// Delete file first
	if err := s.storageService.DeleteRecording(recording.StoragePath); err != nil {
		logrus.Warnf("[ManagerService] Failed to delete recording file %s: %v", recording.StoragePath, err)
		// Continue to delete database record even if file deletion fails
	}

	// Delete database record
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete recording: %w", err)
	}

	logrus.Infof("[ManagerService] Deleted recording %s (user: %s, type: %s)",
		id, recording.Username, recording.RecordingType)

	return nil
}

// SearchRecordings performs full-text search on recordings
func (s *ManagerService) SearchRecordings(
	ctx context.Context,
	query string,
	page, limit int,
) ([]*models.TerminalRecording, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	recordings, total, err := s.repo.Search(ctx, query, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("search failed: %w", err)
	}

	return recordings, total, nil
}

// GetStatistics retrieves aggregated statistics about recordings
func (s *ManagerService) GetStatistics(ctx context.Context) (*repository.RecordingStatistics, error) {
	stats, err := s.repo.GetStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	return stats, nil
}

// GetPlaybackContent retrieves recording content in Asciinema v2 format
// Returns an io.ReadCloser for streaming the .cast file
func (s *ManagerService) GetPlaybackContent(ctx context.Context, id uuid.UUID) (io.ReadCloser, error) {
	// Get recording metadata
	recording, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("recording not found: %w", err)
	}

	// Check if recording is completed
	if recording.EndedAt == nil {
		return nil, fmt.Errorf("recording is still in progress")
	}

	// Open recording file
	reader, err := s.storageService.ReadRecording(recording.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read recording file: %w", err)
	}

	// Fix header if width/height are 0 (legacy recordings)
	if recording.Cols == 0 || recording.Rows == 0 {
		return s.fixAsciinemaHeader(reader, recording)
	}

	return reader, nil
}

// fixAsciinemaHeader fixes Asciinema header with correct dimensions from recording metadata
func (s *ManagerService) fixAsciinemaHeader(reader io.ReadCloser, recording *models.TerminalRecording) (io.ReadCloser, error) {
	defer reader.Close()

	// Read entire file content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	// Split into lines
	lines := make([][]byte, 0)
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		lines = append(lines, append([]byte{}, scanner.Bytes()...))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}

	if len(lines) == 0 {
		return nil, fmt.Errorf("empty recording file")
	}

	// Parse existing header
	var header map[string]interface{}
	if err := json.Unmarshal(lines[0], &header); err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	// Fix width and height
	cols := recording.Cols
	rows := recording.Rows
	if cols == 0 {
		cols = 120 // Default fallback
	}
	if rows == 0 {
		rows = 30 // Default fallback
	}

	header["width"] = cols
	header["height"] = rows

	// Marshal fixed header
	fixedHeader, err := json.Marshal(header)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fixed header: %w", err)
	}

	// Reconstruct file content with fixed header
	var buf bytes.Buffer
	buf.Write(fixedHeader)
	buf.WriteByte('\n')
	for i := 1; i < len(lines); i++ {
		buf.Write(lines[i])
		buf.WriteByte('\n')
	}

	return io.NopCloser(&buf), nil
}

// DownloadRecording provides a recording file for download
// Returns file reader and metadata for setting headers
func (s *ManagerService) DownloadRecording(ctx context.Context, id uuid.UUID) (io.ReadCloser, string, error) {
	// Get recording metadata
	recording, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("recording not found: %w", err)
	}

	// Check if recording is completed
	if recording.EndedAt == nil {
		return nil, "", fmt.Errorf("recording is still in progress")
	}

	// Open recording file
	reader, err := s.storageService.ReadRecording(recording.StoragePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read recording file: %w", err)
	}

	// Generate filename for download
	// Format: {username}_{recording_type}_{started_at}.cast
	filename := fmt.Sprintf("%s_%s_%s.cast",
		recording.Username,
		recording.RecordingType,
		recording.StartedAt.Format("20060102_150405"),
	)

	return reader, filename, nil
}

// CreateRecording creates a new recording record
// Used by terminal handlers when starting a new session
func (s *ManagerService) CreateRecording(ctx context.Context, recording *models.TerminalRecording) error {
	// Ensure base directory exists
	if err := s.storageService.EnsureBaseDir(); err != nil {
		return fmt.Errorf("failed to ensure base directory: %w", err)
	}

	// Generate storage path
	storagePath := s.storageService.GetRecordingPath(recording.ID, recording.StartedAt)
	recording.StoragePath = storagePath

	// Create recording record
	if err := s.repo.Create(ctx, recording); err != nil {
		return fmt.Errorf("failed to create recording: %w", err)
	}

	logrus.Infof("[ManagerService] Created recording %s (user: %s, type: %s)",
		recording.ID, recording.Username, recording.RecordingType)

	return nil
}

// FinalizeRecording updates recording metadata when session ends
// Used by terminal handlers when closing a session
func (s *ManagerService) FinalizeRecording(
	ctx context.Context,
	recordingID uuid.UUID,
	endedAt time.Time,
	duration int,
) error {
	// Get recording
	recording, err := s.repo.GetByID(ctx, recordingID)
	if err != nil {
		return fmt.Errorf("recording not found: %w", err)
	}

	// Check if already finalized
	if recording.EndedAt != nil {
		logrus.Warnf("[ManagerService] Recording %s already finalized", recordingID)
		return nil
	}

	// Get file size
	fileInfo, err := os.Stat(recording.StoragePath)
	var fileSize int64
	if err != nil {
		logrus.Warnf("[ManagerService] Failed to get file size for %s: %v", recording.StoragePath, err)
		fileSize = 0
	} else {
		fileSize = fileInfo.Size()
	}

	// Update recording
	recording.EndedAt = &endedAt
	recording.Duration = duration
	recording.FileSize = fileSize

	if err := s.repo.Update(ctx, recording); err != nil {
		return fmt.Errorf("failed to finalize recording: %w", err)
	}

	logrus.Infof("[ManagerService] Finalized recording %s (duration: %ds, size: %d bytes)",
		recordingID, duration, fileSize)

	return nil
}

// WriteRecordingData writes streaming data to recording file
// Used by terminal handlers during active sessions
func (s *ManagerService) WriteRecordingData(
	recordingID uuid.UUID,
	startedAt time.Time,
	data io.Reader,
) (string, int64, error) {
	storagePath, written, err := s.storageService.WriteRecording(recordingID, startedAt, data)
	if err != nil {
		return "", 0, fmt.Errorf("failed to write recording data: %w", err)
	}

	return storagePath, written, nil
}

// ValidateAsciinemaFormat validates if the recording file is valid Asciinema v2 format
// Checks header line and frame format
func (s *ManagerService) ValidateAsciinemaFormat(storagePath string) error {
	file, err := os.Open(storagePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Check first line (header)
	if !scanner.Scan() {
		return fmt.Errorf("empty file")
	}

	headerLine := scanner.Text()
	if len(headerLine) == 0 {
		return fmt.Errorf("invalid asciinema file: empty header")
	}

	// Header should be valid JSON with "version" field
	// Simple validation: check if starts with '{'
	if headerLine[0] != '{' {
		return fmt.Errorf("invalid asciinema file: header must be JSON")
	}

	// Optional: Check a few frame lines (should be JSON arrays)
	frameCount := 0
	for scanner.Scan() && frameCount < 5 {
		line := scanner.Text()
		if len(line) > 0 && line[0] != '[' {
			return fmt.Errorf("invalid asciinema file: frame must be JSON array")
		}
		frameCount++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan file: %w", err)
	}

	return nil
}
