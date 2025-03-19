package sp_api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	osRuntime "runtime"

	"github.com/caner-cetin/halycon/internal/amazon/catalog"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inbound"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inventory"
	"github.com/caner-cetin/halycon/internal/amazon/listings"
	"github.com/caner-cetin/halycon/internal/amazon/product_type_definitions"
	"github.com/caner-cetin/halycon/internal/config"
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

	// Token is the access token used for authentication
	Token string

	// TokenManager is the token manager used to acquire and refresh access tokens
	TokenManager *TokenManager

	rlManager *RateLimiterManager
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
func (a *Client) GetCatalogService() *catalog.ClientWithResponses {
	return a.services[CatalogServiceName].(*catalog.ClientWithResponses)
}

// GetListingsService returns the ListingsService implementation from the client's service map.
// GetListingsService returns the listings service implementation from the client's service map.
func (a *Client) GetListingsService() *listings.ClientWithResponses {
	return a.services[ListingsServiceName].(*listings.ClientWithResponses)
}

// GetFBAInboundService returns the FBA inbound service implementation from the client's service map.
func (a *Client) GetFBAInboundService() *fba_inbound.ClientWithResponses {
	return a.services[FBAInboundServiceName].(*fba_inbound.ClientWithResponses)
}

// GetFBAInventoryService returns the FBA Inventory service implementation from the client's service map.
func (a *Client) GetFBAInventoryService() *fba_inventory.ClientWithResponses {
	return a.services[FBAInventoryServiceName].(*fba_inventory.ClientWithResponses)
}

// GetProductTypeDefinitionsService returns the ProductTypeDefinitions service implementation from the client's service map.
func (a *Client) GetProductTypeDefinitionsService() *product_type_definitions.ClientWithResponses {
	return a.services[ProductTypeDefinitionsServiceName].(*product_type_definitions.ClientWithResponses)
}

// SearchCatalogItems searches for catalog items
func (a *Client) SearchCatalogItems(ctx context.Context, params *catalog.SearchCatalogItemsParams) (*catalog.SearchCatalogItemsResp, error) {
	return recordError(a.GetCatalogService().SearchCatalogItemsWithResponse(ctx, params, a.WithAuth(), a.WithRateLimit(SearchCatalogItemsRLKey))) //nolint:typecheck
}

// GetCatalogItem retrieves a catalog item
func (a *Client) GetCatalogItem(ctx context.Context, asin string, params *catalog.GetCatalogItemParams) (*catalog.GetCatalogItemResp, error) {
	return recordError(a.GetCatalogService().GetCatalogItemWithResponse(ctx, asin, params, a.WithAuth(), a.WithRateLimit(GetCatalogItemsRLKey))) //nolint:typecheck
}

// GetListingsItem retrieves a specific listings item.
func (a *Client) GetListingsItem(ctx context.Context, params *listings.GetListingsItemParams, sellerId string, sku string) (*listings.GetListingsItemResp, error) {
	return recordError(a.GetListingsService().GetListingsItemWithResponse(ctx, sellerId, sku, params, a.WithAuth(), a.WithRateLimit(GetListingsItemRLKey))) //nolint:typecheck
}

// PatchListingsItem updates specific fields of a listings item for the given seller ID and SKU
func (a *Client) PatchListingsItem(ctx context.Context, params *listings.PatchListingsItemParams, body listings.PatchListingsItemJSONRequestBody, sellerId string, sku string) (*listings.PatchListingsItemResp, error) {
	return recordError(a.GetListingsService().PatchListingsItemWithResponse(ctx, sellerId, sku, params, body, a.WithAuth(), a.WithRateLimit(PatchListingsItemRLKey))) //nolint:typecheck
}

// DeleteListingsItem removes a listings item from Amazon's catalog
func (a *Client) DeleteListingsItem(ctx context.Context, params *listings.DeleteListingsItemParams, sellerId string, sku string) (*listings.DeleteListingsItemResp, error) {
	return recordError(a.GetListingsService().DeleteListingsItemWithResponse(ctx, sellerId, sku, params, a.WithAuth(), a.WithRateLimit(DeleteListingsItemRLKey))) //nolint:typecheck
}

// GetFBAInventorySummaries retrieves FBA inventory summaries
func (a *Client) GetFBAInventorySummaries(ctx context.Context, params *fba_inventory.GetInventorySummariesParams) (*fba_inventory.GetInventorySummariesResp, error) {
	return recordError(a.GetFBAInventoryService().GetInventorySummariesWithResponse(ctx, params, a.WithAuth(), a.WithRateLimit(FBAInventorySummariesRLKey))) //nolint:typecheck
}

