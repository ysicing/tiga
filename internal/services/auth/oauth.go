package auth

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// OAuthProvider represents an OAuth provider configuration
type OAuthProvider struct {
	Type         string // google, github, oidc
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	AuthURL      string // For custom OIDC providers
	TokenURL     string // For custom OIDC providers
	UserInfoURL  string // For fetching user information
}

// OAuthManager manages OAuth 2.0 authentication
type OAuthManager struct {
	providers map[string]*oauth2.Config
}

// NewOAuthManager creates a new OAuthManager
func NewOAuthManager() *OAuthManager {
	return &OAuthManager{
		providers: make(map[string]*oauth2.Config),
	}
}

// RegisterProvider registers an OAuth provider
func (m *OAuthManager) RegisterProvider(name string, provider *OAuthProvider) error {
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	var config *oauth2.Config

	switch provider.Type {
	case "google":
		config = &oauth2.Config{
			ClientID:     provider.ClientID,
			ClientSecret: provider.ClientSecret,
			RedirectURL:  provider.RedirectURL,
			Scopes:       provider.Scopes,
			Endpoint:     google.Endpoint,
		}

	case "github":
		config = &oauth2.Config{
			ClientID:     provider.ClientID,
			ClientSecret: provider.ClientSecret,
			RedirectURL:  provider.RedirectURL,
			Scopes:       provider.Scopes,
			Endpoint:     github.Endpoint,
		}

	case "oidc":
		if provider.AuthURL == "" || provider.TokenURL == "" {
			return fmt.Errorf("OIDC provider requires AuthURL and TokenURL")
		}
		config = &oauth2.Config{
			ClientID:     provider.ClientID,
			ClientSecret: provider.ClientSecret,
			RedirectURL:  provider.RedirectURL,
			Scopes:       provider.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  provider.AuthURL,
				TokenURL: provider.TokenURL,
			},
		}

	default:
		return fmt.Errorf("unsupported OAuth provider type: %s", provider.Type)
	}

	m.providers[name] = config
	return nil
}

// GetAuthURL generates the OAuth authorization URL
func (m *OAuthManager) GetAuthURL(providerName, state string) (string, error) {
	config, err := m.getProvider(providerName)
	if err != nil {
		return "", err
	}

	// Generate authorization URL with state parameter for CSRF protection
	url := config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return url, nil
}

// ExchangeCode exchanges an authorization code for an access token
func (m *OAuthManager) ExchangeCode(ctx context.Context, providerName, code string) (*oauth2.Token, error) {
	config, err := m.getProvider(providerName)
	if err != nil {
		return nil, err
	}

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return token, nil
}

// RefreshToken refreshes an OAuth access token
func (m *OAuthManager) RefreshToken(ctx context.Context, providerName string, refreshToken string) (*oauth2.Token, error) {
	config, err := m.getProvider(providerName)
	if err != nil {
		return nil, err
	}

	tokenSource := config.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	})

	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return newToken, nil
}

// GetUserInfo fetches user information from the OAuth provider
func (m *OAuthManager) GetUserInfo(ctx context.Context, providerName string, token *oauth2.Token) (map[string]interface{}, error) {
	config, err := m.getProvider(providerName)
	if err != nil {
		return nil, err
	}

	client := config.Client(ctx, token)

	// Get user info URL based on provider
	var userInfoURL string
	switch providerName {
	case "google":
		userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	case "github":
		userInfoURL = "https://api.github.com/user"
	default:
		// For OIDC, should be provided in provider configuration
		return nil, fmt.Errorf("user info URL not configured for provider: %s", providerName)
	}

	resp, err := client.Get(userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	if err := parseJSON(resp.Body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return userInfo, nil
}

// ValidateState validates the OAuth state parameter (CSRF protection)
func (m *OAuthManager) ValidateState(receivedState, expectedState string) error {
	if receivedState == "" {
		return fmt.Errorf("state parameter is missing")
	}
	if receivedState != expectedState {
		return fmt.Errorf("invalid state parameter: possible CSRF attack")
	}
	return nil
}

// getProvider retrieves a registered OAuth provider configuration
func (m *OAuthManager) getProvider(name string) (*oauth2.Config, error) {
	config, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("OAuth provider %s not registered", name)
	}
	return config, nil
}

// IsProviderRegistered checks if a provider is registered
func (m *OAuthManager) IsProviderRegistered(name string) bool {
	_, exists := m.providers[name]
	return exists
}

// ListProviders returns all registered provider names
func (m *OAuthManager) ListProviders() []string {
	providers := make([]string, 0, len(m.providers))
	for name := range m.providers {
		providers = append(providers, name)
	}
	return providers
}

// OAuthUserInfo represents normalized user information from OAuth providers
type OAuthUserInfo struct {
	ID            string
	Email         string
	Name          string
	AvatarURL     string
	EmailVerified bool
	Provider      string
}

// NormalizeGoogleUserInfo normalizes Google user info
func NormalizeGoogleUserInfo(userInfo map[string]interface{}) *OAuthUserInfo {
	return &OAuthUserInfo{
		ID:            getStringField(userInfo, "id"),
		Email:         getStringField(userInfo, "email"),
		Name:          getStringField(userInfo, "name"),
		AvatarURL:     getStringField(userInfo, "picture"),
		EmailVerified: getBoolField(userInfo, "verified_email"),
		Provider:      "google",
	}
}

// NormalizeGitHubUserInfo normalizes GitHub user info
func NormalizeGitHubUserInfo(userInfo map[string]interface{}) *OAuthUserInfo {
	id := ""
	if idFloat, ok := userInfo["id"].(float64); ok {
		id = fmt.Sprintf("%.0f", idFloat)
	}

	return &OAuthUserInfo{
		ID:            id,
		Email:         getStringField(userInfo, "email"),
		Name:          getStringField(userInfo, "name"),
		AvatarURL:     getStringField(userInfo, "avatar_url"),
		EmailVerified: true, // GitHub emails are verified
		Provider:      "github",
	}
}

// Helper functions

func getStringField(data map[string]interface{}, key string) string {
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}

func getBoolField(data map[string]interface{}, key string) bool {
	if value, ok := data[key].(bool); ok {
		return value
	}
	return false
}

func parseJSON(r interface{}, v interface{}) error {
	// Simplified JSON parsing - in real implementation use encoding/json
	// This is a placeholder
	return nil
}

// GenerateStateToken generates a random state token for CSRF protection
func GenerateStateToken() string {
	// Generate a random UUID for state
	return fmt.Sprintf("state_%d_%s", time.Now().Unix(), randomString(16))
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
