package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
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
	ProductType           string
	Requirements          string
	AutofillMarketplaceId bool
	AutofillLanguageTag   bool
}

type GetListingConfig struct {
	DisplayAttritubes bool
}

var (
	createListingsCmd = &cobra.Command{
		Use: "create",
		Run: WrapCommandWithResources(createListings, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceListings}}),
	}
	createListingsCfg CreateListingsConfig
	getListingCmd     = &cobra.Command{
		Use: "get",
		Run: WrapCommandWithResources(getListing, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceListings}}),
	}
	getListingCfg    GetListingConfig
	deleteListingCmd = &cobra.Command{
		Use: "delete",
		Run: WrapCommandWithResources(deleteListing, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceListings}}),
	}
	deleteListingCfg listings.DeleteListingsItemParams
	listingsCmd      = &cobra.Command{
		Use: "listings",
	}
	listingOperationSku string
)

func getListingsCmd() *cobra.Command {
	createListingsCmd.PersistentFlags().BoolVar(&createListingsCfg.AutofillMarketplaceId, "fill-marketplace-id", false, "adds {\"marketplace_id\": ...} to every json object in attributes")
	createListingsCmd.PersistentFlags().BoolVar(&createListingsCfg.AutofillLanguageTag, "fill-language-tag", false, "adds {\"language_tag\": ...} to every json object in attributes")
	createListingsCmd.PersistentFlags().StringVarP(&createListingsCfg.Input, "input", "i", "", "Attributes JSON file")
	createListingsCmd.PersistentFlags().StringVarP(&createListingsCfg.ProductType, "type", "p", "", "Attributes JSON file")
	createListingsCmd.PersistentFlags().StringVarP(&createListingsCfg.Requirements, "requirements", "r", "", "Attributes JSON file")
	createListingsCmd.PersistentFlags().StringVar(&createListingsCfg.IssueLocale, "issue-locale", "", "Locale for issue localization. Default: When no locale is provided, the default locale of the first marketplace is used. Localization defaults to en_US when a localized message is not available in the specified locale.")

	getListingCmd.PersistentFlags().BoolVar(&getListingCfg.DisplayAttritubes, "display-attributes", false, "logs listing attributes line by line if given")
	listingsCmd.PersistentFlags().StringVarP(&listingOperationSku, "sku", "s", "", "")
	listingsCmd.AddCommand(createListingsCmd)
	listingsCmd.AddCommand(getListingCmd)
	listingsCmd.AddCommand(deleteListingCmd)
	return listingsCmd
}

func createListings(cmd *cobra.Command, args []string) {
	if createListingsCfg.Input == "" {
		log.Fatal().Msg("input file not given")
	}
	app := GetApp(cmd)
	var params listings.PutListingsItemParams
	params.MarketplaceIds = viper.GetStringSlice(config.AMAZON_MARKETPLACE_ID.Key)
	params.SellerID = strings.TrimSpace(viper.GetString(config.AMAZON_MERCHANT_TOKEN.Key))
	params.Sku = listingOperationSku
	params.IncludedData = []string{"issues"}

	if createListingsCfg.IssueLocale != "" {
		params.IssueLocale = &createListingsCfg.IssueLocale
	}

	var body models.ListingsItemPutRequest
	body.ProductType = ptr.String(createListingsCfg.ProductType)
	body.Requirements = createListingsCfg.Requirements
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
	attr_bytes := internal.ReadFile(createListingsCfg.Input)
	attrs := fastjson.MustParseBytes(attr_bytes).GetObject()
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
	schema := attrs.Get("$schema")
	if schema != nil {
		attrs.Del("$schema")
	}

	var attr_interface interface{}
	err := json.Unmarshal(attrs.MarshalTo(nil), &attr_interface)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	body.Attributes = attr_interface
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
	logListingIssues(result.Payload.Issues)
}

func getListing(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	var params listings.GetListingsItemParams
	params.MarketplaceIds = viper.GetStringSlice(config.AMAZON_MARKETPLACE_ID.Key)
	params.SellerID = viper.GetString(config.AMAZON_MERCHANT_TOKEN.Key)
	params.Sku = listingOperationSku
	params.IncludedData = []string{"summaries", "issues", "offers", "relationships", "attributes"}
	result, err := app.Amazon.Client.GetListingsItem(&params)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	logListingIssues(result.Payload.Issues)
	for _, summary := range result.Payload.Summaries {
		log.Info().
			Str("asin", summary.Asin).
			Str("condition", summary.ConditionType).
			Str("name", summary.ItemName).
			Str("status", strings.Join(summary.Status, ",")).Send()
	}
	for _, relationship := range result.Payload.Relationships {
		for i, rls := range relationship.Relationships {
			ev := log.Info().
				Str("type", *rls.Type).
				Str("child_skus", strings.Join(rls.ChildSkus, ",")).
				Str("parent_skus", strings.Join(rls.ParentSkus, ","))
			if rls.VariationTheme != nil {
				ev.Str("theme", *rls.VariationTheme.Theme).
					Str("theme_attributes", strings.Join(rls.VariationTheme.Attributes, ","))
			}
			ev.Msgf("relationship %d", i+1)

		}
	}
	if len(result.Payload.Offers) == 0 {
		log.Warn().Msg("no offers found")
	} else {
		for i, offer := range result.Payload.Offers {
			ev := log.Info().
				Str("type", *offer.OfferType)
			if offer.Points != nil {
				ev.Int64("points", *offer.Points.PointsNumber)
			}
			if offer.Audience != nil {
				ev.Str("audience_name", offer.Audience.DisplayName).
					Str("audience_value", offer.Audience.Value)
			}
			if offer.Price != nil {
				ev.Str("currency_code", *offer.Price.CurrencyCode).
					Str("price", string(*offer.Price.Amount))
			}
			ev.Msgf("offer %d", i+1)
		}
	}
	if getListingCfg.DisplayAttritubes {
		attrs_bytes, err := json.Marshal(result.Payload.Attributes)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		attrs := fastjson.MustParseBytes(attrs_bytes)
		attrs.GetObject().Visit(func(key []byte, v *fastjson.Value) {
			ev := log.Info()
			for _, obj := range v.GetArray() {
				obj.GetObject().Visit(func(key []byte, v *fastjson.Value) {
					if v.Type() == fastjson.TypeString {
						marshalled, err := strconv.Unquote(string(v.MarshalTo(nil)))
						if err != nil {
							log.Fatal().Err(err).Send()
						}
						ev.Str(string(key), marshalled)
					} else {
						marshalled := v.MarshalTo(nil)
						ev.Bytes(string(key), marshalled)
					}
				})
			}
			ev.Msg(string(key))
		})
	}
}

func deleteListing(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	var params = deleteListingCfg
	params.Sku = listingOperationSku
	params.MarketplaceIds = viper.GetStringSlice(config.AMAZON_MARKETPLACE_ID.Key)
	params.SellerID = viper.GetString(config.AMAZON_MERCHANT_TOKEN.Key)
	result, err := app.Amazon.Client.DeleteListingsItem(&params)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	log.Info().
		Str("status", *result.Payload.Status).
		Str("sku", *result.Payload.Sku).
		Str("submission_id", *result.Payload.SubmissionID).
		Send()
	logListingIssues(result.Payload.Issues)
}

func logListingIssues(issues []*models.Issue) {
	for _, issue := range issues {
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
