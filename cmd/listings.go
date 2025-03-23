package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/listings"
	"github.com/fatih/color"
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
	Related           bool
}

type patchListingConfig struct {
	EditFile string
	Params   listings.PatchListingsItemParams
	Body     listings.PatchListingsItemJSONRequestBody
}

type deleteListingConfig struct {
	Params        listings.DeleteListingsItemParams
	DeleteRelated bool
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
	deleteListingCfg deleteListingConfig
	patchListingCmd  = &cobra.Command{
		Use: "patch",
		Run: WrapCommandWithResources(patchListing, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceListings}}),
	}
	patchListingCfg patchListingConfig
	listingsCmd     = &cobra.Command{
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

	getListingCmd.PersistentFlags().BoolVar(&getListingCfg.DisplayAttritubes, "attributes", false, "logs listing attributes line by line if given")
	getListingCmd.PersistentFlags().BoolVar(&getListingCfg.Related, "related", false, "also display products related with this product (variations etc...)")

	patchListingCmd.PersistentFlags().StringVarP(&patchListingCfg.EditFile, "input", "i", "", "json file containing edits")

	deleteListingCmd.PersistentFlags().BoolVar(&getListingCfg.Related, "related", false, "also delete related (child // parent) listings")

	listingsCmd.PersistentFlags().StringVarP(&listingOperationSku, "sku", "s", "", "")
	listingsCmd.AddCommand(createListingsCmd)
	listingsCmd.AddCommand(getListingCmd)
	listingsCmd.AddCommand(deleteListingCmd)
	listingsCmd.AddCommand(patchListingCmd)
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
	var results = []*listings.Item{result}
	if getListingCfg.Related && result.Relationships != nil {
		for _, relationship := range *result.Relationships {
			for _, rls := range relationship.Relationships {
				if rls.ChildSkus != nil {
					fmt.Printf("%s %s\n", color.CyanString("Related:"), color.YellowString("Querying child SKUs: %s", strings.Join(*rls.ChildSkus, ",")))
					for _, child := range *rls.ChildSkus {
						status, err := app.Amazon.Client.GetListingsItem(cmd.Context(), &params, cfg.Amazon.Auth.DefaultMerchant.SellerToken, child)
						if err != nil {
							log.Error().Err(err).Send()
							return
						}
						results = append(results, status.JSON200)
					}
				}
				if rls.ParentSkus != nil {
					fmt.Printf("%s %s\n", color.CyanString("Related:"), color.YellowString("Querying parent SKUs: %s", strings.Join(*rls.ParentSkus, ",")))
					for _, parent := range *rls.ParentSkus {
						status, err := app.Amazon.Client.GetListingsItem(cmd.Context(), &params, cfg.Amazon.Auth.DefaultMerchant.SellerToken, parent)
						if err != nil {
							log.Error().Err(err).Send()
							return
						}
						results = append(results, status.JSON200)
					}
				}
			}
		}
	}

	for i, result := range results {
		if i > 0 {
			fmt.Println(color.HiYellowString("\n%s\n", strings.Repeat("=", 80)))
		}
		fmt.Println(color.New(color.Bold).Sprintf("LISTING #%d: %s", i+1, result.Sku))
		fmt.Println(color.HiYellowString("%s\n", strings.Repeat("-", 80)))

		if result.Issues != nil && len(*result.Issues) > 0 {
			fmt.Println(color.HiRedString("ISSUES:"))
			printIssues(*result.Issues)
			fmt.Println()
		}

		if result.Summaries != nil && len(*result.Summaries) > 0 {
			fmt.Println(color.HiGreenString("SUMMARIES:"))
			for i, summary := range *result.Summaries {
				if i > 0 {
					fmt.Println()
				}
				fmt.Printf("  %s:\n", color.GreenString("Summary #%d", i+1))
				if summary.Asin != nil {
					fmt.Printf("    %s: %s\n", color.CyanString("ASIN"), *summary.Asin)
				}
				if summary.ConditionType != nil {
					fmt.Printf("    %s: %s\n", color.CyanString("Condition"), string(*summary.ConditionType))
				}
				if summary.ItemName != nil {
					fmt.Printf("    %s: %s\n", color.CyanString("Name"), *summary.ItemName)
				}
				if summary.Status != nil {
					var statuses []string
					for _, st := range summary.Status {
						statuses = append(statuses, string(st))
					}
					fmt.Printf("    %s: %s\n", color.CyanString("Status"), strings.Join(statuses, ", "))
				}
			}
			fmt.Println()
		}

		if result.Relationships != nil && len(*result.Relationships) > 0 {
			fmt.Println(color.HiBlueString("RELATIONSHIPS:"))
			for _, relationship := range *result.Relationships {
				for i, rls := range relationship.Relationships {
					fmt.Printf("  %s:\n", color.BlueString("Relationship #%d", i+1))
					fmt.Printf("    %s: %s\n", color.CyanString("Type"), string(rls.Type))
					if rls.ChildSkus != nil {
						fmt.Printf("    %s (%d): %s\n", color.CyanString("Child SKUs"), len(*rls.ChildSkus), strings.Join(*rls.ChildSkus, ", "))
					}
					if rls.ParentSkus != nil {
						fmt.Printf("    %s: %s\n", color.CyanString("Parent SKUs"), strings.Join(*rls.ParentSkus, ", "))
					}
					if rls.VariationTheme != nil {
						fmt.Printf("    %s: %s\n", color.CyanString("Theme"), rls.VariationTheme.Theme)
						fmt.Printf("    %s: %s\n", color.CyanString("Theme Attributes"), strings.Join(rls.VariationTheme.Attributes, ", "))
					}
					fmt.Println()
				}
			}
		}

		if result.Offers != nil {
			if len(*result.Offers) == 0 {
				fmt.Println(color.HiYellowString("OFFERS: None found"))
			} else {
				fmt.Println(color.HiYellowString("OFFERS:"))
				for i, offer := range *result.Offers {
					fmt.Printf("  %s:\n", color.YellowString("Offer #%d", i+1))
					fmt.Printf("    %s: %s\n", color.CyanString("Type"), string(offer.OfferType))
					fmt.Printf("    %s: %d\n", color.CyanString("Points"), offer.Points.PointsNumber)
					fmt.Printf("    %s: %s %s\n", color.CyanString("Price"), string(offer.Price.Amount), offer.Price.CurrencyCode)

					if offer.Audience != nil {
						fmt.Printf("    %s: %s (%s)\n", color.CyanString("Audience"), *offer.Audience.DisplayName, *offer.Audience.Value)
					}
					fmt.Println()
				}
			}
		}

		if getListingCfg.DisplayAttritubes && result.Attributes != nil {
			fmt.Println(color.HiCyanString("ATTRIBUTES:"))
			attrs_bytes, err := json.Marshal(result.Attributes)
			if err != nil {
				log.Error().Err(err).Send()
				return
			}

			var prettyJSON bytes.Buffer
			err = json.Indent(&prettyJSON, attrs_bytes, "  ", "  ")
			if err != nil {
				log.Error().Err(err).Send()
				return
			}

			printJSONWithPaths(fastjson.MustParseBytes(attrs_bytes), "/attributes", 2)
			fmt.Println()
		}
	}
}

