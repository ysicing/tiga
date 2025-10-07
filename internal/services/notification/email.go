package notification

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// EmailConfig represents email notifier configuration
type EmailConfig struct {
	SMTPHost     string   `json:"smtp_host"`
	SMTPPort     int      `json:"smtp_port"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	From         string   `json:"from"`
	To           []string `json:"to"`
	UseTLS       bool     `json:"use_tls"`
	InsecureSkip bool     `json:"insecure_skip"`
}

// EmailNotifier sends email notifications
type EmailNotifier struct {
	config *EmailConfig
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(config *EmailConfig) *EmailNotifier {
	return &EmailNotifier{
		config: config,
	}
}

// Send sends an email notification
func (n *EmailNotifier) Send(ctx context.Context, notification *Notification) error {
	// Determine recipients
	recipients := n.config.To
	if notification.Destination != "" {
		recipients = []string{notification.Destination}
	}

	if len(recipients) == 0 {
		return fmt.Errorf("no email recipients configured")
	}

	// Build email message
	subject := fmt.Sprintf("[%s] %s", strings.ToUpper(string(notification.Severity)), notification.Title)
	body := notification.Message

	// Add metadata if present
	if len(notification.Metadata) > 0 {
		body += "\n\n--- Metadata ---\n"
		for k, v := range notification.Metadata {
			body += fmt.Sprintf("%s: %v\n", k, v)
		}
	}

	message := []byte(fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body))

	// Setup authentication
	auth := smtp.PlainAuth("", n.config.Username, n.config.Password, n.config.SMTPHost)

	// Setup TLS config
	tlsConfig := &tls.Config{
		InsecureSkipVerify: n.config.InsecureSkip,
		ServerName:         n.config.SMTPHost,
	}

	addr := fmt.Sprintf("%s:%d", n.config.SMTPHost, n.config.SMTPPort)

	if n.config.UseTLS {
		// Connect with TLS
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, n.config.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Quit()

		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}

		if err := client.Mail(n.config.From); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}

		for _, recipient := range recipients {
			if err := client.Rcpt(recipient); err != nil {
				return fmt.Errorf("failed to add recipient %s: %w", recipient, err)
			}
		}

		wc, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to get data writer: %w", err)
		}

		if _, err := wc.Write(message); err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}

		if err := wc.Close(); err != nil {
			return fmt.Errorf("failed to close data writer: %w", err)
		}
	} else {
		// Send without TLS
		if err := smtp.SendMail(addr, auth, n.config.From, recipients, message); err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	}

	return nil
}

// Type returns the notifier type
func (n *EmailNotifier) Type() string {
	return "email"
}

// Validate validates the email configuration
func (n *EmailNotifier) Validate() error {
	if n.config.SMTPHost == "" {
		return fmt.Errorf("SMTP host is required")
	}

	if n.config.SMTPPort == 0 {
		return fmt.Errorf("SMTP port is required")
	}

	if n.config.Username == "" {
		return fmt.Errorf("SMTP username is required")
	}

	if n.config.Password == "" {
		return fmt.Errorf("SMTP password is required")
	}

	if n.config.From == "" {
		return fmt.Errorf("from address is required")
	}

	return nil
}
