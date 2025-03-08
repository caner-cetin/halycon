package sp_api

import (
	"github.com/caner-cetin/halycon/internal/amazon/catalog/client/catalog"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inbound/client/fba_inbound"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inventory/client/fba_inventory"
	"github.com/caner-cetin/halycon/internal/amazon/listings/client/listings"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"golang.org/x/time/rate"
)

const (
	CatalogServiceName      = "services.catalog"
	ListingsServiceName     = "services.listings"
	FBAInboundServiceName   = "services.fba.inbound"
	FBAInventoryServiceName = "services.fba.inventory"
)

const (
	SearchCatalogItemsRLKey    = "rate_limiter.catalog.searchItems"
	GetCatalogItemsRLKey       = "rate_limiter.catalog.getItems"
	GetListingsItemRLKey       = "rate_limiter.listings.getItem"
	FBAInventorySummariesRLKey = "rate_limiter.fba.inventorySummaries"
	CreateInboundPlanRLKey     = "rate_limiter.fba.createInboundPlan"
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

func (a *Client) SearchCatalogItems(params *catalog.SearchCatalogItemsParams) (*catalog.SearchCatalogItemsOK, error) {
	catalogClient := a.GetCatalogService()
	authOpt := func(op *runtime.ClientOperation) {
		op.AuthInfo = a.authInfo
	}
	return catalogClient.SearchCatalogItems(params, authOpt, WithRateLimit(a.rateLimiters[SearchCatalogItemsRLKey]))
}

func (a *Client) GetCatalogItem(params *catalog.GetCatalogItemParams) (*catalog.GetCatalogItemOK, error) {
	catalogClient := a.GetCatalogService()
	authOpt := func(op *runtime.ClientOperation) {
		op.AuthInfo = a.authInfo
	}
	return catalogClient.GetCatalogItem(params, authOpt, WithRateLimit(a.rateLimiters[GetCatalogItemsRLKey]))
}

func (a *Client) GetListingsItem(params *listings.GetListingsItemParams) (*listings.GetListingsItemOK, error) {
	listingsClient := a.GetListingsService()
	authOpt := func(op *runtime.ClientOperation) {
		op.AuthInfo = a.authInfo
	}
	return listingsClient.GetListingsItem(params, authOpt, WithRateLimit(a.rateLimiters[GetListingsItemRLKey]))
}

func (a *Client) GetFBAInventorySummaries(params *fba_inventory.GetInventorySummariesParams) (*fba_inventory.GetInventorySummariesOK, error) {
	inventoryClient := a.GetFBAInventoryService()
	authOpt := func(op *runtime.ClientOperation) {
		op.AuthInfo = a.authInfo
	}
	return inventoryClient.GetInventorySummaries(params, authOpt, WithRateLimit(a.rateLimiters[FBAInventorySummariesRLKey]))
}

func (a *Client) CreateFBAInboundPlan(params *fba_inbound.CreateInboundPlanParams) (*fba_inbound.CreateInboundPlanAccepted, error) {
	inboundClient := a.GetFBAInboundService()
	authOpt := func(op *runtime.ClientOperation) {
		op.AuthInfo = a.authInfo
	}
	return inboundClient.CreateInboundPlan(params, authOpt, WithRateLimit(a.rateLimiters[CreateInboundPlanRLKey]))
}

func (a *Client) SetRateLimits() {
	a.rateLimiters = map[string]*rate.Limiter{
		SearchCatalogItemsRLKey:    rate.NewLimiter(rate.Limit(2), 2),
		GetCatalogItemsRLKey:       rate.NewLimiter(rate.Limit(2), 2),
		GetListingsItemRLKey:       rate.NewLimiter(rate.Limit(5), 10),
		FBAInventorySummariesRLKey: rate.NewLimiter(rate.Limit(2), 2),
		CreateInboundPlanRLKey:     rate.NewLimiter(rate.Limit(2), 2),
	}
}

func NewAuthorizedClient(token string) *Client {
	authInfo := runtime.ClientAuthInfoWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
		// doesnt work without Authorization header
		if err := r.SetHeaderParam("Authorization", "Bearer "+token); err != nil {
			return err
		}
		// also doesnt work without x-amz-access-token header
		return r.SetHeaderParam("x-amz-access-token", token)
	})

	a := &Client{
		services: map[string]interface{}{},
		authInfo: authInfo,
	}
	a.SetRateLimits()
	return a
}
