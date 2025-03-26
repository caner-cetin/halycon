package sp_api

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/catalog"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inbound"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inventory"
	"github.com/caner-cetin/halycon/internal/amazon/feeds"
	"github.com/caner-cetin/halycon/internal/amazon/listings"
	"github.com/caner-cetin/halycon/internal/amazon/product_type_definitions"
	"github.com/caner-cetin/halycon/internal/config"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

const (
	CatalogServiceName                = "services.catalog"
	ListingsServiceName               = "services.listings"
	FBAInboundServiceName             = "services.fba.inbound"
	FBAInventoryServiceName           = "services.fba.inventory"
	ProductTypeDefinitionsServiceName = "services.listings.product_type.definitions"
	FeedsServiceName                  = "services.feeds"
)

type Client struct {
	services     map[string]interface{}
	rateLimiters map[string]*rate.Limiter
	Token        string
	TokenManager *TokenManager
	rlManager    *RateLimiterManager
}

func (a *Client) AddService(name string, service interface{}) {
	a.services[name] = service
}

func (a *Client) GetService(name string) interface{} {
	return a.services[name]
}

func (a *Client) GetCatalogService() *catalog.ClientWithResponses {
	return a.services[CatalogServiceName].(*catalog.ClientWithResponses)
}

func (a *Client) GetListingsService() *listings.ClientWithResponses {
	return a.services[ListingsServiceName].(*listings.ClientWithResponses)
}

func (a *Client) GetFBAInboundService() *fba_inbound.ClientWithResponses {
	return a.services[FBAInboundServiceName].(*fba_inbound.ClientWithResponses)
}

func (a *Client) GetFBAInventoryService() *fba_inventory.ClientWithResponses {
	return a.services[FBAInventoryServiceName].(*fba_inventory.ClientWithResponses)
}

func (a *Client) GetProductTypeDefinitionsService() *product_type_definitions.ClientWithResponses {
	return a.services[ProductTypeDefinitionsServiceName].(*product_type_definitions.ClientWithResponses)
}

func (a *Client) GetFeedsService() *feeds.ClientWithResponses {
	return a.services[FeedsServiceName].(*feeds.ClientWithResponses)
}

func (a *Client) SearchCatalogItems(ctx context.Context, params *catalog.SearchCatalogItemsParams) (*catalog.SearchCatalogItemsResp, error) {
	return recordError(a.GetCatalogService().SearchCatalogItemsWithResponse(ctx, params, a.WithAuth(), a.WithRateLimit(SearchCatalogItemsRLKey))) //nolint:typecheck
}

func (a *Client) GetCatalogItem(ctx context.Context, asin string, params *catalog.GetCatalogItemParams) (*catalog.GetCatalogItemResp, error) {
	return recordError(a.GetCatalogService().GetCatalogItemWithResponse(ctx, asin, params, a.WithAuth(), a.WithRateLimit(GetCatalogItemsRLKey))) //nolint:typecheck
}

func (a *Client) GetListingsItem(ctx context.Context, params *listings.GetListingsItemParams, sellerId string, sku string) (*listings.GetListingsItemResp, error) {
	return recordError(a.GetListingsService().GetListingsItemWithResponse(ctx, sellerId, sku, params, a.WithAuth(), a.WithRateLimit(GetListingsItemRLKey))) //nolint:typecheck
}

func (a *Client) PatchListingsItem(ctx context.Context, params *listings.PatchListingsItemParams, body listings.PatchListingsItemJSONRequestBody, sellerId string, sku string) (*listings.PatchListingsItemResp, error) {
	return recordError(a.GetListingsService().PatchListingsItemWithResponse(ctx, sellerId, sku, params, body, a.WithAuth(), a.WithRateLimit(PatchListingsItemRLKey))) //nolint:typecheck
}

func (a *Client) DeleteListingsItem(ctx context.Context, params *listings.DeleteListingsItemParams, sellerId string, sku string) (*listings.DeleteListingsItemResp, error) {
	return recordError(a.GetListingsService().DeleteListingsItemWithResponse(ctx, sellerId, sku, params, a.WithAuth(), a.WithRateLimit(DeleteListingsItemRLKey))) //nolint:typecheck
}

func (a *Client) GetFBAInventorySummaries(ctx context.Context, params *fba_inventory.GetInventorySummariesParams) (*fba_inventory.GetInventorySummariesResp, error) {
	return recordError(a.GetFBAInventoryService().GetInventorySummariesWithResponse(ctx, params, a.WithAuth(), a.WithRateLimit(FBAInventorySummariesRLKey))) //nolint:typecheck
}

func (a *Client) CreateFBAInboundPlan(ctx context.Context, params fba_inbound.CreateInboundPlanJSONRequestBody) (*fba_inbound.CreateInboundPlanResp, error) {
	return recordError(a.GetFBAInboundService().CreateInboundPlanWithResponse(ctx, params, a.WithAuth(), a.WithRateLimit(CreateInboundPlanRLKey))) //nolint:typecheck
}

func (a *Client) GetInboundOperationStatus(ctx context.Context, operation_id string) (*fba_inbound.GetInboundOperationStatusResp, error) {
	return recordError(a.GetFBAInboundService().GetInboundOperationStatusWithResponse(ctx, operation_id, a.WithAuth(), a.WithRateLimit(GetInboundOperationStatusRLKey))) //nolint:typecheck
}

