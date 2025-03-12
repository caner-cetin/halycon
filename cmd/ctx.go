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

type ResourceType int

const (
	ResourceAmazon ResourceType = iota
)

type ServiceType int

const (
	ServiceCatalog ServiceType = iota
	ServiceListings
	ServiceFBAInbound
	ServiceFBAInventory
	ServiceProductTypeDefinitions
)

type ResourceConfig struct {
	Resources []ResourceType
	Services  []ServiceType
}

type AppCtx struct {
	Amazon struct {
		Client *sp_api.Client
		Token  string
	}
}

func WrapCommandWithResources(fn func(cmd *cobra.Command, args []string), resourceConfig ResourceConfig) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		app := AppCtx{}
		for _, resource := range resourceConfig.Resources {
			switch resource {
			case ResourceAmazon:
				if !viper.IsSet(config.AMAZON_AUTH_CLIENT_ID.Key) {
					log.Fatal().Msg("amazon client ID is not set")
				}
				if !viper.IsSet(config.AMAZON_AUTH_CLIENT_SECRET.Key) {
					log.Fatal().Msg("amazon client secret is not set")
				}
				if !viper.IsSet(config.AMAZON_AUTH_REFRESH_TOKEN.Key) {
					log.Fatal().Msg("amazon refresh token is not set")
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
					log.Fatal().Err(err).Msg("error while acquiring access token")
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
