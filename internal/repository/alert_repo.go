package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// AlertRepository handles alert data operations
type AlertRepository struct {
	db *gorm.DB
}

// NewAlertRepository creates a new AlertRepository
func NewAlertRepository(db *gorm.DB) *AlertRepository {
	return &AlertRepository{db: db}
}

// --- Alert Rule Methods ---

// CreateRule creates a new alert rule
func (r *AlertRepository) CreateRule(ctx context.Context, rule *models.Alert) error {
	if err := r.db.WithContext(ctx).Create(rule).Error; err != nil {
		return fmt.Errorf("failed to create alert rule: %w", err)
	}
	return nil
}

// GetRuleByID retrieves an alert rule by ID
func (r *AlertRepository) GetRuleByID(ctx context.Context, id uuid.UUID) (*models.Alert, error) {
	var rule models.Alert
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&rule).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("alert rule not found")
		}
		return nil, fmt.Errorf("failed to get alert rule: %w", err)
	}

	return &rule, nil
}

// GetRuleByName retrieves an alert rule by name
func (r *AlertRepository) GetRuleByName(ctx context.Context, name string) (*models.Alert, error) {
	var rule models.Alert
	err := r.db.WithContext(ctx).
		Where("name = ?", name).
		First(&rule).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("alert rule not found")
		}
		return nil, fmt.Errorf("failed to get alert rule: %w", err)
	}

	return &rule, nil
}

// UpdateRule updates an alert rule
func (r *AlertRepository) UpdateRule(ctx context.Context, rule *models.Alert) error {
	if err := r.db.WithContext(ctx).Save(rule).Error; err != nil {
		return fmt.Errorf("failed to update alert rule: %w", err)
	}
	return nil
}

// DeleteRule soft deletes an alert rule
func (r *AlertRepository) DeleteRule(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Alert{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete alert rule: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("alert rule not found")
	}

	return nil
}

// ListRulesFilter represents alert rule list filters
type ListRulesFilter struct {
	InstanceID *uuid.UUID // Filter by instance ID
	Enabled    *bool      // Filter by enabled status
	Severity   string     // Filter by severity
	Search     string     // Search in name, description
	Page       int        // Page number (1-based)
	PageSize   int        // Page size
}

// ListRules retrieves a paginated list of alert rules with filters
func (r *AlertRepository) ListRules(ctx context.Context, filter *ListRulesFilter) ([]*models.Alert, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Alert{}).Preload("Instance")

	// Apply filters
	if filter.InstanceID != nil {
		query = query.Where("instance_id = ?", *filter.InstanceID)
	}

	if filter.Enabled != nil {
		query = query.Where("enabled = ?", *filter.Enabled)
	}

	if filter.Severity != "" {
		query = query.Where("severity = ?", filter.Severity)
	}

	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", search, search)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count alert rules: %w", err)
	}

	// Pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Fetch results
	var rules []*models.Alert
	if err := query.Order("created_at DESC").Find(&rules).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list alert rules: %w", err)
	}

	return rules, total, nil
}

// ListEnabledRules retrieves all enabled alert rules
func (r *AlertRepository) ListEnabledRules(ctx context.Context) ([]*models.Alert, error) {
	var rules []*models.Alert
	err := r.db.WithContext(ctx).
		Preload("Instance").
		Where("enabled = ?", true).
		Order("name ASC").
		Find(&rules).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list enabled alert rules: %w", err)
	}

	return rules, nil
}

// ListRulesByInstance retrieves all alert rules for a specific instance
func (r *AlertRepository) ListRulesByInstance(ctx context.Context, instanceID uuid.UUID) ([]*models.Alert, error) {
	var rules []*models.Alert
	err := r.db.WithContext(ctx).
		Preload("Instance").
		Where("instance_id = ?", instanceID).
		Order("name ASC").
		Find(&rules).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list rules by instance: %w", err)
	}

	return rules, nil
}