func (a *Client) SearchProductTypeDefinitions(ctx context.Context, params *product_type_definitions.SearchDefinitionsProductTypesParams) (*product_type_definitions.SearchDefinitionsProductTypesResp, error) {
	return recordError(a.GetProductTypeDefinitionsService().SearchDefinitionsProductTypesWithResponse(ctx, params, a.WithAuth(), a.WithRateLimit(SearchProductTypeDefinitionsRLKey))) //nolint:typecheck
}

func (a *Client) GetProductTypeDefinition(ctx context.Context, productType string, params *product_type_definitions.GetDefinitionsProductTypeParams) (*product_type_definitions.GetDefinitionsProductTypeResp, error) {
	return recordError(a.GetProductTypeDefinitionsService().GetDefinitionsProductTypeWithResponse(ctx, productType, params, a.WithAuth(), a.WithRateLimit(GetProductTypeDefinitionRLKey))) //nolint:typecheck
}

func (a *Client) PutListingsItem(ctx context.Context, sellerId string, sku string, params *listings.PutListingsItemParams, body listings.PutListingsItemJSONRequestBody) (*listings.PutListingsItemResp, error) {
	return recordError(a.GetListingsService().PutListingsItemWithResponse(ctx, sellerId, sku, params, body, a.WithAuth(), a.WithRateLimit(CreateListingRLKey))) //nolint:typecheck
}

func (a *Client) GetFeeds(ctx context.Context, params *feeds.GetFeedsParams) (*feeds.GetFeedsResp, error) {
	return recordError(a.GetFeedsService().GetFeedsWithResponse(ctx, params, a.WithAuth(), a.WithRateLimit(GetFeedsRLKey))) //nolint: typecheck
}

func (a *Client) CreateFeedDocument(ctx context.Context, contentType feeds.CreateFeedDocumentJSONRequestBody) (*feeds.CreateFeedDocumentResp, error) {
	return recordError(a.GetFeedsService().CreateFeedDocumentWithResponse(ctx, contentType, a.WithAuth(), a.WithRateLimit(CreateFeedDocumentRLKey))) //nolint: typecheck
}

func (a *Client) CreateFeed(ctx context.Context, body feeds.CreateFeedJSONRequestBody) (*feeds.CreateFeedResp, error) {
	return recordError(a.GetFeedsService().CreateFeedWithResponse(ctx, body, a.WithAuth(), a.WithRateLimit(CreateFeedRLKey))) //nolint: typecheck
}

func (a *Client) GetFeed(ctx context.Context, id string) (*feeds.GetFeedResp, error) {
	return recordError(a.GetFeedsService().GetFeedWithResponse(ctx, id, a.WithAuth(), a.WithRateLimit(GetFeedRLKey))) //nolint: typecheck
}

func (a *Client) GetFeedDocument(ctx context.Context, id string) (*feeds.GetFeedDocumentResp, error) {
	return recordError(a.GetFeedsService().GetFeedDocumentWithResponse(ctx, id, a.WithAuth(), a.WithRateLimit(GetFeedDocumentRLKey))) //nolint: typecheck
}

const (
	SearchCatalogItemsRLKey           = "catalog.searchItems"
	GetCatalogItemsRLKey              = "catalog.getItems"
	GetListingsItemRLKey              = "listings.getItem"
	PatchListingsItemRLKey            = "listings.patchListings"
	DeleteListingsItemRLKey           = "listings.deleteListingsItem"
	CreateListingRLKey                = "listings.createItem"
	FBAInventorySummariesRLKey        = "fba.inventorySummaries"
	CreateInboundPlanRLKey            = "fba.createInboundPlan"
	GetInboundOperationStatusRLKey    = "fba.getInboundOperationStatus"
	SearchProductTypeDefinitionsRLKey = "listings.search_product_type_definitions"
	GetProductTypeDefinitionRLKey     = "listings.get_product_type_definitions"
	GetFeedsRLKey                     = "feeds.getFeeds"
	CreateFeedDocumentRLKey           = "feeds.createFeedDocument"
	CreateFeedRLKey                   = "feeds.createFeed"
	GetFeedRLKey                      = "feeds.getFeed"
	GetFeedDocumentRLKey              = "feeds.getFeedDocument"
)

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
		GetFeedsRLKey:                     rate.NewLimiter(rate.Limit(0.0222), 10),
		CreateFeedDocumentRLKey:           rate.NewLimiter(rate.Limit(0.5), 15),
		CreateFeedRLKey:                   rate.NewLimiter(rate.Limit(0.0083), 15),
		GetFeedRLKey:                      rate.NewLimiter(rate.Limit(2), 15),
		GetFeedDocumentRLKey:              rate.NewLimiter(rate.Limit(0.0222), 10),
	}
}

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

func (a *Client) WithRateLimit(key string) func(ctx context.Context, req *http.Request) error {
	return a.rlManager.RateLimiterInterceptor(key)
}

func (a *Client) WithAuth() func(ctx context.Context, req *http.Request) error {
	return func(ctx context.Context, r *http.Request) error {
		r.Header.Set("x-amz-access-token", a.Token)
		r.Header.Set("x-amz-date", time.Now().UTC().Format("20060102T150405Z"))
		r.Header.Set("user-agent", fmt.Sprintf("Halycon/%s (Language=Go; Platform=%s)", internal.Version, runtime.GOOS))
		r.Header.Set("host", fmt.Sprintf("https://%s", config.Config.Amazon.Auth.DefaultClient.APIEndpoint))
		return nil
	}
}
