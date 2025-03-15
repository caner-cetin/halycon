package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inbound/client/fba_inbound"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inbound/models"
	"github.com/caner-cetin/halycon/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type createShipmentPlanConfig struct {
	Input string
}

var (
	createShipmentPlanCmd = &cobra.Command{
		Use: "create",
		Run: WrapCommandWithResources(createShipmentPlan, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceFBAInbound}}),
	}
	createShipmentPlanCfg = createShipmentPlanConfig{}
	shipmentCmd           = &cobra.Command{
		Use: "shipment",
	}
)

var (
	operationStatusCmd = &cobra.Command{
		Use: "status",
		Run: WrapCommandWithResources(getOperationStatusCmd, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceFBAInbound}}),
	}

	operationCmd = &cobra.Command{
		Use: "operation",
	}
	operationId string
)

func getShipmentCmd() *cobra.Command {
	createShipmentPlanCmd.PersistentFlags().StringVarP(&createShipmentPlanCfg.Input, "input", "i", "", "comma delimited input consisting ASIN, SKU, product name, quantity in order (output of asin to sku command)")

	operationStatusCmd.PersistentFlags().StringVarP(&operationId, "id", "i", "", "operation id")
	operationCmd.AddCommand(operationStatusCmd)
	shipmentCmd.AddCommand(operationCmd)

	shipmentCmd.AddCommand()
	shipmentCmd.AddCommand(createShipmentPlanCmd)
	return shipmentCmd
}
func createShipmentPlan(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)

	params := fba_inbound.NewCreateInboundPlanParams()
	params.Body = new(models.CreateInboundPlanRequest)
	params.Body.DestinationMarketplaces = viper.GetStringSlice(config.AMAZON_MARKETPLACE_ID.Key)

	ev := log.Error()
	var msg string = ""
	if !viper.IsSet(config.AMAZON_FBA_SHIP_FROM_ADDRESS_LINE_1.Key) {
		msg = "shipment address line 1 must be set"
	}
	if !viper.IsSet(config.AMAZON_FBA_SHIP_FROM_CITY.Key) {
		msg = "shipment city must be set"
	}
	if !viper.IsSet(config.AMAZON_FBA_SHIP_FROM_NAME.Key) {
		msg = "contact name must be set"
	}
	if !viper.IsSet(config.AMAZON_FBA_SHIP_FROM_PHONE_NUMBER.Key) {
		msg = "phone number must be set"
	}
	if !viper.IsSet(config.AMAZON_FBA_SHIP_FROM_POSTAL_CODE.Key) {
		msg = "postal code must be set"
	}
	if msg != "" {
		ev.Msg(msg)
		return
	}
	ev.Discard()
	params.Body.SourceAddress = &models.AddressInput{
		AddressLine1: ptr.String(viper.GetString(config.AMAZON_FBA_SHIP_FROM_ADDRESS_LINE_1.Key)),
		City:         ptr.String(viper.GetString(config.AMAZON_FBA_SHIP_FROM_CITY.Key)),
		Name:         ptr.String(viper.GetString(config.AMAZON_FBA_SHIP_FROM_NAME.Key)),
		PhoneNumber:  ptr.String(viper.GetString(config.AMAZON_FBA_SHIP_FROM_PHONE_NUMBER.Key)),
		PostalCode:   ptr.String(viper.GetString(config.AMAZON_FBA_SHIP_FROM_POSTAL_CODE.Key)),
		CountryCode:  ptr.String(viper.GetString(config.AMAZON_FBA_SHIP_FROM_COUNTRY_CODE.Key)),
	}
	if viper.IsSet(config.AMAZON_FBA_SHIP_FROM_ADDRESS_LINE_2.Key) {
		params.Body.SourceAddress.AddressLine2 = viper.GetString(config.AMAZON_FBA_SHIP_FROM_ADDRESS_LINE_2.Key)
	}
	if viper.IsSet(config.AMAZON_FBA_SHIP_FROM_COMPANY_NAME.Key) {
		params.Body.SourceAddress.CompanyName = viper.GetString(config.AMAZON_FBA_SHIP_FROM_COMPANY_NAME.Key)
	}
	if viper.IsSet(config.AMAZON_FBA_SHIP_FROM_STATE_PROVINCE.Key) {
		params.Body.SourceAddress.StateOrProvinceCode = viper.GetString(config.AMAZON_FBA_SHIP_FROM_STATE_PROVINCE.Key)
	}
	if viper.IsSet(config.AMAZON_FBA_SHIP_FROM_EMAIL.Key) {
		params.Body.SourceAddress.Email = viper.GetString(config.AMAZON_FBA_SHIP_FROM_EMAIL.Key)
	}

	input := internal.OpenFile(createShipmentPlanCfg.Input)
	defer input.Close()
	reader := csv.NewReader(input)
	products, err := reader.ReadAll()
	if err != nil {
		log.Error().
			Err(err).
			Str("file", createShipmentPlanCfg.Input).
			Msg("error reading csv")
		return
	}

	// todo: config key
	defaultPrepOwner := models.PrepOwnerNONE
	defaultLabelOwner := models.LabelOwnerNONE

	prepRequirements := loadPrepRequirements()

	items := make([]*models.ItemInput, 0, len(products)-1)
	for _, product := range products[1:] {
		sku := product[1]
		quantity, err := strconv.Atoi(product[3])
		if err != nil {
			log.Error().Err(err).Str("quantity", product[3]).Msg("error while converting quantity to integer")
			return
		}
		quantity_64 := int64(quantity)

		prepOwner := defaultPrepOwner
		labelOwner := defaultLabelOwner

		if requirements, exists := prepRequirements[sku]; exists {
			if requirements.PrepOwner != "" {
				prepOwner = requirements.PrepOwner
			}
			if requirements.LabelOwner != "" {
				labelOwner = requirements.LabelOwner
			}
		}

		items = append(items, &models.ItemInput{
			Msku:       &sku,
			Quantity:   &quantity_64,
			PrepOwner:  models.NewPrepOwner(prepOwner),
			LabelOwner: models.NewLabelOwner(labelOwner),
		})
	}
	params.Body.Items = items

	result, err := app.Amazon.Client.CreateFBAInboundPlan(params)
	if err != nil {
		if prepErrors := extractPrepOwnerErrors(err); len(prepErrors) > 0 {
			log.Info().Msg("Found SKUs requiring prep, updating and retrying...")

			for sku := range prepErrors {
				requirements := ItemRequirements{
					PrepOwner:  models.PrepOwnerNONE,
					LabelOwner: models.LabelOwnerNONE,
				}

				if existing, exists := prepRequirements[sku]; exists {
					requirements = existing
				}

				if strings.Contains(err.Error(), sku+" requires prepOwner") {
					requirements.PrepOwner = models.PrepOwnerSELLER
				}
				if strings.Contains(err.Error(), sku+" requires labelOwner") {
					requirements.LabelOwner = models.LabelOwnerSELLER
				}

				prepRequirements[sku] = requirements
			}

			savePrepRequirements(prepRequirements)
			for i, item := range items {
				if _, exists := prepErrors[*item.Msku]; exists {
					if strings.Contains(err.Error(), *item.Msku+" requires prepOwner") {
						items[i].PrepOwner = models.NewPrepOwner(models.PrepOwnerSELLER)
					}

					if strings.Contains(err.Error(), *item.Msku+" requires labelOwner") {
						items[i].LabelOwner = models.NewLabelOwner(models.LabelOwnerSELLER)
					}
				}
			}

			result, err = app.Amazon.Client.CreateFBAInboundPlan(params)
			if err != nil {
				log.Error().Err(err).Msg("error occurred while creating inbound shipment plan after prep update")
				return
			}
		} else {
			log.Error().Err(err).Msg("error occurred while creating inbound shipment plan")
			return
		}
	}

	log.Info().Str("inbound_plan_id", *result.Payload.InboundPlanID).Str("operation_id", *result.Payload.OperationID).Msg("success!")
	shouldOpenPlan, err := internal.PromptFor("Open plan with default browser? [y/N]")
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	if strings.TrimSpace(strings.ToLower(strings.ToLower(shouldOpenPlan))) == "y" {
		// todo: change .com, I dont know how other marketplaces works.
		url := fmt.Sprintf("https://sellercentral.amazon.com/fba/sendtoamazon/confirm_content_step?wf=%s", *result.Payload.InboundPlanID)
		err = internal.OpenURL(url)
		if err != nil {
			log.Error().Str("url", url).Err(err).Send()
			return
		}
	}
}