// ToggleRule enables or disables an alert rule
func (r *AlertRepository) ToggleRule(ctx context.Context, id uuid.UUID, enabled bool) error {
	result := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Where("id = ?", id).
		Update("enabled", enabled)

	if result.Error != nil {
		return fmt.Errorf("failed to toggle alert rule: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("alert rule not found")
	}

	return nil
}

// --- Alert Event Methods ---

// CreateEvent creates a new alert event
func (r *AlertRepository) CreateEvent(ctx context.Context, event *models.AlertEvent) error {
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("failed to create alert event: %w", err)
	}
	return nil
}

// GetEventByID retrieves an alert event by ID
func (r *AlertRepository) GetEventByID(ctx context.Context, id uuid.UUID) (*models.AlertEvent, error) {
	var event models.AlertEvent
	err := r.db.WithContext(ctx).
		Preload("Alert").
		Where("id = ?", id).
		First(&event).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("alert event not found")
		}
		return nil, fmt.Errorf("failed to get alert event: %w", err)
	}

	return &event, nil
}

// UpdateEvent updates an alert event
func (r *AlertRepository) UpdateEvent(ctx context.Context, event *models.AlertEvent) error {
	if err := r.db.WithContext(ctx).Save(event).Error; err != nil {
		return fmt.Errorf("failed to update alert event: %w", err)
	}
	return nil
}

// DeleteEvent deletes an alert event
func (r *AlertRepository) DeleteEvent(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.AlertEvent{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete alert event: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("alert event not found")
	}

	return nil
}

// ListEventsFilter represents alert event list filters
type ListEventsFilter struct {
	AlertID    *uuid.UUID // Filter by alert ID
	InstanceID *uuid.UUID // Filter by instance ID
	Status     string     // Filter by status
	Severity   string     // Filter by severity (not directly on event, will need join)
	StartTime  time.Time  // Filter by start time
	EndTime    time.Time  // Filter by end time
	Page       int        // Page number (1-based)
	PageSize   int        // Page size
}

// ListEvents retrieves a paginated list of alert events with filters
func (r *AlertRepository) ListEvents(ctx context.Context, filter *ListEventsFilter) ([]*models.AlertEvent, int64, error) {
	query := r.db.WithContext(ctx).Preload("Alert")

	// Apply filters
	if filter.AlertID != nil {
		query = query.Where("alert_id = ?", *filter.AlertID)
	}

	if filter.InstanceID != nil {
		query = query.Where("instance_id = ?", *filter.InstanceID)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.Severity != "" {
		// Severity is on the alert, not the event, so we need a join
		query = query.Joins("JOIN alerts ON alerts.id = alert_events.alert_id").
			Where("alerts.severity = ?", filter.Severity)
	}

	if !filter.StartTime.IsZero() {
		query = query.Where("started_at >= ?", filter.StartTime)
	}

	if !filter.EndTime.IsZero() {
		query = query.Where("started_at <= ?", filter.EndTime)
	}

	// Count total
	var total int64
	if err := query.Model(&models.AlertEvent{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count alert events: %w", err)
	}

	// Pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// Fetch results
	var events []*models.AlertEvent
	if err := query.Order("started_at DESC").Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list alert events: %w", err)
	}

	return events, total, nil
}

// ListActiveEvents retrieves all active (firing) alert events
func (r *AlertRepository) ListActiveEvents(ctx context.Context) ([]*models.AlertEvent, error) {
	var events []*models.AlertEvent
	err := r.db.WithContext(ctx).
		Preload("Alert").
		Where("status = ?", "firing").
		Order("started_at DESC").
		Find(&events).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list active events: %w", err)
	}

	return events, nil
}

// ListEventsByInstance retrieves all alert events for a specific instance
func (r *AlertRepository) ListEventsByInstance(ctx context.Context, instanceID uuid.UUID) ([]*models.AlertEvent, error) {
	var events []*models.AlertEvent
	err := r.db.WithContext(ctx).
		Preload("Alert").
		Where("instance_id = ?", instanceID).
		Order("started_at DESC").
		Find(&events).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list events by instance: %w", err)
	}

	return events, nil
}

