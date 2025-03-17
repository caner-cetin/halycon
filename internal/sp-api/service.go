package sp_api

import (
	"fmt"
	"time"

	osRuntime "runtime"

	"github.com/caner-cetin/halycon/internal/amazon/catalog/client/catalog"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inbound/client/fba_inbound"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inventory/client/fba_inventory"
	"github.com/caner-cetin/halycon/internal/amazon/listings/client/listings"
	"github.com/caner-cetin/halycon/internal/amazon/product_type_definitions/client/definitions"
	"github.com/caner-cetin/halycon/internal/config"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

const (
	// CatalogServiceName represents the Amazon Selling Partner Catalog API service
	CatalogServiceName = "services.catalog"
	// ListingsServiceName represents the Amazon Selling Partner Listings API service
	ListingsServiceName = "services.listings"
	// FBAInboundServiceName represents the Amazon Selling Partner FBA Inbound API service
	FBAInboundServiceName = "services.fba.inbound"
	// FBAInventoryServiceName represents the Amazon Selling Partner FBA Inventory API service
	FBAInventoryServiceName = "services.fba.inventory"
	// ProductTypeDefinitionsServiceName represents the Amazon Selling Partner Product Type Definitions API service
	ProductTypeDefinitionsServiceName = "services.listings.product_type.definitions"
)

// Client is responsible for managing the Amazon Selling Partner API services.
// It maintains a collection of service instances, rate limiters for each service,
// and authentication information required for API calls.
type Client struct {
	// services holds instances of various SP-API service clients indexed by service name
	services map[string]interface{}

	// rateLimiters manages the rate limiting for each service to comply with SP-API usage limits
	rateLimiters map[string]*rate.Limiter

	// authInfo contains the authentication credentials and information required for API calls
	authInfo runtime.ClientAuthInfoWriter

	// Token is the access token used for authentication
	Token string

	// TokenManager is the token manager used to acquire and refresh access tokens
	TokenManager *TokenManager
}

// AddService adds a service with the given name to the client's services map.
func (a *Client) AddService(name string, service interface{}) {
	a.services[name] = service
}

// GetService returns a service by name from the client's services map.
func (a *Client) GetService(name string) interface{} {
	return a.services[name]
}

// GetCatalogService returns the catalog service implementation from the client's service map.
func (a *Client) GetCatalogService() catalog.ClientService {
	return a.services[CatalogServiceName].(catalog.ClientService)
}

// GetListingsService returns the ListingsService implementation from the client's service map.
func (a *Client) GetListingsService() listings.ClientService {
	return a.services[ListingsServiceName].(listings.ClientService)
}

// GetFBAInboundService returns the FBA inbound client service from the client's service map.
func (a *Client) GetFBAInboundService() fba_inbound.ClientService {
	return a.services[FBAInboundServiceName].(fba_inbound.ClientService)
}

// GetFBAInventoryService returns the FBA Inventory client service from the client's service map.
func (a *Client) GetFBAInventoryService() fba_inventory.ClientService {
	return a.services[FBAInventoryServiceName].(fba_inventory.ClientService)
}

// GetProductTypeDefinitionsService returns the ProductTypeDefinitions service client implementation.
func (a *Client) GetProductTypeDefinitionsService() definitions.ClientService {
	return a.services[ProductTypeDefinitionsServiceName].(definitions.ClientService)
}

// SearchCatalogItems searches for catalog items based on the provided parameters.
func (a *Client) SearchCatalogItems(params *catalog.SearchCatalogItemsParams) (*catalog.SearchCatalogItemsOK, error) {
	return a.GetCatalogService().SearchCatalogItems(params, a.WithAuth(), a.WithRateLimit(SearchCatalogItemsRLKey))
}

// GetCatalogItem retrieves a catalog item using the provided parameters, with authentication and rate limiting.
func (a *Client) GetCatalogItem(params *catalog.GetCatalogItemParams) (*catalog.GetCatalogItemOK, error) {
	return a.GetCatalogService().GetCatalogItem(params, a.WithAuth(), a.WithRateLimit(GetCatalogItemsRLKey))
}

// GetListingsItem retrieves a specific listings item using the provided parameters.
func (a *Client) GetListingsItem(params *listings.GetListingsItemParams) (*listings.GetListingsItemOK, error) {
	return a.GetListingsService().GetListingsItem(params, a.WithAuth(), a.WithRateLimit(GetListingsItemRLKey))
}

// DeleteListingsItem removes a listings item from Amazon's catalog
func (a *Client) DeleteListingsItem(params *listings.DeleteListingsItemParams) (*listings.DeleteListingsItemOK, error) {
	return a.GetListingsService().DeleteListingsItem(params, a.WithAuth(), a.WithRateLimit(DeleteListingsItemRLKey))
}

// GetFBAInventorySummaries retrieves FBA inventory summaries
func (a *Client) GetFBAInventorySummaries(params *fba_inventory.GetInventorySummariesParams) (*fba_inventory.GetInventorySummariesOK, error) {
	return a.GetFBAInventoryService().GetInventorySummaries(params, a.WithAuth(), a.WithRateLimit(FBAInventorySummariesRLKey))
}

// CreateFBAInboundPlan creates an inbound plan for FBA (Fulfillment by Amazon)
func (a *Client) CreateFBAInboundPlan(params *fba_inbound.CreateInboundPlanParams) (*fba_inbound.CreateInboundPlanAccepted, error) {
	return a.GetFBAInboundService().CreateInboundPlan(params, a.WithAuth(), a.WithRateLimit(CreateInboundPlanRLKey))
}

// GetInboundOperationStatus retrieves the status of an inbound operation for a fulfillment by Amazon order.
func (a *Client) GetInboundOperationStatus(params *fba_inbound.GetInboundOperationStatusParams) (*fba_inbound.GetInboundOperationStatusOK, error) {
	return a.GetFBAInboundService().GetInboundOperationStatus(params, a.WithAuth(), a.WithRateLimit(GetInboundOperationStatusRLKey))
}

// SearchProductTypeDefinitions searches for product type definitions based on provided search parameters.
func (a *Client) SearchProductTypeDefinitions(params *definitions.SearchDefinitionsProductTypesParams) (*definitions.SearchDefinitionsProductTypesOK, error) {
	return a.GetProductTypeDefinitionsService().SearchDefinitionsProductTypes(params, a.WithAuth(), a.WithRateLimit(SearchProductTypeDefinitionsRLKey))
}

// GetProductTypeDefinition retrieves the product type definition.
func (a *Client) GetProductTypeDefinition(params *definitions.GetDefinitionsProductTypeParams) (*definitions.GetDefinitionsProductTypeOK, error) {
	return a.GetProductTypeDefinitionsService().GetDefinitionsProductType(params, a.WithAuth(), a.WithRateLimit(GetProductTypeDefinitionRLKey))
}

// CreateListing submits a new product listing under FBM program.
func (a *Client) CreateListing(params *listings.PutListingsItemParams) (*listings.PutListingsItemOK, error) {
	return a.GetListingsService().PutListingsItem(params, a.WithAuth(), a.WithRateLimit(CreateListingRLKey))
}

// GetCatalog retrieves catalog item information.
func (a *Client) GetCatalog(params *catalog.GetCatalogItemParams) (*catalog.GetCatalogItemOK, error) {
	return a.GetCatalogService().GetCatalogItem(params, a.WithAuth())
}

// rate limiter keys for client's rate limiter mapping, each one of them leads to a rate.Limiter instance.
const (
	SearchCatalogItemsRLKey           = "rate_limiter.catalog.searchItems"
	GetCatalogItemsRLKey              = "rate_limiter.catalog.getItems"
	GetListingsItemRLKey              = "rate_limiter.listings.getItem"
	DeleteListingsItemRLKey           = "rate_limiter.listings.deleteListingsItem"
	CreateListingRLKey                = "rate_limiter.listings.createItem"
	FBAInventorySummariesRLKey        = "rate_limiter.fba.inventorySummaries"
	CreateInboundPlanRLKey            = "rate_limiter.fba.createInboundPlan"
	GetInboundOperationStatusRLKey    = "rate_limiter.fba.getInboundOperationStatus"
	SearchProductTypeDefinitionsRLKey = "rate_limiter.listings.search_product_type_definitions"
	GetProductTypeDefinitionRLKey     = "rate_limiter.listings.get_product_type_definitions"
)

// SetRateLimits populates the rateLimiters map with predefined rate limits
// Each rate limiter is configured with a specific rate (operations per second) and burst capacity.
// The rate limiters help ensure API calls comply with Amazon's throttling requirements
// and prevent exceeding their service quotas.
func (a *Client) SetRateLimits() {
	a.rateLimiters = map[string]*rate.Limiter{
		SearchCatalogItemsRLKey:           rate.NewLimiter(rate.Limit(2), 2),
		GetCatalogItemsRLKey:              rate.NewLimiter(rate.Limit(2), 2),
		GetListingsItemRLKey:              rate.NewLimiter(rate.Limit(5), 10),
		DeleteListingsItemRLKey:           rate.NewLimiter(rate.Limit(5), 10),
		FBAInventorySummariesRLKey:        rate.NewLimiter(rate.Limit(2), 2),
		CreateInboundPlanRLKey:            rate.NewLimiter(rate.Limit(2), 2),
		GetInboundOperationStatusRLKey:    rate.NewLimiter(rate.Limit(2), 6),
		SearchProductTypeDefinitionsRLKey: rate.NewLimiter(rate.Limit(5), 10),
		GetProductTypeDefinitionRLKey:     rate.NewLimiter(rate.Limit(5), 10),
		CreateListingRLKey:                rate.NewLimiter(rate.Limit(5), 10),
	}
}

// NewAuthorizedClient creates and returns a new authenticated SP-API client with the provided access token.
// It sets up necessary authentication headers including:
// - Bearer token authorization
// - Amazon specific access token headers
// - Host header
// - Request timestamp
// - User agent
func NewAuthorizedClient() (*Client, error) {
	client := config.Config.Amazon.Auth.DefaultClient
	merchant := config.Config.Amazon.Auth.DefaultMerchant
	var auth = AuthConfig{
		ClientID:     client.ID,
		ClientSecret: client.Secret,
		RefreshToken: merchant.RefreshToken,
		Endpoint:     client.AuthEndpoint,
	}
	var tokenManager = NewTokenManager(auth)
	token, err := tokenManager.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("error while acquiring access token: %w", err)
	}
	log.Debug().Str("token_prefix", token[:10]+"...").Msg("acquired access token")
	authInfo := runtime.ClientAuthInfoWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
		// doesnt work without Authorization header
		if err := r.SetHeaderParam("Authorization", "Bearer "+token); err != nil {
			return fmt.Errorf("failed to set Authorization header: %w", err)
		}
		// also doesnt work without x-amz-access-token header
		if err := r.SetHeaderParam("x-amz-access-token", token); err != nil {
			return fmt.Errorf("failed to set x-amz-access-token header: %w", err)
		}
		// also doesnt work without X-Amz-Access-Token header
		if err := r.SetHeaderParam("X-Amz-Access-Token", token); err != nil {
			return fmt.Errorf("failed to set X-Amz-Access-Token header: %w", err)
		}
		if err := r.SetHeaderParam("host", fmt.Sprintf("https://%s", client.APIEndpoint)); err != nil {
			return fmt.Errorf("failed to set host header: %w", err)
		}
		if err := r.SetHeaderParam("x-amz-date", time.Now().UTC().Format("20060102T150405Z")); err != nil {
			return fmt.Errorf("failed to set x-amz-date header: %w", err)
		}
		if err := r.SetHeaderParam("user-agent", fmt.Sprintf("Halycon/0.2 (Language=Go; Platform=%s)", osRuntime.GOOS)); err != nil {
			return fmt.Errorf("failed to set user-agent header: %w", err)
		}
		return nil
	})

	a := &Client{
		services:     map[string]interface{}{},
		authInfo:     authInfo,
		TokenManager: tokenManager,
		Token:        token,
	}
	a.SetRateLimits()
	return a, nil
}

// WithRateLimit returns a function that applies rate limiting to a client operation.
// It takes a key string that identifies which rate limiter to use from the client's rate limiters map.
// The returned function, when called with a runtime.ClientOperation, will replace its Client with a
// rate-limited HTTP client that enforces the rate limits defined for the given key.
func (a *Client) WithRateLimit(key string) func(op *runtime.ClientOperation) {
	return func(op *runtime.ClientOperation) {
		httpClient := NewRateLimitedClient(a.rateLimiters[key])
		op.Client = httpClient
	}
}

// WithAuth returns a function that is used to set the authentication information for a client operation.
// This is typically used with the client operation configuration to ensure that the request is authenticated.
func (a *Client) WithAuth() func(op *runtime.ClientOperation) {
	return func(op *runtime.ClientOperation) {
		op.AuthInfo = a.authInfo
	}
}
