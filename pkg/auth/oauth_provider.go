package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
)

// OAuthProvider defines the interface for OAuth providers
type OAuthProvider interface {
	GetAuthURL(state string) string
	ExchangeCodeForToken(code string) (*TokenResponse, error)
	GetUserInfo(accessToken string) (*models.User, error)
	RefreshToken(refreshToken string) (*TokenResponse, error)
	GetProviderName() string
}

// OAuthConfig holds common OAuth configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       string
}

// TokenResponse represents OAuth token response with refresh token support
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	Scope        string `json:"scope"`
}

// Claims represents JWT claims with refresh token support
type Claims struct {
	UserID       string   `json:"user_id"` // Changed from uint to string for UUID support
	Username     string   `json:"username"`
	Name         string   `json:"name"`
	AvatarURL    string   `json:"avatar_url"`
	Provider     string   `json:"provider"`
	RefreshToken string   `json:"refresh_token,omitempty"`
	OIDCGroups   []string `json:"oidc_groups,omitempty"`
	jwt.RegisteredClaims
}

type GenericProvider struct {
	Config      OAuthConfig
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	Name        string
}

// discoverOAuthEndpoints discovers OAuth endpoints from issuer's well-known configuration
// TODO: cache well-known configuration
func discoverOAuthEndpoints(issuer, providerName string) (*struct {
	AuthURL     string
	TokenURL    string
	UserInfoURL string
}, error) {
	wellKnown := issuer
	var err error
	if !strings.HasSuffix(issuer, "/.well-known/openid-configuration") {
		wellKnown, err = url.JoinPath(issuer, ".well-known", "openid-configuration")
		if err != nil {
			return nil, fmt.Errorf("failed to construct well-known URL: %w", err)
		}
	}

	// Create HTTP client with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnown, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch well-known configuration: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch well-known configuration: HTTP %d", resp.StatusCode)
	}

	var meta struct {
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
		UserinfoEndpoint      string `json:"userinfo_endpoint"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		logrus.Warnf("Failed to parse well-known configuration for %s: %v", providerName, err)
		return nil, fmt.Errorf("failed to parse well-known configuration: %w", err)
	}

	logrus.Debugf("Discovered %s openid configuration", providerName)
	return &struct {
		AuthURL     string
		TokenURL    string
		UserInfoURL string
	}{
		AuthURL:     meta.AuthorizationEndpoint,
		TokenURL:    meta.TokenEndpoint,
		UserInfoURL: meta.UserinfoEndpoint,
	}, nil
}

func NewGenericProvider(op models.OAuthProvider) (*GenericProvider, error) {
	if op.Issuer != "" && (op.AuthURL == "" || op.TokenURL == "" || op.UserInfoURL == "") {
		meta, err := discoverOAuthEndpoints(op.Issuer, string(op.Name))
		if err != nil {
			logrus.Errorf("Failed to discover OAuth endpoints for %s: %v", op.Name, err)
			return nil, err
		}
		op.AuthURL = meta.AuthURL
		op.TokenURL = meta.TokenURL
		op.UserInfoURL = meta.UserInfoURL
	}
	if op.AuthURL == "" || op.TokenURL == "" || op.UserInfoURL == "" {
		return nil, fmt.Errorf("provider %s is missing required URLs", op.Name)
	}

	scopes := []string{}
	if op.Scopes != "" {
		scopes = strings.Split(op.Scopes, ",")
	}

	gp := &GenericProvider{
		Config: OAuthConfig{
			ClientID:     op.ClientID,
			ClientSecret: string(op.ClientSecret),
			RedirectURL:  op.RedirectURL,
			Scopes:       strings.Join(scopes, " "),
		},
		AuthURL:     op.AuthURL,
		TokenURL:    op.TokenURL,
		UserInfoURL: op.UserInfoURL,
		Name:        string(op.Name),
	}
	return gp, nil
}

func (g *GenericProvider) GetProviderName() string {
	return g.Name
}

func (g *GenericProvider) GetAuthURL(state string) string {
	params := url.Values{}
	params.Add("client_id", g.Config.ClientID)
	params.Add("redirect_uri", g.Config.RedirectURL)
	// TODO: fix me
	params.Add("scope", g.Config.Scopes)
	params.Add("state", state)
	params.Add("response_type", "code")

	return g.AuthURL + "?" + params.Encode()
}

func (g *GenericProvider) ExchangeCodeForToken(code string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", g.Config.ClientID)
	data.Set("client_secret", g.Config.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", g.Config.RedirectURL)

	req, err := http.NewRequest("POST", g.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (g *GenericProvider) RefreshToken(refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", g.Config.ClientID)
	data.Set("client_secret", g.Config.ClientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequest("POST", g.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (g *GenericProvider) GetUserInfo(accessToken string) (*models.User, error) {
	req, err := http.NewRequest("GET", g.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	logrus.Debugf("User info from %s: %v", g.Name, userInfo)

	// Map common fields - this might need customization per provider
	user := &models.User{
		Provider: g.Name,
	}

	user.Sub = ""
	if id, ok := userInfo["id"]; ok {
		user.Sub = fmt.Sprintf("%v", id)
	} else if sub, ok := userInfo["sub"]; ok {
		user.Sub = fmt.Sprintf("%v", sub)
	}
	if username, ok := userInfo["username"]; ok {
		user.Username = fmt.Sprintf("%v", username)
	} else if login, ok := userInfo["login"]; ok {
		user.Username = fmt.Sprintf("%v", login)
	} else if email, ok := userInfo["email"]; ok {
		user.Username = fmt.Sprintf("%v", email)
	}
	if name, ok := userInfo["name"]; ok {
		user.Name = fmt.Sprintf("%v", name)
	}
	if avatar, ok := userInfo["avatar_url"]; ok {
		user.AvatarURL = fmt.Sprintf("%v", avatar)
	} else if picture, ok := userInfo["picture"]; ok {
		user.AvatarURL = fmt.Sprintf("%v", picture)
	}

	var groups []interface{}
	if v, ok := userInfo["groups"]; ok {
		if arr, ok := v.([]interface{}); ok {
			groups = arr
		}
	} else if roles, ok := userInfo["roles"]; ok {
		if arr, ok := roles.([]interface{}); ok {
			groups = arr
		}
	}

	if len(groups) != 0 {
		user.OIDCGroups = make([]string, len(groups))
		for i, v := range groups {
			user.OIDCGroups[i] = fmt.Sprintf("%v", v)
		}
	}
	return user, nil
}
