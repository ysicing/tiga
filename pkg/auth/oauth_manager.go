package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/pkg/common"
)

type OAuthManager struct {
	jwtSecret         string
	oauthProviderRepo *repository.OAuthProviderRepository
}

func NewOAuthManager(jwtSecret string) *OAuthManager {
	return &OAuthManager{
		jwtSecret:         jwtSecret,
		oauthProviderRepo: repository.NewOAuthProviderRepository(models.DB),
	}
}

func getRequestHost(c *gin.Context) string {
	if common.Host != "" {
		return common.Host
	}
	proto := c.Request.Header.Get("X-Forwarded-Proto")
	host := c.Request.Header.Get("X-Forwarded-Host")
	if proto != "" && host != "" {
		return proto + "://" + host
	}
	if c.Request.Host != "" {
		scheme := "https"
		if c.Request.TLS == nil {
			scheme = "http"
		}
		return scheme + "://" + c.Request.Host
	}

	return "http://localhost"
}

func (om *OAuthManager) GetProvider(c *gin.Context, name string) (OAuthProvider, error) {
	dbProvider, err := om.oauthProviderRepo.GetByName(context.Background(), name)
	if err != nil {
		return nil, err
	}
	dbProvider.RedirectURL, _ = url.JoinPath(getRequestHost(c), "/api/auth/callback")
	return NewGenericProvider(*dbProvider)
}

func (om *OAuthManager) GetAvailableProviders() []string {
	var providers []string
	dbProviders, err := om.oauthProviderRepo.ListEnabled(context.Background())
	if err != nil {
		logrus.Warnf("Failed to load OAuth providers from database: %v", err)
	} else {
		for _, provider := range dbProviders {
			providers = append(providers, string(provider.Name))
		}
	}
	return providers
}

func (om *OAuthManager) GenerateState() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func (om *OAuthManager) GenerateJWT(user *models.User, refreshToken string) (string, error) {
	now := time.Now()
	expirationTime := now.Add(common.JWTExpirationSeconds * time.Second)

	claims := Claims{
		UserID:       user.ID.String(), // Convert UUID to string
		Username:     user.Username,
		Name:         user.Name,
		AvatarURL:    user.AvatarURL,
		Provider:     user.Provider,
		RefreshToken: refreshToken,
		OIDCGroups:   user.OIDCGroups,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "tiga",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(om.jwtSecret))
}

func (om *OAuthManager) ValidateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(om.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func (om *OAuthManager) RefreshJWT(c *gin.Context, tokenString string) (string, error) {
	// We need to accept expired tokens here so we can extract claims and refresh using
	// the stored refresh token. Parse the token while skipping claims validation but
	// still validate the signature.
	var claims Claims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(om.jwtSecret), nil
	}, jwt.WithoutClaimsValidation())
	if err != nil {
		return "", err
	}

	// Ensure signature was valid
	if token == nil || !token.Valid {
		return "", fmt.Errorf("invalid token signature")
	}

	// Check if token is close to expiration (within 1 hour)
	if time.Until(claims.ExpiresAt.Time) > time.Hour {
		// Token not close to expiry - return original token
		return tokenString, nil
	}

	// If we have a refresh token, try to refresh the OAuth token
	if claims.RefreshToken != "" {
		provider, err := om.GetProvider(c, claims.Provider)
		if err != nil {
			return "", err
		}

		tokenResp, err := provider.RefreshToken(claims.RefreshToken)
		if err != nil {
			return "", err
		}

		// Get updated user info with new access token
		user, err := provider.GetUserInfo(tokenResp.AccessToken)
		if err != nil {
			return "", err
		}

		// Generate new JWT with refreshed token
		newRefreshToken := tokenResp.RefreshToken
		if newRefreshToken == "" {
			newRefreshToken = claims.RefreshToken // Keep the old refresh token if no new one provided
		}

		return om.GenerateJWT(user, newRefreshToken)
	}

	// If no refresh token available, just generate a new JWT with existing claims
	// This is for providers like GitHub that don't expire tokens
	// Parse the UUID from the claims.UserID string
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID in claims: %w", err)
	}

	user := &models.User{
		ID:         userID,
		Username:   claims.Username,
		Name:       claims.Name,
		AvatarURL:  claims.AvatarURL,
		Provider:   claims.Provider,
		OIDCGroups: claims.OIDCGroups,
	}

	return om.GenerateJWT(user, "")
}
