package notification

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// DingTalkConfig represents DingTalk notifier configuration
type DingTalkConfig struct {
	WebhookURL string   `json:"webhook_url"`
	Secret     string   `json:"secret"`
	AtMobiles  []string `json:"at_mobiles"`
	AtAll      bool     `json:"at_all"`
}

// DingTalkNotifier sends DingTalk notifications
type DingTalkNotifier struct {
	config     *DingTalkConfig
	httpClient *http.Client
}

// NewDingTalkNotifier creates a new DingTalk notifier
func NewDingTalkNotifier(config *DingTalkConfig) *DingTalkNotifier {
	return &DingTalkNotifier{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send sends a DingTalk notification
func (n *DingTalkNotifier) Send(ctx context.Context, notification *Notification) error {
	// Build message
	var color string
	switch notification.Severity {
	case SeverityCritical:
		color = "🔴"
	case SeverityError:
		color = "🟠"
	case SeverityWarning:
		color = "🟡"
	default:
		color = "🟢"
	}

	title := fmt.Sprintf("%s %s", color, notification.Title)
	content := notification.Message

	// Add metadata if present
	if len(notification.Metadata) > 0 {
		content += "\n\n**详细信息：**\n"
		for k, v := range notification.Metadata {
			content += fmt.Sprintf("- %s: %v\n", k, v)
		}
	}

	// Build payload
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  fmt.Sprintf("### %s\n\n%s", title, content),
		},
		"at": map[string]interface{}{
			"atMobiles": n.config.AtMobiles,
			"isAtAll":   n.config.AtAll,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Sign request if secret is configured
	webhookURL := n.config.WebhookURL
	if n.config.Secret != "" {
		timestamp := time.Now().UnixMilli()
		sign := n.sign(timestamp, n.config.Secret)
		webhookURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", webhookURL, timestamp, sign)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DingTalk notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DingTalk returned status %d", resp.StatusCode)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	errcode, _ := result["errcode"].(float64)
	if errcode != 0 {
		errmsg, _ := result["errmsg"].(string)
		return fmt.Errorf("DingTalk error: %s (code: %.0f)", errmsg, errcode)
	}

	return nil
}

// Type returns the notifier type
func (n *DingTalkNotifier) Type() string {
	return "dingtalk"
}

// Validate validates the DingTalk configuration
func (n *DingTalkNotifier) Validate() error {
	if n.config.WebhookURL == "" {
		return fmt.Errorf("DingTalk webhook URL is required")
	}

	return nil
}

// sign generates DingTalk signature
func (n *DingTalkNotifier) sign(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return url.QueryEscape(signature)
}
