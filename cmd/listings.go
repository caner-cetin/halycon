package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/listings/client/listings"
	"github.com/caner-cetin/halycon/internal/amazon/listings/models"
	"github.com/caner-cetin/halycon/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/valyala/fastjson"
)

type CreateListingsConfig struct {
	IssueLocale           string
	Input                 string
	AutofillMarketplaceId bool
	AutofillLanguageTag   bool
}

type CreateListingsInput struct {
	ProductType  string      `json:"productType"`
	Requirements string      `json:"requirements"`
	Attributes   interface{} `json:"attributes"`
}

var (
	createListingsCmd = &cobra.Command{
		Use: "create",
		Run: WrapCommandWithResources(createListings, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceListings}}),
	}
	getListingCmd = &cobra.Command{
		Use: "get",
		Run: WrapCommandWithResources(getListing, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceListings}}),
	}
	createListingsCfg CreateListingsConfig
	listingsCmd       = &cobra.Command{
		Use: "listings",
	}
)

func getListingsCmd() *cobra.Command {
	createListingsCmd.PersistentFlags().BoolVar(&createListingsCfg.AutofillMarketplaceId, "fill-marketplace-id", false, "adds {\"marketplace_id\": ...} to every json object in attributes")
	createListingsCmd.PersistentFlags().BoolVar(&createListingsCfg.AutofillLanguageTag, "fill-language-tag", false, "adds {\"language_tag\": ...} to every json object in attributes")
	createListingsCmd.PersistentFlags().StringVarP(&createListingsCfg.Input, "input", "i", "", "Input JSON file")
	createListingsCmd.PersistentFlags().StringVar(&createListingsCfg.IssueLocale, "issue-locale", "", "Locale for issue localization. Default: When no locale is provided, the default locale of the first marketplace is used. Localization defaults to en_US when a localized message is not available in the specified locale.")
	listingsCmd.AddCommand(createListingsCmd)
	listingsCmd.AddCommand(getListingCmd)
	return listingsCmd
}

func createListings(cmd *cobra.Command, args []string) {
	if createListingsCfg.Input == "" {
		log.Fatal().Msg("input file not given")
	}
	app := GetApp(cmd)

	listing_bytes := internal.ReadFile(createListingsCfg.Input)
	var listing CreateListingsInput
	err := json.Unmarshal(listing_bytes, &listing)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	var params listings.PutListingsItemParams
	params.MarketplaceIds = viper.GetStringSlice(config.AMAZON_MARKETPLACE_ID.Key)
	params.SellerID = strings.TrimSpace(viper.GetString(config.AMAZON_MERCHANT_TOKEN.Key))
	params.Sku = "W9-EYD8-3OOO"
	params.IncludedData = []string{"issues"}

	if createListingsCfg.IssueLocale != "" {
		params.IssueLocale = &createListingsCfg.IssueLocale
	}

	var body models.ListingsItemPutRequest
	body.ProductType = ptr.String(listing.ProductType)
	body.Requirements = listing.Requirements
	var marketplace_id *fastjson.Value
	var language_tag *fastjson.Value
	if createListingsCfg.AutofillMarketplaceId {
		marketplace_id = fastjson.MustParse(fmt.Sprintf(`"%s"`, viper.GetStringSlice(config.AMAZON_MARKETPLACE_ID.Key)[0]))
	}
	if createListingsCfg.AutofillLanguageTag {
		language_tag = fastjson.MustParse(fmt.Sprintf(`"%s"`, viper.GetString(config.AMAZON_DEFAULT_LANGUAGE_TAG.Key)))
	}
	var should_fill_marketplace_id = marketplace_id != nil
	var should_fill_language_tag = language_tag != nil
	attrs := fastjson.MustParseBytes(listing_bytes).GetObject("attributes")
	attrs.Visit(func(key []byte, v *fastjson.Value) {
		for _, attr_detail := range v.GetArray() {
			attr_detail_obj := attr_detail.GetObject()
			if attr_detail_obj == nil {
				continue
			}
			if should_fill_marketplace_id {
				mid := attr_detail_obj.Get("marketplace_id")
				if mid == nil {
					attr_detail_obj.Set("marketplace_id", marketplace_id)
				}
			}
			if should_fill_language_tag {
				ltg := attr_detail_obj.Get("language_tag")
				if ltg == nil {
					attr_detail_obj.Set("language_tag", language_tag)
				}
			}
		}
	})
	body.Attributes = listing.Attributes
	params.Body = &body

	result, err := app.Amazon.Client.CreateListing(&params)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	log.Info().
		Str("status", *result.Payload.Status).
		Str("submission_id", *result.Payload.SubmissionID).
		Str("sku", *result.Payload.Sku).
		Send()
	for _, issue := range result.Payload.Issues {
		ev := log.Warn().
			Str("message", *issue.Message).
			Str("code", *issue.Code).
			Str("severity", *issue.Severity).
			Str("attribute", strings.Join(issue.AttributeNames, ","))
		if issue.Enforcements != nil && issue.Enforcements.Exemption != nil {
			ev.Interface("exemption_expiry_date", issue.Enforcements.Exemption.ExpiryDate)
			ev.Str("exemption_status", *issue.Enforcements.Exemption.Status)
			for i, action := range issue.Enforcements.Actions {
				ev.Str(fmt.Sprintf("action %d", i), *action.Action)
			}
		}
		ev.Send()
	}
}

func getListing(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	var params listings.GetListingsItemParams
	params.MarketplaceIds = viper.GetStringSlice(config.AMAZON_MARKETPLACE_ID.Key)
	params.SellerID = viper.GetString(config.AMAZON_MERCHANT_TOKEN.Key)
	params.Sku = "W9-EYD8-3OGI"
	result, err := app.Amazon.Client.GetListingsItem(&params)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	fmt.Println(result.Payload.Issues)
}
