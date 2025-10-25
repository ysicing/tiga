package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/repository"
)

// RecordingCleanupService handles cleanup of invalid terminal recordings
type RecordingCleanupService struct {
	recordingRepo repository.TerminalRecordingRepositoryInterface
}

// NewRecordingCleanupService creates a new recording cleanup service
func NewRecordingCleanupService(recordingRepo repository.TerminalRecordingRepositoryInterface) *RecordingCleanupService {
	return &RecordingCleanupService{
		recordingRepo: recordingRepo,
	}
}

// CleanupInvalidRecordings cleans up terminal recordings with zero file size or zero duration
// Returns the number of recordings cleaned up
func (s *RecordingCleanupService) CleanupInvalidRecordings(ctx context.Context) (int, error) {
	startTime := time.Now()

	logrus.Info("[RecordingCleanup] Starting cleanup of invalid terminal recordings")

	// Find recordings with zero file size or zero duration
	filter := &repository.TerminalRecordingFilter{
		Limit:     1000, // Process in batches of 1000
		Offset:    0,
		SortBy:    "created_at",
		SortOrder: "ASC",
	}

	totalCleaned := 0
	for {
		// Get next batch
		recordings, total, err := s.recordingRepo.List(ctx, filter)
		if err != nil {
			return totalCleaned, fmt.Errorf("failed to list recordings: %w", err)
		}

		if len(recordings) == 0 {
			break
		}

		// Filter invalid recordings (zero file size or zero duration)
		var invalidIDs []uuid.UUID
		for _, recording := range recordings {
			if recording.FileSize == 0 || recording.Duration == 0 {
				invalidIDs = append(invalidIDs, recording.ID)
				logrus.Debugf("[RecordingCleanup] Found invalid recording: ID=%s, FileSize=%d, Duration=%d",
					recording.ID.String(), recording.FileSize, recording.Duration)
			}
		}

		// Delete invalid recordings
		if len(invalidIDs) > 0 {
			for _, id := range invalidIDs {
				if err := s.recordingRepo.Delete(ctx, id); err != nil {
					logrus.Warnf("[RecordingCleanup] Failed to delete recording %s: %v", id.String(), err)
					continue
				}
				totalCleaned++
			}
		}

		// Check if we've processed all records
		if len(recordings) < filter.Limit || filter.Offset+filter.Limit >= int(total) {
			break
		}

		// Move to next batch
		filter.Offset += filter.Limit
	}

	duration := time.Since(startTime)
	logrus.Infof("[RecordingCleanup] Cleanup completed: cleaned %d invalid recordings in %v", totalCleaned, duration)

	return totalCleaned, nil
}

// CleanupTask implements scheduler.Task interface
type RecordingCleanupTask struct {
	service *RecordingCleanupService
}

// NewRecordingCleanupTask creates a new recording cleanup task
func NewRecordingCleanupTask(service *RecordingCleanupService) *RecordingCleanupTask {
	return &RecordingCleanupTask{
		service: service,
	}
}

// Run executes the cleanup task
func (t *RecordingCleanupTask) Run(ctx context.Context) error {
	cleaned, err := t.service.CleanupInvalidRecordings(ctx)
	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	logrus.Infof("[RecordingCleanupTask] Successfully cleaned %d invalid recordings", cleaned)
	return nil
}

// Name returns the task name
func (t *RecordingCleanupTask) Name() string {
	return "Terminal Recording Cleanup - Clean up recordings with zero file size or duration"
}