// ItemRequirements defines the ownership requirements for item preparation and labeling.
// It specifies which entities are responsible for preparing and labeling items in a shipment.
type ItemRequirements struct {
	// PrepOwner indicates the entity responsible for preparing the item for shipment.
	PrepOwner models.PrepOwner `json:"prep_owner"`

	// LabelOwner indicates the entity responsible for labeling the item for shipment.
	LabelOwner models.LabelOwner `json:"label_owner"`
}

// PrepRequirements is a mapping from preparation ID to the items required for that preparation.
// It defines the requirements needed for different preparation processes in a shipment.
type PrepRequirements map[string]ItemRequirements

func extractPrepOwnerErrors(err error) map[string]string {
	prepErrors := make(map[string]string)

	rePrepOwner := regexp.MustCompile(`ERROR: ([A-Za-z0-9_-]+) requires prepOwner but NONE was assigned`)
	reLabelOwner := regexp.MustCompile(`ERROR: ([A-Za-z0-9_-]+) requires labelOwner but NONE was assigned`)

	errorStr := err.Error()

	prepMatches := rePrepOwner.FindAllStringSubmatch(errorStr, -1)
	for _, match := range prepMatches {
		if len(match) >= 2 {
			sku := match[1]
			prepErrors[sku] = string(models.PrepOwnerSELLER)
		}
	}

	labelMatches := reLabelOwner.FindAllStringSubmatch(errorStr, -1)
	for _, match := range labelMatches {
		if len(match) >= 2 {
			sku := match[1]
			prepErrors[sku] = string(models.PrepOwnerSELLER)
		}
	}

	return prepErrors
}