func printIssues(issues []listings.Issue) {
	for i, issue := range issues {
		fmt.Printf("  %s:\n", color.RedString("Issue #%d", i+1))
		fmt.Printf("    %s: %s\n", color.CyanString("Code"), issue.Code)
		fmt.Printf("    %s: %s\n", color.CyanString("Message"), issue.Message)
		fmt.Printf("    %s: %s\n", color.CyanString("Severity"), string(issue.Severity))
		if issue.AttributeNames != nil {
			fmt.Printf("    %s: %s\n", color.CyanString("Attribute"), strings.Join(*issue.AttributeNames, ","))
		}
		fmt.Println()
	}
}

func printJSONWithPaths(v *fastjson.Value, currentPath string, indent int) {
	indentStr := strings.Repeat("  ", indent)
	pathColor := color.New(color.FgHiBlue).SprintFunc()
	valueColor := color.New(color.FgHiWhite).SprintFunc()
	typeColor := color.New(color.FgYellow).SprintFunc()

	switch v.Type() {
	case fastjson.TypeObject:
		fmt.Printf("%s%s %s\n", indentStr, pathColor(currentPath), typeColor("(Object)"))
		v.GetObject().Visit(func(key []byte, childValue *fastjson.Value) {
			keyStr := string(key)
			newPath := currentPath + "/" + keyStr

			// Print value directly if it's a simple type
			switch childValue.Type() {
			case fastjson.TypeString:
				marshalled, err := strconv.Unquote(string(childValue.MarshalTo(nil)))
				if err == nil {
					fmt.Printf("%s%s: %s %s\n", indentStr+"  ", pathColor(keyStr), valueColor(marshalled), typeColor("(String)"))
				} else {
					fmt.Printf("%s%s: %s\n", indentStr+"  ", pathColor(keyStr), typeColor("(String - Error unquoting)"))
				}
			case fastjson.TypeNumber:
				fmt.Printf("%s%s: %s %s\n", indentStr+"  ", pathColor(keyStr), valueColor(string(childValue.MarshalTo(nil))), typeColor("(Number)"))
			case fastjson.TypeTrue, fastjson.TypeFalse:
				fmt.Printf("%s%s: %s %s\n", indentStr+"  ", pathColor(keyStr), valueColor(string(childValue.MarshalTo(nil))), typeColor("(Boolean)"))
			case fastjson.TypeNull:
				fmt.Printf("%s%s: %s\n", indentStr+"  ", pathColor(keyStr), typeColor("(Null)"))
			case fastjson.TypeObject:
				printJSONWithPaths(childValue, newPath, indent+1)
			case fastjson.TypeArray:
				fmt.Printf("%s%s %s\n", indentStr+"  ", pathColor(keyStr), typeColor("(Array)"))
				printJSONWithPaths(childValue, newPath, indent+1)
			}
		})
	case fastjson.TypeArray:
		fmt.Printf("%s%s %s\n", indentStr, pathColor(currentPath), typeColor("(Array)"))
		arr := v.GetArray()
		for i, item := range arr {
			indexPath := fmt.Sprintf("%s/%d", currentPath, i)

			// For simple values in arrays, print them directly
			switch item.Type() {
			case fastjson.TypeString:
				marshalled, err := strconv.Unquote(string(item.MarshalTo(nil)))
				if err == nil {
					fmt.Printf("%s[%d]: %s %s\n", indentStr+"  ", i, valueColor(marshalled), typeColor("(String)"))
				} else {
					fmt.Printf("%s[%d]: %s\n", indentStr+"  ", i, typeColor("(String - Error unquoting)"))
				}
			case fastjson.TypeNumber:
				fmt.Printf("%s[%d]: %s %s\n", indentStr+"  ", i, valueColor(string(item.MarshalTo(nil))), typeColor("(Number)"))
			case fastjson.TypeTrue, fastjson.TypeFalse:
				fmt.Printf("%s[%d]: %s %s\n", indentStr+"  ", i, valueColor(string(item.MarshalTo(nil))), typeColor("(Boolean)"))
			case fastjson.TypeNull:
				fmt.Printf("%s[%d]: %s\n", indentStr+"  ", i, typeColor("(Null)"))
			case fastjson.TypeObject, fastjson.TypeArray:
				// For complex types, continue recursion
				printJSONWithPaths(item, indexPath, indent+1)
			}
		}
	}
}

