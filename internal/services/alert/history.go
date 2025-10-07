package alert

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// HistoryService handles alert event history business logic
type HistoryService struct {
	alertRepo *repository.AlertRepository
}

// NewHistoryService creates a new alert history service
func NewHistoryService(alertRepo *repository.AlertRepository) *HistoryService {
	return &HistoryService{
		alertRepo: alertRepo,
	}
}

// CreateEvent creates a new alert event
func (s *HistoryService) CreateEvent(ctx context.Context, alertID, instanceID uuid.UUID, message string, details map[string]interface{}) (*models.AlertEvent, error) {
	event := &models.AlertEvent{
		AlertID:    alertID,
		InstanceID: instanceID,
		Status:     "firing",
		Message:    message,
		Details:    details,
		StartedAt:  time.Now(),
		Notified:   false,
	}

	if err := s.alertRepo.CreateEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to create alert event: %w", err)
	}

	return event, nil
}

// ResolveEvent marks an alert event as resolved
func (s *HistoryService) ResolveEvent(ctx context.Context, eventID uuid.UUID) error {
	event, err := s.alertRepo.GetEventByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	if event.IsResolved() {
		return fmt.Errorf("event is already resolved")
	}

	if err := s.alertRepo.ResolveEvent(ctx, eventID); err != nil {
		return fmt.Errorf("failed to resolve event: %w", err)
	}

	return nil
}

// MarkEventNotified marks an event as notified
func (s *HistoryService) MarkEventNotified(ctx context.Context, eventID uuid.UUID) error {
	event, err := s.alertRepo.GetEventByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	now := time.Now()
	event.Notified = true
	event.NotificationSentAt = &now

	if err := s.alertRepo.UpdateEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to mark event as notified: %w", err)
	}

	return nil
}

// GetEvent retrieves an alert event by ID
func (s *HistoryService) GetEvent(ctx context.Context, eventID uuid.UUID) (*models.AlertEvent, error) {
	event, err := s.alertRepo.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return event, nil
}

// EventFilter represents query parameters for alert events
type EventFilter struct {
	AlertID    *uuid.UUID
	InstanceID *uuid.UUID
	Status     string
	StartTime  *time.Time
	EndTime    *time.Time
	Notified   *bool
	Page       int
	PageSize   int
}

// QueryEvents retrieves alert events with filters
func (s *HistoryService) QueryEvents(ctx context.Context, filter *EventFilter) ([]*models.AlertEvent, int64, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	// Build repository filter
	repoFilter := &repository.ListEventsFilter{
		AlertID:    filter.AlertID,
		InstanceID: filter.InstanceID,
		Status:     filter.Status,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}

	if filter.StartTime != nil {
		repoFilter.StartTime = *filter.StartTime
	}
	if filter.EndTime != nil {
		repoFilter.EndTime = *filter.EndTime
	}

	return s.alertRepo.ListEvents(ctx, repoFilter)
}

