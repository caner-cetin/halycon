package cmd

import (
	"strings"

	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/catalog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type getCatalogItemConfig struct {
	Asin   string
	Locale string
}

var (
	getCatalogItemCmd = &cobra.Command{
		Use: "get",
		Run: WrapCommandWithResources(getCatalogItem, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceCatalog}}),
	}
	getCatalogItemCfg getCatalogItemConfig

	catalogCmd = &cobra.Command{
		Use: "catalog",
	}
)

func getCatalogCmd() *cobra.Command {
	flags := getCatalogItemCmd.PersistentFlags()
	flags.StringVar(&getCatalogItemCfg.Asin, "asin", "", "")
	flags.StringVar(&getCatalogItemCfg.Locale, "locale", "", "Locale for retrieving localized summaries. Defaults to the primary locale of the marketplace.")
	getCatalogItemCmd.MarkFlagRequired("asin")
	catalogCmd.AddCommand(getCatalogItemCmd)
	return catalogCmd
}

func getCatalogItem(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	var params catalog.GetCatalogItemParams
	params.Locale = &getCatalogItemCfg.Locale
	params.MarketplaceIds = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID
	params.IncludedData = &[]catalog.GetCatalogItemParamsIncludedData{"identifiers", "summaries", "attributes", "relationships"}
	status, err := app.Amazon.Client.GetCatalogItem(cmd.Context(), getCatalogItemCfg.Asin, &params)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	result := status.JSON200
	internal.DisplayInterface(result.Attributes)
	for _, relationship := range *result.Relationships {
		for i, rls := range relationship.Relationships {
			ev := log.Info().
				Str("type", string(rls.Type))
			if rls.ChildAsins != nil {
				ev.Str("child_asins", strings.Join(*rls.ChildAsins, ","))
			}
			if rls.ParentAsins != nil {
				ev.Str("parent_asins", strings.Join(*rls.ParentAsins, ","))
			}
			if rls.VariationTheme != nil {
				ev.Str("theme", *rls.VariationTheme.Theme).
					Str("theme_attributes", strings.Join(*rls.VariationTheme.Attributes, ","))
			}
			ev.Msgf("relationship %d", i+1)

		}
	}
}
