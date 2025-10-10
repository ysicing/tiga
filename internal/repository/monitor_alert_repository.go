package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
)

// MonitorAlertRuleFilter represents filtering options for monitor alert rule queries
type MonitorAlertRuleFilter struct {
	Page     int
	PageSize int
	Type     string // host/service
	Enabled  *bool
	Severity string
}

// MonitorAlertEventFilter represents filtering options for monitor alert event queries
type MonitorAlertEventFilter struct {
	Page     int
	PageSize int
	RuleID   uint
	Status   string // firing/acknowledged/resolved
	Severity string
	Start    time.Time
	End      time.Time
}

// MonitorAlertRepository defines the interface for monitor alert data access
type MonitorAlertRepository interface {
	// Alert rules
	CreateRule(ctx context.Context, rule *models.MonitorAlertRule) error
	GetRuleByID(ctx context.Context, id uuid.UUID) (*models.MonitorAlertRule, error)
	ListRules(ctx context.Context, filter MonitorAlertRuleFilter) ([]*models.MonitorAlertRule, int64, error)
	UpdateRule(ctx context.Context, rule *models.MonitorAlertRule) error
	DeleteRule(ctx context.Context, id uuid.UUID) error
	GetActiveRules(ctx context.Context, ruleType string) ([]*models.MonitorAlertRule, error)

	// Alert events
	CreateEvent(ctx context.Context, event *models.MonitorAlertEvent) error
	GetEventByID(ctx context.Context, id uuid.UUID) (*models.MonitorAlertEvent, error)
	ListEvents(ctx context.Context, filter MonitorAlertEventFilter) ([]*models.MonitorAlertEvent, int64, error)
	UpdateEvent(ctx context.Context, event *models.MonitorAlertEvent) error
	AcknowledgeEvent(ctx context.Context, eventID, userID uuid.UUID, note string) error
	ResolveEvent(ctx context.Context, eventID, userID uuid.UUID, note string) error
	GetFiringEvents(ctx context.Context, ruleID uuid.UUID) ([]*models.MonitorAlertEvent, error)
	GetEventStatistics(ctx context.Context, start, end time.Time) (map[string]interface{}, error)
}

// monitorAlertRepository implements MonitorAlertRepository
type monitorAlertRepository struct {
	db *gorm.DB
}

// NewMonitorAlertRepository creates a new monitor alert repository
func NewMonitorAlertRepository(db *gorm.DB) MonitorAlertRepository {
	return &monitorAlertRepository{db: db}
}

