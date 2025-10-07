package common

import (
	"os"
)

const (
	JWTExpirationSeconds = 24 * 60 * 60 // 24 hours

	NodeTerminalPodName = "tiga-node-terminal-agent"

	KubectlAnnotation = "kubectl.kubernetes.io/last-applied-configuration"
)

// Legacy configuration variables
// TODO: Migrate to internal/config system for unified configuration management
var (
	Host = ""

	NodeTerminalImage = "busybox:latest"
	WebhookUsername   = os.Getenv("WEBHOOK_USERNAME")
	WebhookPassword   = os.Getenv("WEBHOOK_PASSWORD")
	WebhookEnabled    = WebhookUsername != "" && WebhookPassword != ""

	tigaEncryptKey = "tiga-default-encryption-key-change-in-production"

	AnonymousUserEnabled = false

	CookieExpirationSeconds = 2 * JWTExpirationSeconds // double jwt

	DisableGZIP         = true
	DisableVersionCheck = false
)

// SetEncryptKey sets the encryption key from configuration
// This should be called once during application initialization
func SetEncryptKey(key string) {
	if key != "" {
		tigaEncryptKey = key
	}
}

// GetEncryptKey returns the encryption key
// Use SetEncryptKey() during initialization to set from config
func GetEncryptKey() string {
	return tigaEncryptKey
}
