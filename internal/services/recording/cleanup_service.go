package recording

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// CleanupService provides cleanup logic for terminal recordings
// Handles expired recordings, invalid recordings, and orphan files
type CleanupService struct {
	repo           repository.RecordingRepositoryInterface
	storageService StorageServiceInterface
	config         *config.Config
}

// NewCleanupService creates a new cleanup service instance
func NewCleanupService(
	repo repository.RecordingRepositoryInterface,
	storageService StorageServiceInterface,
	cfg *config.Config,
) *CleanupService {
	return &CleanupService{
		repo:           repo,
		storageService: storageService,
		config:         cfg,
	}
}

// CleanupInvalidRecordings cleans up recordings with zero file size or zero duration
// Uses batch processing for efficient deletion
func (s *CleanupService) CleanupInvalidRecordings(ctx context.Context) (int, error) {
	const batchSize = 1000
	totalDeleted := 0

	logrus.Info("[CleanupService] Starting invalid recordings cleanup")

	for {
		// Find invalid recordings in batches
		recordings, err := s.repo.FindInvalid(ctx, batchSize)
		if err != nil {
			return totalDeleted, fmt.Errorf("failed to find invalid recordings: %w", err)
		}

		if len(recordings) == 0 {
			break // No more invalid recordings
		}

		// Delete recordings and files
		deleted, err := s.batchDelete(ctx, recordings, "invalid")
		totalDeleted += deleted
		if err != nil {
			logrus.Warnf("[CleanupService] Batch delete failed: %v", err)
		}

		// Stop if less than batch size (last batch)
		if len(recordings) < batchSize {
			break
		}
	}

	logrus.Infof("[CleanupService] Invalid recordings cleanup completed: %d deleted", totalDeleted)
	return totalDeleted, nil
}

// CleanupExpiredRecordings cleans up recordings older than retention period
func (s *CleanupService) CleanupExpiredRecordings(ctx context.Context) (int, error) {
	const batchSize = 1000
	totalDeleted := 0
	retentionDays := s.config.Recording.RetentionDays
	if retentionDays <= 0 {
		retentionDays = 90 // Default retention
	}

	logrus.Infof("[CleanupService] Starting expired recordings cleanup (retention: %d days)", retentionDays)

	for {
		// Find expired recordings in batches
		recordings, err := s.repo.FindExpired(ctx, retentionDays, batchSize)
		if err != nil {
			return totalDeleted, fmt.Errorf("failed to find expired recordings: %w", err)
		}

		if len(recordings) == 0 {
			break // No more expired recordings
		}

		// Delete recordings and files
		deleted, err := s.batchDelete(ctx, recordings, "expired")
		totalDeleted += deleted
		if err != nil {
			logrus.Warnf("[CleanupService] Batch delete failed: %v", err)
		}

		// Stop if less than batch size (last batch)
		if len(recordings) < batchSize {
			break
		}
	}

	logrus.Infof("[CleanupService] Expired recordings cleanup completed: %d deleted", totalDeleted)
	return totalDeleted, nil
}

// CleanupOrphanFiles finds and deletes files without corresponding database records
// Scans the storage directory and checks each file against the database
func (s *CleanupService) CleanupOrphanFiles(ctx context.Context) (int, error) {
	// Only LocalStorageService supports orphan file cleanup
	localStorage, ok := s.storageService.(*LocalStorageService)
	if !ok {
		logrus.Debug("[CleanupService] Skipping orphan file cleanup (not using local storage)")
		return 0, nil
	}

	logrus.Info("[CleanupService] Starting orphan file cleanup")

	basePath := localStorage.basePath
	orphanFiles := []string{}

	// Walk through all .cast files
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .cast files
		if filepath.Ext(path) != ".cast" {
			return nil
		}

		// Extract recording ID from filename
		filename := filepath.Base(path)
		recordingIDStr := filename[:len(filename)-5] // Remove .cast extension
		recordingID, err := uuid.Parse(recordingIDStr)
		if err != nil {
			logrus.Warnf("[CleanupService] Invalid recording filename: %s", path)
			return nil
		}

		// Check if recording exists in database
		_, err = s.repo.GetByID(ctx, recordingID)
		if err != nil {
			// Recording not found in database, mark as orphan
			orphanFiles = append(orphanFiles, path)
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to walk storage directory: %w", err)
	}

	// Delete orphan files in parallel
	deleted := s.parallelDeleteFiles(orphanFiles, 10) // 10 workers

	logrus.Infof("[CleanupService] Orphan file cleanup completed: %d deleted", deleted)
	return deleted, nil
}

