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
)

type CreateListingsConfig struct {
	DryRun      bool
	IssueLocale string
	Input       string
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
	createListingsCmd.PersistentFlags().StringVarP(&createListingsCfg.Input, "input", "i", "", "Input JSON file")
	createListingsCmd.PersistentFlags().BoolVar(&createListingsCfg.DryRun, "dry-run", false, "If set, creation mode will be set to VALIDATION_PREVIEW.")
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

	if createListingsCfg.DryRun {
		log.Info().Msg("dry run, running with preview mode")
		// params.Mode = ptr.String("VALIDATION_PREVIEW")
	}
	if createListingsCfg.IssueLocale != "" {
		params.IssueLocale = &createListingsCfg.IssueLocale
	}

	var body models.ListingsItemPutRequest
	body.ProductType = ptr.String(listing.ProductType)
	body.Requirements = listing.Requirements
	// TODO: THIS PIECE OF SHIT DOES NOT WORK
	// STRAIGHT UP, DOES, NOT, WORK.
	// 3:00AM WRN The provided value for 'Closure' is invalid. attribute=closure category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Model Name' is invalid. attribute=model_name category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Age Range Description' is invalid. attribute=age_range_description category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Part Number' is invalid. attribute=part_number category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Material' is invalid. attribute=material category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Number Of Pockets' is invalid. attribute=number_of_pockets category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Brand Name' is invalid. attribute=brand category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Number of Sections' is invalid. attribute=number_of_sections category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Country of Origin' is invalid. attribute=country_of_origin category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Lining Description' is invalid. attribute=lining_description category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Is exempt from a supplier declared external identifier' is invalid. attribute=supplier_declared_has_product_identifier_exemption category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Form Factor' is invalid. attribute=form_factor category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Style' is invalid. attribute=style category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Wallet Card Slot Count' is invalid. attribute=wallet_card_slot_count category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Subject Character' is invalid. attribute=subject_character category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Product Description' is invalid. attribute=product_description category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Product Care Instructions' is invalid. attribute=care_instructions category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Dangerous Goods Regulations' is invalid. attribute=supplier_declared_dg_hz_regulation category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Bullet Point' is invalid. attribute=bullet_point category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Compliance - Wallet Type' is invalid. attribute=compliance_wallet_type category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Item Type Keyword' is invalid. attribute=item_type_keyword category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Item Display Dimensions' is invalid. attribute=item_display_dimensions category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Leather Type' is invalid. attribute=leather_type category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Title' is invalid. attribute=item_name category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Wallet Compartment Type' is invalid. attribute=wallet_compartment_type category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Number of Compartments' is invalid. attribute=number_of_compartments category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Color' is invalid. attribute=color category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Pocket Description' is invalid. attribute=pocket_description category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Pattern' is invalid. attribute=pattern category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Item Weight' is invalid. attribute=item_weight category= code=4000001 severity=ERROR
	// 3:00AM WRN The provided value for 'Embellishment Feature' is invalid. attribute=embellishment_feature category= code=4000001 severity=ERROR
	// NO MATTER WHAT I DO, IT JUST REFUSES TO READ THE ATTRIBUTES, AT ALL.
	//
	// i tried using json.RawMessage, json.Marshal(fastjson.MustParseBytes(listings_bytes).GetObject("attributes").MarshalTo(nil))
	// none of the solutions works
	// this h-1b abuser slavery mill abomination cant even get a single API endpoint right but they wont hire you if you cant invert binary tree on whiteboard
	// I AM SO FUCKING TIRED
	// ITS BEEN THREE HOURS AND IT JUST DOES NOT READ ATTRIBUTES
	// AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
	// see you tomorrow
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
			Str("category", strings.Join(issue.Categories, ",")).
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