// AcknowledgeEvent acknowledges an alert event
func (r *AlertRepository) AcknowledgeEvent(ctx context.Context, id uuid.UUID, acknowledgedBy uuid.UUID, note string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":               "acknowledged",
		"acknowledged_at":      &now,
		"acknowledged_by":      &acknowledgedBy,
		"acknowledgement_note": note,
	}

	result := r.db.WithContext(ctx).
		Model(&models.AlertEvent{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to acknowledge alert event: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("alert event not found")
	}

	return nil
}

// ResolveEvent resolves an alert event
func (r *AlertRepository) ResolveEvent(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":      "resolved",
		"resolved_at": &now,
	}

	result := r.db.WithContext(ctx).
		Model(&models.AlertEvent{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to resolve alert event: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("alert event not found")
	}

	return nil
}

// CountEventsByStatus counts alert events by status
func (r *AlertRepository) CountEventsByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.AlertEvent{}).
		Where("status = ?", status).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count events by status: %w", err)
	}

	return count, nil
}

// CountEventsBySeverity counts alert events by severity
func (r *AlertRepository) CountEventsBySeverity(ctx context.Context, severity string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.AlertEvent{}).
		Where("severity = ?", severity).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count events by severity: %w", err)
	}

	return count, nil
}

// DeleteOldEvents deletes alert events older than a specified time
func (r *AlertRepository) DeleteOldEvents(ctx context.Context, olderThan time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("started_at < ? AND status != ?", olderThan, "firing").
		Delete(&models.AlertEvent{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old events: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// AlertStatistics represents alert statistics
type AlertStatistics struct {
	TotalRules         int64            `json:"total_rules"`
	EnabledRules       int64            `json:"enabled_rules"`
	TotalEvents        int64            `json:"total_events"`
	ActiveEvents       int64            `json:"active_events"`
	AcknowledgedEvents int64            `json:"acknowledged_events"`
	ResolvedEvents     int64            `json:"resolved_events"`
	BySeverity         map[string]int64 `json:"by_severity"`
}

// GetStatistics retrieves alert statistics
func (r *AlertRepository) GetStatistics(ctx context.Context) (*AlertStatistics, error) {
	stats := &AlertStatistics{
		BySeverity: make(map[string]int64),
	}

	// Total rules
	if err := r.db.WithContext(ctx).Model(&models.Alert{}).Count(&stats.TotalRules).Error; err != nil {
		return nil, fmt.Errorf("failed to count total rules: %w", err)
	}

	// Enabled rules
	if err := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Where("enabled = ?", true).
		Count(&stats.EnabledRules).Error; err != nil {
		return nil, fmt.Errorf("failed to count enabled rules: %w", err)
	}

	// Total events
	if err := r.db.WithContext(ctx).Model(&models.AlertEvent{}).Count(&stats.TotalEvents).Error; err != nil {
		return nil, fmt.Errorf("failed to count total events: %w", err)
	}

	// Active events
	stats.ActiveEvents, _ = r.CountEventsByStatus(ctx, "firing")
	stats.AcknowledgedEvents, _ = r.CountEventsByStatus(ctx, "acknowledged")
	stats.ResolvedEvents, _ = r.CountEventsByStatus(ctx, "resolved")

	// Group by severity
	type SeverityCount struct {
		Severity string
		Count    int64
	}
	var severityCounts []SeverityCount
	if err := r.db.WithContext(ctx).
		Model(&models.AlertEvent{}).
		Select("severity, COUNT(*) as count").
		Group("severity").
		Scan(&severityCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to count by severity: %w", err)
	}
	for _, sc := range severityCounts {
		stats.BySeverity[sc.Severity] = sc.Count
	}

	return stats, nil
}
