package amazon

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
	"github.com/caner-cetin/halycon/internal/amazon/catalog/client/catalog"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inbound/client/fba_inbound"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inventory/client/fba_inventory"
	"github.com/caner-cetin/halycon/internal/amazon/listings/client/listings"
	"github.com/caner-cetin/halycon/internal/config"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
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

const (
	CatalogServiceName      = "services.catalog"
	ListingsServiceName     = "services.listings"
	FBAInboundServiceName   = "services.fba.inbound"
	FBAInventoryServiceName = "services.fba.inventory"
)

type AuthorizedClient struct {
	services map[string]interface{}
	authInfo runtime.ClientAuthInfoWriter
}

func (a *AuthorizedClient) AddService(name string, service interface{}) {
	a.services[name] = service
}

func (a *AuthorizedClient) GetService(name string) interface{} {
	return a.services[name]
}

func (a *AuthorizedClient) GetCatalogService() catalog.ClientService {
	return a.services[CatalogServiceName].(catalog.ClientService)
}

func (a *AuthorizedClient) GetListingsService() listings.ClientService {
	return a.services[ListingsServiceName].(listings.ClientService)
}

func (a *AuthorizedClient) GetFBAInboundService() fba_inbound.ClientService {
	return a.services[FBAInboundServiceName].(fba_inbound.ClientService)
}

func (a *AuthorizedClient) GetFBAInventoryService() fba_inventory.ClientService {
	return a.services[FBAInventoryServiceName].(fba_inventory.ClientService)
}

func (a *AuthorizedClient) SearchCatalogItems(params *catalog.SearchCatalogItemsParams) (*catalog.SearchCatalogItemsOK, error) {
	catalogClient := a.GetCatalogService()
	authOpt := func(op *runtime.ClientOperation) {
		op.AuthInfo = a.authInfo
	}
	return catalogClient.SearchCatalogItems(params, authOpt, internal.ConfigureRateLimiting(2, 2))
}

func (a *AuthorizedClient) GetCatalogItem(params *catalog.GetCatalogItemParams) (*catalog.GetCatalogItemOK, error) {
	catalogClient := a.GetCatalogService()
	authOpt := func(op *runtime.ClientOperation) {
		op.AuthInfo = a.authInfo
	}
	return catalogClient.GetCatalogItem(params, authOpt, internal.ConfigureRateLimiting(2, 2))
}

func (a *AuthorizedClient) GetListingsItem(params *listings.GetListingsItemParams) (*listings.GetListingsItemOK, error) {
	listingsClient := a.GetListingsService()
	authOpt := func(op *runtime.ClientOperation) {
		op.AuthInfo = a.authInfo
	}
	return listingsClient.GetListingsItem(params, authOpt, internal.ConfigureRateLimiting(5, 10))
}

func (a *AuthorizedClient) GetFBAInventorySummaries(params *fba_inventory.GetInventorySummariesParams) (*fba_inventory.GetInventorySummariesOK, error) {
	inventoryClient := a.GetFBAInventoryService()
	authOpt := func(op *runtime.ClientOperation) {
		op.AuthInfo = a.authInfo
	}
	return inventoryClient.GetInventorySummaries(params, authOpt, internal.ConfigureRateLimiting(2, 2))
}

func (a *AuthorizedClient) CreateFBAInboundPlan(params *fba_inbound.CreateInboundPlanParams) (*fba_inbound.CreateInboundPlanAccepted, error) {
	inboundClient := a.GetFBAInboundService()
	authOpt := func(op *runtime.ClientOperation) {
		op.AuthInfo = a.authInfo
	}
	return inboundClient.CreateInboundPlan(params, authOpt, internal.ConfigureRateLimiting(2, 2))
}

func NewAuthorizedClient(token string) *AuthorizedClient {
	authInfo := runtime.ClientAuthInfoWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
		// doesnt work without Authorization header
		if err := r.SetHeaderParam("Authorization", "Bearer "+token); err != nil {
			return err
		}
		// also doesnt work without x-amz-access-token header
		return r.SetHeaderParam("x-amz-access-token", token)
	})

	return &AuthorizedClient{
		services: map[string]interface{}{},
		authInfo: authInfo,
	}
}