func deleteListing(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	var getListingParams listings.GetListingsItemParams
	getListingParams.MarketplaceIds = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID
	getListingParams.IncludedData = &[]listings.GetListingsItemParamsIncludedData{"relationships"}
	getListingParams.IssueLocale = internal.Ptr("en_US")
	status, err := app.Amazon.Client.GetListingsItem(cmd.Context(), &getListingParams, cfg.Amazon.Auth.DefaultMerchant.SellerToken, listingOperationSku)
	if err != nil {
		log.Error().Err(err).Msg("error getting listing")
		return
	}
	deleteListingCfg.Params.MarketplaceIds = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID
	deleteListingCfg.Params.IssueLocale = internal.Ptr("en_US")

	result := status.JSON200
	if getListingCfg.Related && result.Relationships != nil {
		for _, relationship := range *result.Relationships {
			for _, rls := range relationship.Relationships {
				if rls.ChildSkus != nil {
					fmt.Printf("%s %s\n", color.CyanString("Related:"), color.YellowString("Deleting child SKUs: %s", strings.Join(*rls.ChildSkus, ",")))
					for _, child := range *rls.ChildSkus {
						_, err := app.Amazon.Client.DeleteListingsItem(cmd.Context(), &deleteListingCfg.Params, cfg.Amazon.Auth.DefaultMerchant.SellerToken, child)
						if err != nil {
							log.Error().Err(err).Str("sku", child).Msg("error deleting child sku")
							return
						}
					}
				}
				if rls.ParentSkus != nil {
					fmt.Printf("%s %s\n", color.CyanString("Related:"), color.YellowString("Deleting parent SKUs: %s", strings.Join(*rls.ParentSkus, ",")))
					for _, parent := range *rls.ParentSkus {
						_, err := app.Amazon.Client.DeleteListingsItem(cmd.Context(), &deleteListingCfg.Params, cfg.Amazon.Auth.DefaultMerchant.SellerToken, parent)
						if err != nil {
							log.Error().Err(err).Str("sku", parent).Msg("error deleting parent sku")
							return
						}
					}
				}
			}
		}
	}

	deleteListingCfg.Params.MarketplaceIds = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID
	deleteListingCfg.Params.IssueLocale = internal.Ptr("en_US")
	deleteStatus, err := app.Amazon.Client.DeleteListingsItem(cmd.Context(), &deleteListingCfg.Params, cfg.Amazon.Auth.DefaultMerchant.SellerToken, listingOperationSku)
	if err != nil {
		log.Error().Err(err).Msg("error deleting listing")
		return
	}
	deleteResult := deleteStatus.JSON200
	log.Info().
		Str("status", string(deleteResult.Status)).
		Str("sku", result.Sku).
		Str("submission_id", deleteResult.SubmissionId).
		Send()
	logListingIssues(*deleteResult.Issues)
}

