package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookConfig represents webhook notifier configuration
type WebhookConfig struct {
	URL        string            `json:"url"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers"`
	Timeout    int               `json:"timeout"` // seconds
	RetryCount int               `json:"retry_count"`
	RetryDelay int               `json:"retry_delay"` // seconds
}

// WebhookNotifier sends webhook notifications
type WebhookNotifier struct {
	config     *WebhookConfig
	httpClient *http.Client
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier(config *WebhookConfig) *WebhookNotifier {
	timeout := 30
	if config.Timeout > 0 {
		timeout = config.Timeout
	}

	return &WebhookNotifier{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// Send sends a webhook notification
func (n *WebhookNotifier) Send(ctx context.Context, notification *Notification) error {
	// Build payload
	payload := map[string]interface{}{
		"title":     notification.Title,
		"message":   notification.Message,
		"severity":  notification.Severity,
		"metadata":  notification.Metadata,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	method := n.config.Method
	if method == "" {
		method = "POST"
	}

	retryCount := n.config.RetryCount
	if retryCount == 0 {
		retryCount = 3
	}

	retryDelay := n.config.RetryDelay
	if retryDelay == 0 {
		retryDelay = 1
	}

	var lastErr error
	for attempt := 0; attempt <= retryCount; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(retryDelay) * time.Second)
		}

		req, err := http.NewRequestWithContext(ctx, method, n.config.URL, bytes.NewReader(jsonPayload))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		// Add custom headers
		for key, value := range n.config.Headers {
			req.Header.Set(key, value)
		}

		resp, err := n.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send webhook: %w", err)
			continue
		}

		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return fmt.Errorf("webhook failed after %d retries: %w", retryCount, lastErr)
}

// Type returns the notifier type
func (n *WebhookNotifier) Type() string {
	return "webhook"
}

// Validate validates the webhook configuration
func (n *WebhookNotifier) Validate() error {
	if n.config.URL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	return nil
}