// GetEventsByAlert retrieves all events for a specific alert
func (s *HistoryService) GetEventsByAlert(ctx context.Context, alertID uuid.UUID, page, pageSize int) ([]*models.AlertEvent, int64, error) {
	return s.QueryEvents(ctx, &EventFilter{
		AlertID:  &alertID,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetEventsByInstance retrieves all events for a specific instance
func (s *HistoryService) GetEventsByInstance(ctx context.Context, instanceID uuid.UUID, page, pageSize int) ([]*models.AlertEvent, int64, error) {
	return s.QueryEvents(ctx, &EventFilter{
		InstanceID: &instanceID,
		Page:       page,
		PageSize:   pageSize,
	})
}

// GetFiringEvents retrieves all currently firing events
func (s *HistoryService) GetFiringEvents(ctx context.Context, page, pageSize int) ([]*models.AlertEvent, int64, error) {
	return s.QueryEvents(ctx, &EventFilter{
		Status:   "firing",
		Page:     page,
		PageSize: pageSize,
	})
}

// GetRecentEvents retrieves recent events within a time range
func (s *HistoryService) GetRecentEvents(ctx context.Context, duration time.Duration, page, pageSize int) ([]*models.AlertEvent, int64, error) {
	startTime := time.Now().Add(-duration)
	return s.QueryEvents(ctx, &EventFilter{
		StartTime: &startTime,
		Page:      page,
		PageSize:  pageSize,
	})
}

// GetUnnotifiedEvents retrieves events that haven't been notified yet
func (s *HistoryService) GetUnnotifiedEvents(ctx context.Context, limit int) ([]*models.AlertEvent, error) {
	notified := false
	events, _, err := s.QueryEvents(ctx, &EventFilter{
		Notified: &notified,
		Page:     1,
		PageSize: limit,
	})
	return events, err
}

// EventStatistics represents alert event statistics
type EventStatistics struct {
	TotalEvents     int64                    `json:"total_events"`
	FiringEvents    int64                    `json:"firing_events"`
	ResolvedEvents  int64                    `json:"resolved_events"`
	UnnotifiedCount int64                    `json:"unnotified_count"`
	ByAlert         map[string]int64         `json:"by_alert"`
	ByInstance      map[string]int64         `json:"by_instance"`
	ByStatus        map[string]int64         `json:"by_status"`
	AverageDuration time.Duration            `json:"average_duration"`
	RecentTrend     []map[string]interface{} `json:"recent_trend"`
}

// GetStatistics retrieves alert event statistics
func (s *HistoryService) GetStatistics(ctx context.Context, startTime, endTime time.Time) (*EventStatistics, error) {
	stats := &EventStatistics{
		ByAlert:    make(map[string]int64),
		ByInstance: make(map[string]int64),
		ByStatus:   make(map[string]int64),
	}

	// Get all events in the time range
	events, total, err := s.QueryEvents(ctx, &EventFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
		Page:      1,
		PageSize:  1000, // Large enough to get stats
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	stats.TotalEvents = total

	// Calculate statistics
	var totalDuration time.Duration
	resolvedCount := 0

	for _, event := range events {
		// Count by status
		stats.ByStatus[event.Status]++

		// Count by alert
		alertKey := event.AlertID.String()
		stats.ByAlert[alertKey]++

		// Count by instance
		instanceKey := event.InstanceID.String()
		stats.ByInstance[instanceKey]++

		// Count firing and resolved
		if event.IsFiring() {
			stats.FiringEvents++
		} else if event.IsResolved() {
			stats.ResolvedEvents++
			totalDuration += event.Duration()
			resolvedCount++
		}

		// Count unnotified
		if !event.Notified {
			stats.UnnotifiedCount++
		}
	}

	// Calculate average duration
	if resolvedCount > 0 {
		stats.AverageDuration = totalDuration / time.Duration(resolvedCount)
	}

	return stats, nil
}

// GetEventTimeline retrieves event timeline (grouped by time)
func (s *HistoryService) GetEventTimeline(ctx context.Context, alertID *uuid.UUID, instanceID *uuid.UUID, startTime, endTime time.Time, interval string) ([]map[string]interface{}, error) {
	// This would ideally use database aggregation
	// For now, return a simple implementation
	events, _, err := s.QueryEvents(ctx, &EventFilter{
		AlertID:    alertID,
		InstanceID: instanceID,
		StartTime:  &startTime,
		EndTime:    &endTime,
		Page:       1,
		PageSize:   1000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	// Group events by interval
	timeline := make([]map[string]interface{}, 0)
	buckets := make(map[string]int)

	for _, event := range events {
		// Simplified bucketing by hour
		bucket := event.StartedAt.Format("2006-01-02 15:00:00")
		buckets[bucket]++
	}

	for timestamp, count := range buckets {
		timeline = append(timeline, map[string]interface{}{
			"timestamp": timestamp,
			"count":     count,
		})
	}

	return timeline, nil
}

// CleanupResolvedEvents deletes resolved events older than retention period
func (s *HistoryService) CleanupResolvedEvents(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, fmt.Errorf("retention days must be positive")
	}

	// This would need to be implemented in repository
	// For now, return error indicating not implemented
	return 0, fmt.Errorf("cleanup not yet implemented in repository")
}

// AcknowledgeEvent marks an event as acknowledged by a user
func (s *HistoryService) AcknowledgeEvent(ctx context.Context, eventID uuid.UUID, userID uuid.UUID, note string) error {
	event, err := s.alertRepo.GetEventByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	if event.IsResolved() {
		return fmt.Errorf("cannot acknowledge resolved event")
	}

	// Update event details to include acknowledgment
	details := event.Details
	if details == nil {
		details = make(map[string]interface{})
	}

	details["acknowledged"] = true
	details["acknowledged_by"] = userID.String()
	details["acknowledged_at"] = time.Now()
	details["acknowledgment_note"] = note

	event.Details = details

	if err := s.alertRepo.UpdateEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to acknowledge event: %w", err)
	}

	return nil
}
