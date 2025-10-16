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
// DEPRECATED: These variables are deprecated and will be removed in a future version.
// Use internal/config.Config instead:
//   - Host -> No longer needed
//   - NodeTerminalImage -> config.Kubernetes.NodeTerminalImage
//   - WebhookUsername/WebhookPassword/WebhookEnabled -> config.Webhook.*
//   - tigaEncryptKey (via Get/SetEncryptKey) -> config.Security.EncryptionKey
//   - AnonymousUserEnabled -> config.Features.AnonymousUserEnabled
//   - CookieExpirationSeconds -> config.JWT.ExpiresIn * 2
//   - DisableGZIP -> config.Features.DisableGZIP
//   - DisableVersionCheck -> config.Features.DisableVersionCheck
//
// TODO: Migrate all usages to internal/config system and remove this package
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

// Deprecated: SetEncryptKey sets the encryption key from configuration.
// Use config.Security.EncryptionKey directly instead.
// This function will be removed in a future version.
func SetEncryptKey(key string) {
	if key != "" {
		tigaEncryptKey = key
	}
}

// Deprecated: GetEncryptKey returns the encryption key.
// Use config.Security.EncryptionKey directly instead.
// This function will be removed in a future version.
func GetEncryptKey() string {
	return tigaEncryptKey
}
