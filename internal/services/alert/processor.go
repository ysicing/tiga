package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/internal/services/managers"
	"github.com/ysicing/tiga/internal/services/notification"
)

// AlertProcessor processes and evaluates alert rules
type AlertProcessor struct {
	alertRepo       *repository.AlertRepository
	metricsRepo     *repository.MetricsRepository
	notificationSvc *notification.NotificationService
	coordinator     *managers.ManagerCoordinator
}

// NewAlertProcessor creates a new alert processor
func NewAlertProcessor(
	alertRepo *repository.AlertRepository,
	metricsRepo *repository.MetricsRepository,
	notificationSvc *notification.NotificationService,
	coordinator *managers.ManagerCoordinator,
) *AlertProcessor {
	return &AlertProcessor{
		alertRepo:       alertRepo,
		metricsRepo:     metricsRepo,
		notificationSvc: notificationSvc,
		coordinator:     coordinator,
	}
}

// ProcessAlerts processes all active alert rules
func (p *AlertProcessor) ProcessAlerts(ctx context.Context) error {
	// Get all enabled alert rules
	rules, err := p.alertRepo.ListEnabledRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to list alert rules: %w", err)
	}

	logrus.Debugf("Processing %d alert rules", len(rules))

	for _, rule := range rules {
		if err := p.processRule(ctx, rule); err != nil {
			logrus.Errorf("Failed to process alert rule %s: %v", rule.ID, err)
		}
	}

	return nil
}

// processRule processes a single alert rule
func (p *AlertProcessor) processRule(ctx context.Context, rule *models.Alert) error {
	switch rule.RuleType {
	case "threshold":
		return p.processThresholdRule(ctx, rule)
	case "health":
		return p.processHealthRule(ctx, rule)
	case "availability":
		return p.processAvailabilityRule(ctx, rule)
	default:
		return fmt.Errorf("unsupported rule type: %s", rule.RuleType)
	}
}

// processThresholdRule processes a threshold-based alert rule
func (p *AlertProcessor) processThresholdRule(ctx context.Context, rule *models.Alert) error {
	// Parse rule config
	var config struct {
		MetricName string  `json:"metric_name"`
		Operator   string  `json:"operator"` // >, <, >=, <=, ==
		Threshold  float64 `json:"threshold"`
		Duration   int     `json:"duration"` // seconds
	}

	// Convert JSONB to JSON bytes
	configJSON, err := json.Marshal(rule.RuleConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal rule config: %w", err)
	}

	if err := json.Unmarshal(configJSON, &config); err != nil {
		return fmt.Errorf("failed to parse rule config: %w", err)
	}

	// Get instance ID if specified
	var instanceID *uuid.UUID
	if rule.InstanceID != nil {
		instanceID = rule.InstanceID
	}

	// Get recent metrics
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(config.Duration) * time.Second)

	filter := &repository.MetricsQueryFilter{
		InstanceID: instanceID,
		MetricName: config.MetricName,
		StartTime:  startTime,
		EndTime:    endTime,
	}

	metrics, err := p.metricsRepo.QueryMetrics(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to query metrics: %w", err)
	}

	if len(metrics) == 0 {
		return nil // No data to evaluate
	}

	// Calculate average value
	var sum float64
	for _, metric := range metrics {
		sum += metric.Value
	}
	avgValue := sum / float64(len(metrics))

	// Evaluate threshold
	triggered := false
	switch config.Operator {
	case ">":
		triggered = avgValue > config.Threshold
	case "<":
		triggered = avgValue < config.Threshold
	case ">=":
		triggered = avgValue >= config.Threshold
	case "<=":
		triggered = avgValue <= config.Threshold
	case "==":
		triggered = avgValue == config.Threshold
	}

	if triggered {
		return p.triggerAlert(ctx, rule, map[string]interface{}{
			"metric_name": config.MetricName,
			"value":       avgValue,
			"threshold":   config.Threshold,
			"operator":    config.Operator,
		})
	}

	return nil
}

// processHealthRule processes a health check-based alert rule
func (p *AlertProcessor) processHealthRule(ctx context.Context, rule *models.Alert) error {
	if rule.InstanceID == nil {
		return fmt.Errorf("instance ID is required for health rule")
	}

	// Perform health check
	status, err := p.coordinator.HealthCheck(ctx, *rule.InstanceID)
	if err != nil {
		return p.triggerAlert(ctx, rule, map[string]interface{}{
			"error": err.Error(),
		})
	}

	if !status.Healthy {
		return p.triggerAlert(ctx, rule, map[string]interface{}{
			"health_status": status.Message,
			"details":       status.Details,
		})
	}

	return nil
}

// processAvailabilityRule processes an availability-based alert rule
func (p *AlertProcessor) processAvailabilityRule(ctx context.Context, rule *models.Alert) error {
	if rule.InstanceID == nil {
		return fmt.Errorf("instance ID is required for availability rule")
	}

	// Try to connect to the instance
	manager, err := p.coordinator.GetManager(ctx, *rule.InstanceID)
	if err != nil {
		return p.triggerAlert(ctx, rule, map[string]interface{}{
			"error": "Failed to get manager: " + err.Error(),
		})
	}

	// Perform health check
	status, err := manager.HealthCheck(ctx)
	if err != nil || !status.Healthy {
		return p.triggerAlert(ctx, rule, map[string]interface{}{
			"error":  err,
			"status": status,
		})
	}

	return nil
}

// triggerAlert triggers an alert and sends notifications
func (p *AlertProcessor) triggerAlert(ctx context.Context, rule *models.Alert, metadata map[string]interface{}) error {
	logrus.Warnf("Alert triggered: %s (severity: %s)", rule.Name, rule.Severity)

	// Get instance ID
	var instanceID uuid.UUID
	if rule.InstanceID != nil {
		instanceID = *rule.InstanceID
	} else {
		// If no instance ID, use nil UUID
		instanceID = uuid.Nil
	}

	// Create alert event
	metadataJSON, _ := json.Marshal(metadata)
	var details models.JSONB
	_ = json.Unmarshal(metadataJSON, &details)

	event := &models.AlertEvent{
		AlertID:    rule.ID,
		InstanceID: instanceID,
		Message:    rule.Description,
		Details:    details,
		Status:     "firing",
		StartedAt:  time.Now(),
	}

	if err := p.alertRepo.CreateEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to create alert event: %w", err)
	}

	// Send notifications
	if len(rule.NotificationChannels) > 0 {
		notif := &notification.Notification{
			Title:    fmt.Sprintf("Alert: %s", rule.Name),
			Message:  rule.Description,
			Severity: notification.Severity(rule.Severity),
			Metadata: metadata,
		}

		if err := p.notificationSvc.Send(ctx, rule.NotificationChannels, notif); err != nil {
			logrus.Errorf("Failed to send notification for alert %s: %v", rule.ID, err)
		}
	}

	return nil
}

// ResolveAlert resolves an alert event
func (p *AlertProcessor) ResolveAlert(ctx context.Context, eventID uuid.UUID, userID *uuid.UUID, note string) error {
	return p.alertRepo.ResolveEvent(ctx, eventID)
}

// AcknowledgeAlert acknowledges an alert event
func (p *AlertProcessor) AcknowledgeAlert(ctx context.Context, eventID uuid.UUID, userID uuid.UUID, note string) error {
	return p.alertRepo.AcknowledgeEvent(ctx, eventID, userID, note)
}