// CreateRule creates a new alert rule
func (r *monitorAlertRepository) CreateRule(ctx context.Context, rule *models.MonitorAlertRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

// GetRuleByID retrieves an alert rule by ID
func (r *monitorAlertRepository) GetRuleByID(ctx context.Context, id uuid.UUID) (*models.MonitorAlertRule, error) {
	var rule models.MonitorAlertRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// ListRules retrieves a list of alert rules with filtering
func (r *monitorAlertRepository) ListRules(ctx context.Context, filter MonitorAlertRuleFilter) ([]*models.MonitorAlertRule, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.MonitorAlertRule{})

	// Apply filters
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	if filter.Enabled != nil {
		query = query.Where("enabled = ?", *filter.Enabled)
	}

	if filter.Severity != "" {
		query = query.Where("severity = ?", filter.Severity)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	// Fetch results
	var rules []*models.MonitorAlertRule
	if err := query.Order("created_at DESC").Find(&rules).Error; err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// UpdateRule updates an alert rule
func (r *monitorAlertRepository) UpdateRule(ctx context.Context, rule *models.MonitorAlertRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

// DeleteRule deletes an alert rule
func (r *monitorAlertRepository) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete associated events
		if err := tx.Where("rule_id = ?", id).Delete(&models.MonitorAlertEvent{}).Error; err != nil {
			return err
		}

		// Delete the rule
		return tx.Delete(&models.MonitorAlertRule{}, id).Error
	})
}

// GetActiveRules retrieves all active (enabled) rules of a specific type
func (r *monitorAlertRepository) GetActiveRules(ctx context.Context, ruleType string) ([]*models.MonitorAlertRule, error) {
	var rules []*models.MonitorAlertRule
	query := r.db.WithContext(ctx).Where("enabled = ?", true)

	if ruleType != "" {
		query = query.Where("type = ?", ruleType)
	}

	err := query.Find(&rules).Error
	return rules, err
}

// CreateEvent creates a new alert event
func (r *monitorAlertRepository) CreateEvent(ctx context.Context, event *models.MonitorAlertEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// GetEventByID retrieves an alert event by ID
func (r *monitorAlertRepository) GetEventByID(ctx context.Context, id uuid.UUID) (*models.MonitorAlertEvent, error) {
	var event models.MonitorAlertEvent
	err := r.db.WithContext(ctx).
		Preload("Rule").
		First(&event, id).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// ListEvents retrieves a list of alert events with filtering
func (r *monitorAlertRepository) ListEvents(ctx context.Context, filter MonitorAlertEventFilter) ([]*models.MonitorAlertEvent, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.MonitorAlertEvent{})

	// Apply filters
	if filter.RuleID > 0 {
		query = query.Where("rule_id = ?", filter.RuleID)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.Severity != "" {
		query = query.Where("severity = ?", filter.Severity)
	}

	if !filter.Start.IsZero() {
		query = query.Where("triggered_at >= ?", filter.Start)
	}

	if !filter.End.IsZero() {
		query = query.Where("triggered_at <= ?", filter.End)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	// Fetch results
	var events []*models.MonitorAlertEvent
	if err := query.Preload("Rule").Order("triggered_at DESC").Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

// UpdateEvent updates an alert event
func (r *monitorAlertRepository) UpdateEvent(ctx context.Context, event *models.MonitorAlertEvent) error {
	return r.db.WithContext(ctx).Save(event).Error
}

// AcknowledgeEvent marks an event as acknowledged
func (r *monitorAlertRepository) AcknowledgeEvent(ctx context.Context, eventID uuid.UUID, userID uuid.UUID, note string) error {
	var event models.MonitorAlertEvent
	if err := r.db.WithContext(ctx).First(&event, eventID).Error; err != nil {
		return err
	}

	event.Acknowledge(userID, note)
	return r.db.WithContext(ctx).Save(&event).Error
}

// ResolveEvent marks an event as resolved
func (r *monitorAlertRepository) ResolveEvent(ctx context.Context, eventID uuid.UUID, userID uuid.UUID, note string) error {
	var event models.MonitorAlertEvent
	if err := r.db.WithContext(ctx).First(&event, eventID).Error; err != nil {
		return err
	}

	event.Resolve(userID, note)
	return r.db.WithContext(ctx).Save(&event).Error
}

// GetFiringEvents retrieves all firing events for a rule
func (r *monitorAlertRepository) GetFiringEvents(ctx context.Context, ruleID uuid.UUID) ([]*models.MonitorAlertEvent, error) {
	var events []*models.MonitorAlertEvent
	err := r.db.WithContext(ctx).
		Where("rule_id = ? AND status = ?", ruleID, models.AlertStatusFiring).
		Order("triggered_at DESC").
		Find(&events).Error
	return events, err
}

// GetEventStatistics calculates event statistics for a time period
func (r *monitorAlertRepository) GetEventStatistics(ctx context.Context, start, end time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total events
	var totalEvents int64
	if err := r.db.WithContext(ctx).
		Model(&models.MonitorAlertEvent{}).
		Where("triggered_at >= ? AND triggered_at <= ?", start, end).
		Count(&totalEvents).Error; err != nil {
		return nil, err
	}
	stats["total_events"] = totalEvents

	// Count by status
	var firingCount, acknowledgedCount, resolvedCount int64
	r.db.WithContext(ctx).Model(&models.MonitorAlertEvent{}).
		Where("triggered_at >= ? AND triggered_at <= ? AND status = ?", start, end, models.AlertStatusFiring).
		Count(&firingCount)
	r.db.WithContext(ctx).Model(&models.MonitorAlertEvent{}).
		Where("triggered_at >= ? AND triggered_at <= ? AND status = ?", start, end, models.AlertStatusAcknowledged).
		Count(&acknowledgedCount)
	r.db.WithContext(ctx).Model(&models.MonitorAlertEvent{}).
		Where("triggered_at >= ? AND triggered_at <= ? AND status = ?", start, end, models.AlertStatusResolved).
		Count(&resolvedCount)

	stats["firing_count"] = firingCount
	stats["acknowledged_count"] = acknowledgedCount
	stats["resolved_count"] = resolvedCount

	// Count by severity
	bySeverity := make(map[string]int64)
	var severityStats []struct {
		Severity string
		Count    int64
	}
	r.db.WithContext(ctx).
		Model(&models.MonitorAlertEvent{}).
		Select("severity, COUNT(*) as count").
		Where("triggered_at >= ? AND triggered_at <= ?", start, end).
		Group("severity").
		Scan(&severityStats)

	for _, s := range severityStats {
		bySeverity[s.Severity] = s.Count
	}
	stats["by_severity"] = bySeverity

	return stats, nil
}