// CreateFBAInboundPlan creates an inbound plan for FBA (Fulfillment by Amazon)
func (a *Client) CreateFBAInboundPlan(ctx context.Context, params fba_inbound.CreateInboundPlanJSONRequestBody) (*fba_inbound.CreateInboundPlanResp, error) {
	return recordError(a.GetFBAInboundService().CreateInboundPlanWithResponse(ctx, params, a.WithAuth(), a.WithRateLimit(CreateInboundPlanRLKey))) //nolint:typecheck
}

// GetInboundOperationStatus retrieves the status of an inbound operation
func (a *Client) GetInboundOperationStatus(ctx context.Context, operation_id string) (*fba_inbound.GetInboundOperationStatusResp, error) {
	return recordError(a.GetFBAInboundService().GetInboundOperationStatusWithResponse(ctx, operation_id, a.WithAuth(), a.WithRateLimit(GetInboundOperationStatusRLKey))) //nolint:typecheck
}

// SearchProductTypeDefinitions searches for product type definitions
func (a *Client) SearchProductTypeDefinitions(ctx context.Context, params *product_type_definitions.SearchDefinitionsProductTypesParams) (*product_type_definitions.SearchDefinitionsProductTypesResp, error) {
	return recordError(a.GetProductTypeDefinitionsService().SearchDefinitionsProductTypesWithResponse(ctx, params, a.WithAuth(), a.WithRateLimit(SearchProductTypeDefinitionsRLKey))) //nolint:typecheck
}

// GetProductTypeDefinition retrieves a product type definition
func (a *Client) GetProductTypeDefinition(ctx context.Context, productType string, params *product_type_definitions.GetDefinitionsProductTypeParams) (*product_type_definitions.GetDefinitionsProductTypeResp, error) {
	return recordError(a.GetProductTypeDefinitionsService().GetDefinitionsProductTypeWithResponse(ctx, productType, params, a.WithAuth(), a.WithRateLimit(GetProductTypeDefinitionRLKey))) //nolint:typecheck
}

// PutListingsItem creates or updates a listing item
func (a *Client) PutListingsItem(ctx context.Context, sellerId string, sku string, params *listings.PutListingsItemParams, body listings.PutListingsItemJSONRequestBody) (*listings.PutListingsItemResp, error) {
	return recordError(a.GetListingsService().PutListingsItemWithResponse(ctx, sellerId, sku, params, body, a.WithAuth(), a.WithRateLimit(CreateListingRLKey))) //nolint:typecheck
}

// rate limiter keys for client's rate limiter mapping, each one of them leads to a rate.Limiter instance.
const (
	SearchCatalogItemsRLKey           = "rate_limiter.catalog.searchItems"
	GetCatalogItemsRLKey              = "rate_limiter.catalog.getItems"
	GetListingsItemRLKey              = "rate_limiter.listings.getItem"
	PatchListingsItemRLKey            = "rate_limiter.listings.patchListings"
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
		PatchListingsItemRLKey:            rate.NewLimiter(rate.Limit(5), 5),
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

	a := &Client{
		services:     map[string]interface{}{},
		TokenManager: tokenManager,
		Token:        token,
	}
	a.SetRateLimits()
	a.rlManager = NewRateLimiterManager(a.rateLimiters)
	return a, nil
}

// WithRateLimit returns a function that applies rate limiting to a client operation.
// It takes a key string that identifies which rate limiter to use from the client's rate limiters map.
// The returned function, when called with a runtime.ClientOperation, will replace its Client with a
// rate-limited HTTP client that enforces the rate limits defined for the given key.
func (a *Client) WithRateLimit(key string) func(ctx context.Context, req *http.Request) error {
	return a.rlManager.RateLimiterInterceptor(key)
}

// WithAuth returns a function that is used to set the authentication information for a client operation.
// This is typically used with the client operation configuration to ensure that the request is authenticated.
func (a *Client) WithAuth() func(ctx context.Context, req *http.Request) error {
	return func(ctx context.Context, r *http.Request) error {
		r.Header.Set("x-amz-access-token", a.Token)
		r.Header.Set("x-amz-date", time.Now().UTC().Format("20060102T150405Z"))
		r.Header.Set("user-agent", fmt.Sprintf("Halycon/0.2 (Language=Go; Platform=%s)", osRuntime.GOOS))
		r.Header.Set("host", fmt.Sprintf("https://%s", config.Config.Amazon.Auth.DefaultClient.APIEndpoint))
		return nil
	}
}
