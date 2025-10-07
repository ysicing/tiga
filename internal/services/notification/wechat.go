package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WeChatWorkConfig represents WeChat Work notifier configuration
type WeChatWorkConfig struct {
	WebhookURL          string   `json:"webhook_url"`
	MentionedList       []string `json:"mentioned_list"`
	MentionedMobileList []string `json:"mentioned_mobile_list"`
}

// WeChatWorkNotifier sends WeChat Work notifications
type WeChatWorkNotifier struct {
	config     *WeChatWorkConfig
	httpClient *http.Client
}

// NewWeChatWorkNotifier creates a new WeChat Work notifier
func NewWeChatWorkNotifier(config *WeChatWorkConfig) *WeChatWorkNotifier {
	return &WeChatWorkNotifier{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send sends a WeChat Work notification
func (n *WeChatWorkNotifier) Send(ctx context.Context, notification *Notification) error {
	// Build message
	var emoji string
	switch notification.Severity {
	case SeverityCritical:
		emoji = "🔴"
	case SeverityError:
		emoji = "🟠"
	case SeverityWarning:
		emoji = "🟡"
	default:
		emoji = "🟢"
	}

	content := fmt.Sprintf("%s **%s**\n\n%s", emoji, notification.Title, notification.Message)

	// Add metadata if present
	if len(notification.Metadata) > 0 {
		content += "\n\n详细信息：\n"
		for k, v := range notification.Metadata {
			content += fmt.Sprintf("> %s: %v\n", k, v)
		}
	}

	// Build payload
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"content":               content,
			"mentioned_list":        n.config.MentionedList,
			"mentioned_mobile_list": n.config.MentionedMobileList,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", n.config.WebhookURL, bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send WeChat Work notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("WeChat Work returned status %d", resp.StatusCode)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	errcode, _ := result["errcode"].(float64)
	if errcode != 0 {
		errmsg, _ := result["errmsg"].(string)
		return fmt.Errorf("WeChat Work error: %s (code: %.0f)", errmsg, errcode)
	}

	return nil
}

// Type returns the notifier type
func (n *WeChatWorkNotifier) Type() string {
	return "wechat_work"
}

// Validate validates the WeChat Work configuration
func (n *WeChatWorkNotifier) Validate() error {
	if n.config.WebhookURL == "" {
		return fmt.Errorf("WeChat Work webhook URL is required")
	}

	return nil
}