func patchListing(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	logger := log.With().Str("path", patchListingCfg.EditFile).Logger()
	patch_byte, err := internal.ReadFile(patchListingCfg.EditFile)
	if err != nil {
		logger.Error().Err(err).Msg("error reading patch file")
		return
	}
	edit, err := fastjson.ParseBytes(patch_byte)
	if err != nil {
		logger.Error().Err(err).Msg("error parsing patch file")
		return
	}
	productTypeVal := edit.Get("productType")
	if productTypeVal == nil {
		logger.Error().Msg("no product type found (looking for key: productType)")
		return
	}
	patchesVal := edit.Get("patches")
	if patchesVal == nil {
		logger.Error().Msg("no patch found (looking for key: patches)")
	}
	patchListingCfg.Body.ProductType = "WALLET"
	patches := patchesVal.GetArray()
	if patches == nil {
		log.Error().Msg("patch is not array")
		return
	}
	var patch_ops = make([]listings.PatchOperation, 0, len(patches))
	for i, patch := range patches {
		var patch_op listings.PatchOperation
		ev := log.With().Int("index", i).Logger()
		op_str := patch.GetStringBytes("op")
		if op_str == nil {
			bold := color.New(color.Bold)
			ev.Error().Msg(fmt.Sprintf("patch is missing op or not string (looking for key: op), valid values are %s, %s and %s", bold.Sprint("add"), bold.Sprint("replace"), bold.Sprint("delete")))
			return
		}
		path := patch.GetStringBytes("path")
		if path == nil {
			ev.Error().Msg("patch is missing path or not string (looking for key: path)")
			return
		}
		value := patch.Get("value")
		if value == nil {
			ev.Error().Msg("patch is missing value (looking for key: value)")
			return
		}
		if value.Type() != fastjson.TypeArray {
			ev.Error().Msg("patchs value is not array")
			return
		}
		var val *[]map[string]interface{}
		if err := json.Unmarshal(value.MarshalTo(nil), &val); err != nil {
			log.Error().Err(err).Send()
			return
		}
		patch_op.Value = val
		patch_op.Op = listings.PatchOperationOp(string(op_str))
		patch_op.Path = string(path)
		patch_ops = append(patch_ops, patch_op)
	}
	patchListingCfg.Params.MarketplaceIds = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID
	patchListingCfg.Params.IncludedData = &[]listings.PatchListingsItemParamsIncludedData{"issues"}
	patchListingCfg.Params.IssueLocale = internal.Ptr("en_US")
	patchListingCfg.Body.Patches = patch_ops
	status, err := app.Amazon.Client.PatchListingsItem(cmd.Context(), &patchListingCfg.Params, patchListingCfg.Body, cfg.Amazon.Auth.DefaultMerchant.SellerToken, listingOperationSku)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	result := status.JSON200
	if result.Issues != nil {
		logListingIssues(*status.JSON200.Issues)
	}

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
