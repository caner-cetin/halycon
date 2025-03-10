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

	"github.com/caner-cetin/halycon/internal/config"
	"github.com/spf13/viper"
)

type AuthConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	Endpoint     string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type TokenManager struct {
	config       AuthConfig
	currentToken string
	expiresAt    time.Time
	mutex        sync.Mutex
}

func NewTokenManager(config AuthConfig) *TokenManager {
	return &TokenManager{
		config: config,
	}
}

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
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("error response: %s - %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("error parsing response: %v", err)
	}

	tm.currentToken = tokenResp.AccessToken
	// subtract 5 minutes for safety margin
	tm.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn-300) * time.Second)
	if tokenResp.RefreshToken != "" {
		tm.config.RefreshToken = tokenResp.RefreshToken
		if viper.IsSet(config.AMAZON_AUTH_REFRESH_TOKEN.Key) {
			viper.Set(config.AMAZON_AUTH_REFRESH_TOKEN.Key, tokenResp.RefreshToken)
			viper.WriteConfig()
		}
	}

	return tm.currentToken, nil
}
