package cmd

import (
	"context"
	"fmt"

	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/catalog"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inbound"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inventory"
	"github.com/caner-cetin/halycon/internal/amazon/feeds"
	"github.com/caner-cetin/halycon/internal/amazon/listings"
	"github.com/caner-cetin/halycon/internal/amazon/product_type_definitions"
	sp_api "github.com/caner-cetin/halycon/internal/sp-api"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// ResourceType represents the type of a resource in the system.
// It is used to categorize and differentiate between different kinds of resources
// that can be managed by the application.
type ResourceType int

const (
	// ResourceAmazon represents the Amazon SP-API.
	ResourceAmazon ResourceType = iota
)

// ServiceType represents a type of service in the system.
// It is an enumeration of different service types that can be used within the application.
type ServiceType int

// ServiceCatalog represents the Selling Partner API service for catalog-related operations.
const (
	ServiceCatalog ServiceType = iota
	ServiceListings
	ServiceFBAInbound
	ServiceFBAInventory
	ServiceProductTypeDefinitions
	ServiceFeeds
)

// ResourceConfig defines the configuration structure for resources and services.
// It contains lists of resource types and service types that are to be managed.
type ResourceConfig struct {
	Resources []ResourceType
	Services  []ServiceType
}

// Amazon represents the Amazon API client configuration.
// It holds the SP-API client and authentication token for interacting with Amazon's Selling Partner API.
type Amazon struct {
	Client *sp_api.Client
	Token  string
}

// AppCtx represents the application context structure.
// It encapsulates all the necessary dependencies and configurations
// required for the application to function properly.
// This includes clients for external services such as Amazon's Selling Partner API.
type AppCtx struct {
	Amazon Amazon
}

// WrapCommandWithResources wraps a Cobra command function with resource initialization logic.
// It takes a command function and resource configuration, then returns a new function that:
// 1. Initializes required resources (currently supports Amazon SP-API)
// 2. Sets up authentication and services based on the provided configuration
// 3. Creates an application context with initialized resources
// 4. Injects the context into the command before executing the original function
//
// The wrapper will exit early if required Amazon credentials are not set in the configuration.
func WrapCommandWithResources(fn func(cmd *cobra.Command, args []string), resourceConfig ResourceConfig) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		app := AppCtx{}
		for _, resource := range resourceConfig.Resources {
			if resource == ResourceAmazon {
				var err error
				app.Amazon.Client, err = sp_api.NewAuthorizedClient()
				if err != nil {
					log.Error().Err(err).Msg("failed to create authorized client")
					return
				}
				host := cfg.Amazon.Auth.DefaultClient.APIEndpoint
				basePath := "/"
				scheme := "https"
				server := fmt.Sprintf("%s://%s%s", scheme, host, basePath)
				for _, service := range resourceConfig.Services {
					switch service {
					case ServiceCatalog:
						client, err := catalog.NewClientWithResponses(server)
						if err != nil {
							log.Error().Err(err).Msg("failed to create catalog client")
							return
						}
						app.Amazon.Client.AddService(sp_api.CatalogServiceName, client)
					case ServiceListings:
						client, err := listings.NewClientWithResponses(server)
						if err != nil {
							log.Error().Err(err).Msg("failed to create listings client")
							return
						}
						app.Amazon.Client.AddService(sp_api.ListingsServiceName, client)
					case ServiceFBAInbound:
						client, err := fba_inbound.NewClientWithResponses(server)
						if err != nil {
							log.Error().Err(err).Msg("failed to create fba inbound client")
							return
						}
						app.Amazon.Client.AddService(sp_api.FBAInboundServiceName, client)
					case ServiceFBAInventory:
						client, err := fba_inventory.NewClientWithResponses(server)
						if err != nil {
							log.Error().Err(err).Msg("failed to create fba inventory client")
							return
						}
						app.Amazon.Client.AddService(sp_api.FBAInventoryServiceName, client)
					case ServiceProductTypeDefinitions:
						client, err := product_type_definitions.NewClientWithResponses(server)
						if err != nil {
							log.Error().Err(err).Msg("failed to create product type definitions client")
							return
						}
						app.Amazon.Client.AddService(sp_api.ProductTypeDefinitionsServiceName, client)
					case ServiceFeeds:
						client, err := feeds.NewClientWithResponses(server)
						if err != nil {
							log.Error().Err(err).Msg("failed to create product type definitions client")
							return
						}
						app.Amazon.Client.AddService(sp_api.FeedsServiceName, client)
					}
				}

			}
		}
		cmd.SetContext(context.WithValue(cmd.Context(), internal.APP_CONTEXT, app))
		fn(cmd, args)
	}
}

// GetApp retrieves the application context (AppCtx) from a Cobra command's context.
// It expects the context to contain an AppCtx value stored with internal.APP_CONTEXT key.
// The function performs a type assertion to convert the context value to AppCtx.
//
// Panics if the context value cannot be type-asserted to AppCtx
func GetApp(cmd *cobra.Command) AppCtx {
	return cmd.Context().Value(internal.APP_CONTEXT).(AppCtx)
}
