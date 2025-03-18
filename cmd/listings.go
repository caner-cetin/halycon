package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/listings"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/valyala/fastjson"
)

type createListingsConfig struct {
	IssueLocale           string
	Input                 string
	ProductType           string
	Requirements          string
	AutofillMarketplaceId bool
	AutofillLanguageTag   bool
}

type getListingConfig struct {
	DisplayAttritubes bool
}

var (
	createListingsCmd = &cobra.Command{
		Use: "create",
		Run: WrapCommandWithResources(createListings, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceListings}}),
	}
	createListingsCfg createListingsConfig
	getListingCmd     = &cobra.Command{
		Use: "get",
		Run: WrapCommandWithResources(getListing, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceListings}}),
	}
	getListingCfg    getListingConfig
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
	createListingsCmd.MarkFlagRequired("input")
	createListingsCmd.PersistentFlags().StringVarP(&createListingsCfg.ProductType, "type", "p", "", "product type")
	createListingsCmd.PersistentFlags().StringVarP(&createListingsCfg.Requirements, "requirements", "r", "", "")
	createListingsCmd.PersistentFlags().StringVar(&createListingsCfg.IssueLocale, "issue-locale", "", "Locale for issue localization. Default: When no locale is provided, the default locale of the first marketplace is used. Localization defaults to en_US when a localized message is not available in the specified locale.")

	getListingCmd.PersistentFlags().BoolVar(&getListingCfg.DisplayAttritubes, "display-attributes", false, "logs listing attributes line by line if given")
	listingsCmd.PersistentFlags().StringVarP(&listingOperationSku, "sku", "s", "", "")
	listingsCmd.AddCommand(createListingsCmd)
	listingsCmd.AddCommand(getListingCmd)
	listingsCmd.AddCommand(deleteListingCmd)
	return listingsCmd
}

func createListings(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	var params listings.PutListingsItemParams
	params.MarketplaceIds = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID
	params.IncludedData = internal.Ptr([]listings.PutListingsItemParamsIncludedData{"issues"})

	if createListingsCfg.IssueLocale != "" {
		params.IssueLocale = &createListingsCfg.IssueLocale
	}

	var body listings.ListingsItemPutRequest
	body.ProductType = createListingsCfg.ProductType
	body.Requirements = internal.Ptr(listings.ListingsItemPutRequestRequirements(createListingsCfg.Requirements))
	var marketplace_id *fastjson.Value
	var language_tag *fastjson.Value
	if createListingsCfg.AutofillMarketplaceId {
		marketplace_id = fastjson.MustParse(fmt.Sprintf(`"%s"`, cfg.Amazon.Auth.DefaultMerchant.MarketplaceID[0]))
	}
	if createListingsCfg.AutofillLanguageTag {
		language_tag = fastjson.MustParse(fmt.Sprintf(`"%s"`, cfg.Amazon.DefaultLanguageTag))
	}
	var should_fill_marketplace_id = marketplace_id != nil
	var should_fill_language_tag = language_tag != nil
	attr_bytes, err := internal.ReadFile(createListingsCfg.Input)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
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
	// sanity check
	brand_obj := attrs.Get("brand").
		GetArray()[0].
		GetObject()
	brand_bytes := brand_obj.Get("value").GetStringBytes()
	brand := string(brand_bytes)
	brand = strings.TrimSpace(brand)
	brand_obj.Set("value", fastjson.MustParse(fmt.Sprintf(`"%s"`, brand)))
	schema := attrs.Get("$schema")
	if schema != nil {
		attrs.Del("$schema")
	}

	var attr_interface map[string]interface{}
	err = json.Unmarshal(attrs.MarshalTo(nil), &attr_interface)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	body.Attributes = attr_interface

	status, err := app.Amazon.Client.PutListingsItem(cmd.Context(), cfg.Amazon.Auth.DefaultMerchant.SellerToken, listingOperationSku, &params, body)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	result := status.JSON200
	log.Info().
		Str("status", string(result.Status)).
		Str("submission_id", result.SubmissionId).
		Str("sku", result.Sku).
		Send()
	logListingIssues(*result.Issues)
}

