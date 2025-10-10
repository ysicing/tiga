package host

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
)

// ExpiryScheduler monitors host expiry dates and generates alerts
type ExpiryScheduler struct {
	hostRepo    repository.HostRepository
	alertRepo   repository.MonitorAlertRepository
	db          *gorm.DB
	checkTicker *time.Ticker
	stopCh      chan struct{}
}

// NewExpiryScheduler creates a new expiry scheduler
func NewExpiryScheduler(
	hostRepo repository.HostRepository,
	alertRepo repository.MonitorAlertRepository,
	db *gorm.DB,
) *ExpiryScheduler {
	return &ExpiryScheduler{
		hostRepo:  hostRepo,
		alertRepo: alertRepo,
		db:        db,
		stopCh:    make(chan struct{}),
	}
}

// Start begins the expiry checking routine
func (s *ExpiryScheduler) Start() {
	// Check every day at midnight
	s.checkTicker = time.NewTicker(24 * time.Hour)

	// Run immediately on start
	go s.checkExpiry()

	// Schedule regular checks
	go func() {
		for {
			select {
			case <-s.checkTicker.C:
				s.checkExpiry()
			case <-s.stopCh:
				return
			}
		}
	}()

	logrus.Info("Expiry scheduler started")
}

// Stop stops the scheduler
func (s *ExpiryScheduler) Stop() {
	if s.checkTicker != nil {
		s.checkTicker.Stop()
	}
	close(s.stopCh)
	logrus.Info("Expiry scheduler stopped")
}

// checkExpiry checks all hosts for expiry warnings
func (s *ExpiryScheduler) checkExpiry() {
	ctx := context.Background()
	logrus.Debug("Checking host expiry dates...")

	// Get all hosts with expiry dates
	var hosts []models.HostNode
	if err := s.db.WithContext(ctx).
		Where("expiry_date IS NOT NULL").
		Find(&hosts).Error; err != nil {
		logrus.Errorf("Failed to query hosts for expiry check: %v", err)
		return
	}

	now := time.Now()
	warningThresholds := []struct {
		days    int
		message string
	}{
		{days: 30, message: "Host will expire in 30 days"},
		{days: 7, message: "Host will expire in 7 days"},
		{days: 3, message: "Host will expire in 3 days"},
		{days: 1, message: "Host will expire in 1 day"},
		{days: 0, message: "Host has expired"},
	}

	for _, host := range hosts {
		if host.ExpiryDate == nil {
			continue
		}

		daysUntilExpiry := int(time.Until(*host.ExpiryDate).Hours() / 24)

		// Generate warnings at specific thresholds
		for _, threshold := range warningThresholds {
			if daysUntilExpiry == threshold.days {
				s.generateExpiryAlert(ctx, &host, threshold.message)
				break
			}
		}

		// Also check if already expired
		if host.ExpiryDate.Before(now) && daysUntilExpiry < 0 {
			// Only alert once for expired hosts
			if daysUntilExpiry == 0 {
				s.generateExpiryAlert(ctx, &host, "Host subscription has expired")
			}
		}
	}

	logrus.Debugf("Expiry check completed for %d hosts", len(hosts))
}

// generateExpiryAlert creates an alert event for expiring host
func (s *ExpiryScheduler) generateExpiryAlert(ctx context.Context, host *models.HostNode, message string) {
	// Get or create a system expiry alert rule for this host
	rule := s.getOrCreateExpiryRule(ctx, host)
	if rule == nil {
		logrus.Error("Failed to get or create expiry rule")
		return
	}

	// Check if we already have an active alert for this host and message
	var existingEvent models.MonitorAlertEvent
	err := s.db.WithContext(ctx).
		Where("rule_id = ? AND message = ? AND status = ?",
			rule.ID, message, models.AlertStatusFiring).
		First(&existingEvent).Error

	if err == nil {
		// Alert already exists
		return
	}

	// Create new alert event
	contextData := fmt.Sprintf(`{"host_name":"%s","expiry_date":"%s"}`,
		host.Name, host.ExpiryDate.Format("2006-01-02"))

	event := &models.MonitorAlertEvent{
		RuleID:      rule.ID,
		Severity:    s.getSeverityByDaysLeft(host.ExpiryDate),
		Message:     message,
		Status:      models.AlertStatusFiring,
		Context:     contextData,
		TriggeredAt: time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(event).Error; err != nil {
		logrus.Errorf("Failed to create expiry alert for host %s: %v", host.Name, err)
		return
	}

	logrus.Infof("Created expiry alert for host %s: %s", host.Name, message)
}

// getOrCreateExpiryRule gets or creates an expiry monitoring rule for a host
func (s *ExpiryScheduler) getOrCreateExpiryRule(ctx context.Context, host *models.HostNode) *models.MonitorAlertRule {
	// Try to find existing expiry rule for this host
	var rule models.MonitorAlertRule
	err := s.db.WithContext(ctx).
		Where("target_id = ? AND type = ? AND condition = ?",
			host.ID, models.AlertTypeHost, "expiry").
		First(&rule).Error

	if err == nil {
		return &rule
	}

	// Create new expiry rule
	rule = models.MonitorAlertRule{
		Name:           fmt.Sprintf("Expiry Alert - %s", host.Name),
		Type:           models.AlertTypeHost,
		TargetID:       host.ID,
		Severity:       models.AlertSeverityWarning,
		Condition:      "expiry", // Special condition for expiry checking
		Duration:       0,
		Enabled:        true,
		NotifyChannels: `["email"]`,
		NotifyConfig:   `{}`,
	}

	if err := s.db.WithContext(ctx).Create(&rule).Error; err != nil {
		logrus.Errorf("Failed to create expiry rule for host %s: %v", host.Name, err)
		return nil
	}

	return &rule
}

// getSeverityByDaysLeft determines alert severity based on days until expiry
func (s *ExpiryScheduler) getSeverityByDaysLeft(expiryDate *time.Time) models.AlertSeverity {
	if expiryDate == nil {
		return models.AlertSeverityInfo
	}

	daysLeft := int(time.Until(*expiryDate).Hours() / 24)

	switch {
	case daysLeft < 0:
		return models.AlertSeverityCritical // Already expired
	case daysLeft <= 1:
		return models.AlertSeverityCritical // Expires today or tomorrow
	case daysLeft <= 3:
		return models.AlertSeverityCritical // Expires within 3 days
	case daysLeft <= 7:
		return models.AlertSeverityWarning // Expires within a week
	case daysLeft <= 30:
		return models.AlertSeverityInfo // Expires within a month
	default:
		return models.AlertSeverityInfo
	}
}

// CheckHostExpiry performs an immediate check for a specific host
func (s *ExpiryScheduler) CheckHostExpiry(ctx context.Context, hostID uint) error {
	var host models.HostNode
	if err := s.db.WithContext(ctx).First(&host, hostID).Error; err != nil {
		return fmt.Errorf("host not found: %w", err)
	}

	if host.ExpiryDate == nil {
		return nil // No expiry date set
	}

	now := time.Now()
	if host.ExpiryDate.Before(now) {
		s.generateExpiryAlert(ctx, &host, "Host subscription has expired")
	} else {
		daysUntilExpiry := int(time.Until(*host.ExpiryDate).Hours() / 24)
		if daysUntilExpiry <= 30 {
			message := fmt.Sprintf("Host will expire in %d days", daysUntilExpiry)
			s.generateExpiryAlert(ctx, &host, message)
		}
	}

	return nil
}
