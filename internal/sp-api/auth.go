package sp_api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/config"
)

// AuthConfig represents the authentication configuration for the Amazon SP-API.
// It contains all the necessary credentials and endpoints needed to authenticate
// with the API.
type AuthConfig struct {
	// Get this value when you register your application.
	// Refer to https://developer-docs.amazon.com/sp-api/docs/viewing-your-application-information-and-credentials
	//
	// Required.
	ClientID string
	// Get this value when you register your application.
	// Refer to https://developer-docs.amazon.com/sp-api/docs/viewing-your-application-information-and-credentials
	//
	// Required.
	ClientSecret string
	// The LWA refresh token. Get this value when the selling partner authorizes your application.
	// For more information, refer to https://developer-docs.amazon.com/sp-api/docs/authorizing-selling-partner-api-applications
	//
	// Not required. Include refresh_token for calling operations that require authorization from a selling partner.
	// If you include refresh_token, do not include scope.
	RefreshToken string
	// A Selling Partner API endpoint.
	// See https://developer-docs.amazon.com/sp-api/docs/sp-api-endpoints
	//
	// Required.
	Endpoint string
}

// TokenResponse represents the response structure returned by the Amazon Selling Partner API
// during authentication. It contains access credentials and related information.
type TokenResponse struct {
	// The LWA access token. Maximum size: 2048 bytes.
	AccessToken string `json:"access_token"`
	// The type of token returned. Must be bearer.
	TokenType string `json:"token_type"`
	// The number of seconds before the LWA access token becomes invalid.
	ExpiresIn int `json:"expires_in"`
	// The LWA refresh token that you submitted in the request. Maximum size: 2048 bytes.
	RefreshToken string `json:"refresh_token"`
}

// TokenManager handles authentication token management for the SP-API.
// It manages token retrieval, caching, and automatic renewal when tokens expire.
type TokenManager struct {
	// config contains authentication configuration settings
	config AuthConfig
	// currentToken stores the active access token
	currentToken string
	// expiresAt tracks when the current token will expire
	expiresAt time.Time
	// mutex protects concurrent access to token data
	mutex sync.Mutex
}

// NewTokenManager creates a new TokenManager with the given AuthConfig.
// It initializes the TokenManager with the provided configuration for Amazon SP-API authentication.
// The returned TokenManager can be used to generate and manage access tokens for API requests.
func NewTokenManager(config AuthConfig) *TokenManager {
	return &TokenManager{
		config: config,
	}
}

// GetAccessToken returns a valid access token for the API.
// It checks if the current token is still valid based on expiration time.
// If the current token is valid, it returns it immediately.
// Otherwise, it refreshes the token and returns the new one.
// The function is thread-safe due to mutex lock/unlock operations.
func (tm *TokenManager) GetAccessToken() (string, error) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	if tm.currentToken != "" && time.Now().Before(tm.expiresAt) {
		return tm.currentToken, nil
	}
	return tm.refreshToken()
}

func (tm *TokenManager) refreshToken() (string, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", tm.config.RefreshToken)
	data.Set("client_id", tm.config.ClientID)
	data.Set("client_secret", tm.config.ClientSecret)

	req, err := http.NewRequest("POST", tm.config.Endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req) //nolint: bodyclose
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer internal.CloseReader(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("error response: %s - %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}

	tm.currentToken = tokenResp.AccessToken
	// subtract 5 minutes for safety margin
	tm.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn-300) * time.Second)
	if tokenResp.RefreshToken != "" {
		tm.config.RefreshToken = tokenResp.RefreshToken
		if err := config.SnapshotToDisk(); err != nil {
			return "", fmt.Errorf("error saving refresh token: %w", err)
		}
	}

	return tm.currentToken, nil
}
