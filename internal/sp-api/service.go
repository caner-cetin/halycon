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
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

const (
	CatalogServiceName                = "services.catalog"
	ListingsServiceName               = "services.listings"
	FBAInboundServiceName             = "services.fba.inbound"
	FBAInventoryServiceName           = "services.fba.inventory"
	ProductTypeDefinitionsServiceName = "services.listings.product_type.definitions"
)

type Client struct {
	services     map[string]interface{}
	rateLimiters map[string]*rate.Limiter
	authInfo     runtime.ClientAuthInfoWriter
}

func (a *Client) AddService(name string, service interface{}) {
	a.services[name] = service
}

func (a *Client) GetService(name string) interface{} {
	return a.services[name]
}

func (a *Client) GetCatalogService() catalog.ClientService {
	return a.services[CatalogServiceName].(catalog.ClientService)
}

func (a *Client) GetListingsService() listings.ClientService {
	return a.services[ListingsServiceName].(listings.ClientService)
}

func (a *Client) GetFBAInboundService() fba_inbound.ClientService {
	return a.services[FBAInboundServiceName].(fba_inbound.ClientService)
}

func (a *Client) GetFBAInventoryService() fba_inventory.ClientService {
	return a.services[FBAInventoryServiceName].(fba_inventory.ClientService)
}

func (a *Client) GetProductTypeDefinitionsService() definitions.ClientService {
	return a.services[ProductTypeDefinitionsServiceName].(definitions.ClientService)
}
func (a *Client) SearchCatalogItems(params *catalog.SearchCatalogItemsParams) (*catalog.SearchCatalogItemsOK, error) {
	return a.GetCatalogService().SearchCatalogItems(params, a.WithAuth(), a.WithRateLimit(SearchCatalogItemsRLKey))
}

func (a *Client) GetCatalogItem(params *catalog.GetCatalogItemParams) (*catalog.GetCatalogItemOK, error) {
	return a.GetCatalogService().GetCatalogItem(params, a.WithAuth(), a.WithRateLimit(GetCatalogItemsRLKey))
}

func (a *Client) GetListingsItem(params *listings.GetListingsItemParams) (*listings.GetListingsItemOK, error) {
	return a.GetListingsService().GetListingsItem(params, a.WithAuth(), a.WithRateLimit(GetListingsItemRLKey))
}

func (a *Client) DeleteListingsItem(params *listings.DeleteListingsItemParams) (*listings.DeleteListingsItemOK, error) {
	return a.GetListingsService().DeleteListingsItem(params, a.WithAuth(), a.WithRateLimit(DeleteListingsItemRLKey))
}

func (a *Client) GetFBAInventorySummaries(params *fba_inventory.GetInventorySummariesParams) (*fba_inventory.GetInventorySummariesOK, error) {
	return a.GetFBAInventoryService().GetInventorySummaries(params, a.WithAuth(), a.WithRateLimit(FBAInventorySummariesRLKey))
}

func (a *Client) CreateFBAInboundPlan(params *fba_inbound.CreateInboundPlanParams) (*fba_inbound.CreateInboundPlanAccepted, error) {
	return a.GetFBAInboundService().CreateInboundPlan(params, a.WithAuth(), a.WithRateLimit(CreateInboundPlanRLKey))
}

func (a *Client) GetInboundOperationStatus(params *fba_inbound.GetInboundOperationStatusParams) (*fba_inbound.GetInboundOperationStatusOK, error) {
	return a.GetFBAInboundService().GetInboundOperationStatus(params, a.WithAuth(), a.WithRateLimit(GetInboundOperationStatusRLKey))
}

func (a *Client) SearchProductTypeDefinitions(params *definitions.SearchDefinitionsProductTypesParams) (*definitions.SearchDefinitionsProductTypesOK, error) {
	return a.GetProductTypeDefinitionsService().SearchDefinitionsProductTypes(params, a.WithAuth(), a.WithRateLimit(SearchProductTypeDefinitionsRLKey))
}

func (a *Client) GetProductTypeDefinition(params *definitions.GetDefinitionsProductTypeParams) (*definitions.GetDefinitionsProductTypeOK, error) {
	return a.GetProductTypeDefinitionsService().GetDefinitionsProductType(params, a.WithAuth(), a.WithRateLimit(GetProductTypeDefinitionRLKey))
}

func (a *Client) CreateListing(params *listings.PutListingsItemParams) (*listings.PutListingsItemOK, error) {
	return a.GetListingsService().PutListingsItem(params, a.WithAuth(), a.WithRateLimit(CreateListingRLKey))
}

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

func NewAuthorizedClient(token string) *Client {
	authInfo := runtime.ClientAuthInfoWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
		// doesnt work without Authorization header
		if err := r.SetHeaderParam("Authorization", "Bearer "+token); err != nil {
			return err
		}
		// also doesnt work without x-amz-access-token header
		if err := r.SetHeaderParam("x-amz-access-token", token); err != nil {
			return err
		}
		// also doesnt work without X-Amz-Access-Token header
		if err := r.SetHeaderParam("X-Amz-Access-Token", token); err != nil {
			return err
		}
		if err := r.SetHeaderParam("host", fmt.Sprintf("https://%s", viper.GetString(config.AMAZON_AUTH_ENDPOINT.Key))); err != nil {
			return err
		}
		if err := r.SetHeaderParam("x-amz-date", time.Now().UTC().Format("20060102T150405Z")); err != nil {
			return err
		}
		if err := r.SetHeaderParam("user-agent", fmt.Sprintf("Halycon/1.0 (Language=Go; Platform=%s)", osRuntime.GOOS)); err != nil {
			return err
		}
		return nil
	})

	a := &Client{
		services: map[string]interface{}{},
		authInfo: authInfo,
	}
	a.SetRateLimits()
	return a
}

func (a *Client) WithRateLimit(key string) func(op *runtime.ClientOperation) {
	return func(op *runtime.ClientOperation) {
		httpClient := NewRateLimitedClient(a.rateLimiters[key])
		op.Client = httpClient
	}
}

func (a *Client) WithAuth() func(op *runtime.ClientOperation) {
	return func(op *runtime.ClientOperation) {
		op.AuthInfo = a.authInfo
	}
}