// batchDelete deletes a batch of recordings with parallel file deletion
func (s *CleanupService) batchDelete(ctx context.Context, recordings []*models.TerminalRecording, reason string) (int, error) {
	if len(recordings) == 0 {
		return 0, nil
	}

	// Collect IDs and paths
	ids := make([]uuid.UUID, 0, len(recordings))
	paths := make([]string, 0, len(recordings))
	for _, r := range recordings {
		ids = append(ids, r.ID)
		paths = append(paths, r.StoragePath)
	}

	// Delete files first (parallel)
	deletedFiles := s.parallelDeleteFiles(paths, 10) // 10 workers

	// Delete database records
	if err := s.repo.BulkDelete(ctx, ids); err != nil {
		logrus.Warnf("[CleanupService] Failed to delete %d database records (%s): %v", len(ids), reason, err)
		return deletedFiles, err
	}

	logrus.Debugf("[CleanupService] Deleted %d %s recordings (%d files, %d DB records)",
		len(recordings), reason, deletedFiles, len(ids))

	return len(recordings), nil
}

// parallelDeleteFiles deletes files in parallel using a worker pool
func (s *CleanupService) parallelDeleteFiles(paths []string, workers int) int {
	if len(paths) == 0 {
		return 0
	}

	pathChan := make(chan string, len(paths))
	var wg sync.WaitGroup
	deletedCount := 0
	var mu sync.Mutex

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range pathChan {
				if err := s.storageService.DeleteRecording(path); err != nil {
					logrus.Warnf("[CleanupService] Failed to delete file %s: %v", path, err)
				} else {
					mu.Lock()
					deletedCount++
					mu.Unlock()
				}
			}
		}()
	}

	// Send paths to workers
	for _, path := range paths {
		pathChan <- path
	}
	close(pathChan)

	// Wait for all workers to finish
	wg.Wait()

	return deletedCount
}

// Run executes all cleanup tasks in sequence
// This is the main entry point for scheduled cleanup
func (s *CleanupService) Run(ctx context.Context) error {
	startTime := time.Now()
	logrus.Info("[CleanupService] Starting scheduled cleanup")

	// Step 1: Clean up invalid recordings (zero size/duration)
	invalidCount, err := s.CleanupInvalidRecordings(ctx)
	if err != nil {
		logrus.Errorf("[CleanupService] Invalid recordings cleanup failed: %v", err)
	}

	// Step 2: Clean up expired recordings
	expiredCount, err := s.CleanupExpiredRecordings(ctx)
	if err != nil {
		logrus.Errorf("[CleanupService] Expired recordings cleanup failed: %v", err)
	}

	// Step 3: Clean up orphan files
	orphanCount, err := s.CleanupOrphanFiles(ctx)
	if err != nil {
		logrus.Errorf("[CleanupService] Orphan files cleanup failed: %v", err)
	}

	// Calculate metrics
	duration := time.Since(startTime)
	totalDeleted := invalidCount + expiredCount + orphanCount

	logrus.Infof("[CleanupService] Cleanup completed in %v: invalid=%d, expired=%d, orphan=%d, total=%d",
		duration, invalidCount, expiredCount, orphanCount, totalDeleted)

	// TODO: Record Prometheus metrics (T053)
	// prometheus.RecordCleanupMetrics(invalidCount, expiredCount, orphanCount)

	return nil
}
