package cmd

import (
	"context"

	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/catalog/client/catalog"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inbound/client/fba_inbound"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inventory/client/fba_inventory"
	"github.com/caner-cetin/halycon/internal/amazon/listings/client/listings"
	"github.com/caner-cetin/halycon/internal/amazon/product_type_definitions/client/definitions"
	"github.com/caner-cetin/halycon/internal/config"
	sp_api "github.com/caner-cetin/halycon/internal/sp-api"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
)

// ResourceConfig defines the configuration structure for resources and services.
// It contains lists of resource types and service types that are to be managed.
type ResourceConfig struct {
	Resources []ResourceType
	Services  []ServiceType
}

// AppCtx represents the application context structure.
// It encapsulates all the necessary dependencies and configurations
// required for the application to function properly.
// This includes clients for external services such as Amazon's Selling Partner API.
type AppCtx struct {
	Amazon struct {
		Client *sp_api.Client
		Token  string
	}
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
				if !viper.IsSet(config.AMAZON_AUTH_CLIENT_ID.Key) {
					log.Error().Msg("amazon client ID is not set")
					return
				}
				if !viper.IsSet(config.AMAZON_AUTH_CLIENT_SECRET.Key) {
					log.Error().Msg("amazon client secret is not set")
					return
				}
				if !viper.IsSet(config.AMAZON_AUTH_REFRESH_TOKEN.Key) {
					log.Error().Msg("amazon refresh token is not set")
					return
				}
				var auth = sp_api.AuthConfig{
					ClientID:     viper.GetString(config.AMAZON_AUTH_CLIENT_ID.Key),
					ClientSecret: viper.GetString(config.AMAZON_AUTH_CLIENT_SECRET.Key),
					RefreshToken: viper.GetString(config.AMAZON_AUTH_REFRESH_TOKEN.Key),
					Endpoint:     viper.GetString(config.AMAZON_AUTH_ENDPOINT.Key),
				}
				var tokenManager = sp_api.NewTokenManager(auth)
				token, err := tokenManager.GetAccessToken()
				if err != nil {
					log.Error().Err(err).Msg("error while acquiring access token")
					return
				}
				log.Debug().Str("token_prefix", token[:10]+"...").Msg("acquired access token")
				host := viper.GetString(config.AMAZON_AUTH_HOST.Key)
				basePath := "/"
				scheme := "https"
				app.Amazon.Client = sp_api.NewAuthorizedClient(token)
				for _, service := range resourceConfig.Services {
					switch service {
					case ServiceCatalog:
						app.Amazon.Client.AddService(sp_api.CatalogServiceName, catalog.NewClientWithBearerToken(host, basePath, scheme, token))
					case ServiceListings:
						app.Amazon.Client.AddService(sp_api.ListingsServiceName, listings.NewClientWithBearerToken(host, basePath, scheme, token))
					case ServiceFBAInbound:
						app.Amazon.Client.AddService(sp_api.FBAInboundServiceName, fba_inbound.NewClientWithBearerToken(host, basePath, scheme, token))
					case ServiceFBAInventory:
						app.Amazon.Client.AddService(sp_api.FBAInventoryServiceName, fba_inventory.NewClientWithBearerToken(host, basePath, scheme, token))
					case ServiceProductTypeDefinitions:
						app.Amazon.Client.AddService(sp_api.ProductTypeDefinitionsServiceName, definitions.NewClientWithBearerToken(host, basePath, scheme, token))
					}
				}
				app.Amazon.Token = token

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