func loadPrepRequirements() PrepRequirements {
	prepRequirements := make(PrepRequirements)
	cacheFile := filepath.Join(os.TempDir(), "halycon_item_requirements.json")
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return prepRequirements
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		log.Warn().Err(err).Msg("could not read item requirements cache")
		return prepRequirements
	}
	err = json.Unmarshal(data, &prepRequirements)
	if err != nil {
		log.Warn().Err(err).Msg("could not parse item requirements cache")
		return prepRequirements
	}

	return prepRequirements
}

func savePrepRequirements(prepRequirements PrepRequirements) {
	cacheFile := filepath.Join(os.TempDir(), "halycon_item_requirements.json")
	data, err := json.MarshalIndent(prepRequirements, "", "  ")
	if err != nil {
		log.Warn().Err(err).Msg("could not marshal item requirements")
		return
	}
	err = os.WriteFile(cacheFile, data, 0644)
	if err != nil {
		log.Warn().Err(err).Msg("could not write item requirements cache")
	}
}

func getOperationStatusCmd(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	app := ctx.Value(internal.APP_CONTEXT).(AppCtx)
	params := fba_inbound.NewGetInboundOperationStatusParams()
	params.OperationID = operationId
	status, err := app.Amazon.Client.GetInboundOperationStatus(params)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	logger := log.With().
		Str("id", *status.Payload.OperationID).
		Str("status", string(*status.Payload.OperationStatus)).
		Logger()
	if *status.Payload.OperationStatus == "FAILED" {
		logger.Warn().Send()
	} else {
		logger.Info().Send()
	}
	for i, problem := range status.Payload.OperationProblems {
		log.Warn().
			Str("code", *problem.Code).
			Str("message", *problem.Message).
			Str("details", *problem.Details).
			Str("severity", *problem.Severity).
			Msgf("problem %d", i+1)
	}
}
