package notification

import (
	"context"
	"fmt"
)

// Severity represents notification severity level
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Notification represents a notification message
type Notification struct {
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Severity    Severity               `json:"severity"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Destination string                 `json:"destination,omitempty"`
}

// Notifier is the interface for sending notifications
type Notifier interface {
	// Send sends a notification
	Send(ctx context.Context, notification *Notification) error

	// Type returns the notifier type
	Type() string

	// Validate validates the notifier configuration
	Validate() error
}

// NotificationService manages multiple notifiers
type NotificationService struct {
	notifiers map[string]Notifier
}

// NewNotificationService creates a new notification service
func NewNotificationService() *NotificationService {
	return &NotificationService{
		notifiers: make(map[string]Notifier),
	}
}

// RegisterNotifier registers a notifier
func (s *NotificationService) RegisterNotifier(name string, notifier Notifier) error {
	if err := notifier.Validate(); err != nil {
		return fmt.Errorf("invalid notifier configuration: %w", err)
	}

	s.notifiers[name] = notifier
	return nil
}

// Send sends a notification through specified channels
func (s *NotificationService) Send(ctx context.Context, channels []string, notification *Notification) error {
	if len(channels) == 0 {
		return fmt.Errorf("no notification channels specified")
	}

	var lastErr error
	successCount := 0

	for _, channel := range channels {
		notifier, ok := s.notifiers[channel]
		if !ok {
			lastErr = fmt.Errorf("notifier %s not found", channel)
			continue
		}

		if err := notifier.Send(ctx, notification); err != nil {
			lastErr = err
			continue
		}

		successCount++
	}

	if successCount == 0 {
		return fmt.Errorf("failed to send notification through any channel: %w", lastErr)
	}

	return nil
}

// GetNotifier returns a notifier by name
func (s *NotificationService) GetNotifier(name string) (Notifier, bool) {
	notifier, ok := s.notifiers[name]
	return notifier, ok
}

// ListNotifiers returns all registered notifier names
func (s *NotificationService) ListNotifiers() []string {
	names := make([]string, 0, len(s.notifiers))
	for name := range s.notifiers {
		names = append(names, name)
	}
	return names
}