func getListing(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	var params listings.GetListingsItemParams
	params.MarketplaceIds = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID
	params.IncludedData = &[]listings.GetListingsItemParamsIncludedData{"summaries", "issues", "offers", "relationships", "attributes"}
	status, err := app.Amazon.Client.GetListingsItem(cmd.Context(), &params, cfg.Amazon.Auth.DefaultMerchant.SellerToken, listingOperationSku)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	result := status.JSON200
	logListingIssues(*result.Issues)
	for _, summary := range *result.Summaries {
		ev := log.Info()
		if summary.Asin != nil {
			ev.Str("asin", *summary.Asin)
		}
		if summary.ConditionType != nil {
			ev.Str("condition", string(*summary.ConditionType))
		}
		if summary.ItemName != nil {
			ev.Str("name", *summary.ItemName)
		}
		if summary.Status != nil {
			var sts strings.Builder
			for i, st := range summary.Status {
				if i > 0 {
					sts.WriteString(",")
				}
				sts.WriteString(string(st))
			}
			ev.Str("status", sts.String())
		}
		ev.Send()

	}
	for _, relationship := range *result.Relationships {
		for i, rls := range relationship.Relationships {
			ev := log.Info().
				Str("type", string(rls.Type))
			if rls.ChildSkus != nil {
				ev.Str("child_skus", strings.Join(*rls.ChildSkus, ","))
			}
			if rls.ParentSkus != nil {
				ev.Str("parent_skus", strings.Join(*rls.ParentSkus, ","))
			}
			if rls.VariationTheme != nil {
				ev.Str("theme", rls.VariationTheme.Theme).
					Str("theme_attributes", strings.Join(rls.VariationTheme.Attributes, ","))
			}
			ev.Msgf("relationship %d", i+1)

		}
	}
	if len(*result.Offers) == 0 {
		log.Warn().Msg("no offers found")
	} else {
		for i, offer := range *result.Offers {
			ev := log.Info().
				Str("type", string(offer.OfferType)).
				Int("points", offer.Points.PointsNumber).
				Str("currency_code", offer.Price.CurrencyCode).
				Str("price", string(offer.Price.Amount))
			if offer.Audience != nil {
				ev.Str("audience_name", *offer.Audience.DisplayName).
					Str("audience_value", *offer.Audience.Value)
			}
			ev.Msgf("offer %d", i+1)
		}
	}
	if getListingCfg.DisplayAttritubes {
		attrs_bytes, err := json.Marshal(result.Attributes)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
		attrs := fastjson.MustParseBytes(attrs_bytes)
		attrs.GetObject().Visit(func(key []byte, v *fastjson.Value) {
			ev := log.Info()
			for _, obj := range v.GetArray() {
				obj.GetObject().Visit(func(key []byte, v *fastjson.Value) {
					if v.Type() == fastjson.TypeString {
						marshalled, err := strconv.Unquote(string(v.MarshalTo(nil)))
						if err != nil {
							log.Error().Err(err).Send()
							return
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
	params.MarketplaceIds = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID
	status, err := app.Amazon.Client.DeleteListingsItem(cmd.Context(), &params, *getProductTypeDefinitionCfg.Params.SellerId, listingOperationSku)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	result := status.JSON200
	log.Info().
		Str("status", string(result.Status)).
		Str("sku", result.Sku).
		Str("submission_id", result.SubmissionId).
		Send()
	logListingIssues(*result.Issues)
}

func logListingIssues(issues []listings.Issue) {
	for _, issue := range issues {
		ev := log.Warn().
			Str("code", issue.Code).
			Str("severity", string(issue.Severity))
		if issue.AttributeNames != nil {
			ev.Str("attributes", strings.Join(*issue.AttributeNames, ","))
		}
		if issue.Enforcements != nil {
			ev.Str("exempt", string(issue.Enforcements.Exemption.Status))
			ev.Interface("exemption_expiry_date", issue.Enforcements.Exemption.ExpiryDate)
			var actions []string
			for _, action := range issue.Enforcements.Actions {
				actions = append(actions, action.Action)
			}
			ev.Str("actions", strings.Join(actions, ","))
		}
		ev.Msg(issue.Message)
	}
}
