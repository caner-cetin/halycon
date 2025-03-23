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

	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inbound"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
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

	var params fba_inbound.CreateInboundPlanRequest
	params.DestinationMarketplaces = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID

	ev := log.Error()
	var msg string = ""
	if msg != "" {
		ev.Msg(msg)
		return
	}
	ev.Discard()
	params.SourceAddress = fba_inbound.AddressInput{
		AddressLine1: cfg.Amazon.FBA.DefaultShipFrom.AddressLine1,
		City:         cfg.Amazon.FBA.DefaultShipFrom.City,
		Name:         cfg.Amazon.FBA.DefaultShipFrom.Name,
		PhoneNumber:  cfg.Amazon.FBA.DefaultShipFrom.PhoneNumber,
		PostalCode:   cfg.Amazon.FBA.DefaultShipFrom.PostalCode,
		CountryCode:  cfg.Amazon.FBA.DefaultShipFrom.CountryCode,
	}
	if cfg.Amazon.FBA.DefaultShipFrom.AddressLine2 != "" {
		params.SourceAddress.AddressLine2 = &cfg.Amazon.FBA.DefaultShipFrom.AddressLine2
	}
	if cfg.Amazon.FBA.DefaultShipFrom.CompanyName != "" {
		params.SourceAddress.CompanyName = &cfg.Amazon.FBA.DefaultShipFrom.CompanyName
	}
	if cfg.Amazon.FBA.DefaultShipFrom.StateOrProvince != "" {
		params.SourceAddress.StateOrProvinceCode = &cfg.Amazon.FBA.DefaultShipFrom.StateOrProvince
	}
	if cfg.Amazon.FBA.DefaultShipFrom.Email != "" {
		params.SourceAddress.Email = &cfg.Amazon.FBA.DefaultShipFrom.Email
	}

	input, err := internal.OpenFile(createShipmentPlanCfg.Input)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
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
	defaultPrepOwner := fba_inbound.NONE
	defaultLabelOwner := fba_inbound.LabelOwnerNONE

	prepRequirements := loadPrepRequirements()

	items := make([]fba_inbound.ItemInput, 0, len(products)-1)
	for _, product := range products[1:] {
		sku := product[1]
		quantity, err := strconv.Atoi(product[3])
		if err != nil {
			log.Error().Err(err).Str("quantity", product[3]).Msg("error while converting quantity to integer")
			return
		}

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

		items = append(items, fba_inbound.ItemInput{
			Msku:       sku,
			Quantity:   quantity,
			PrepOwner:  prepOwner,
			LabelOwner: fba_inbound.LabelOwner(labelOwner),
		})
	}
	params.Items = items

	status, err := app.Amazon.Client.CreateFBAInboundPlan(cmd.Context(), params)
	if err != nil {
		if prepErrors := extractPrepOwnerErrors(err); len(prepErrors) > 0 {
			log.Info().Msg("Found SKUs requiring prep, updating and retrying...")

			for sku := range prepErrors {
				requirements := ItemRequirements{
					PrepOwner:  fba_inbound.NONE,
					LabelOwner: fba_inbound.LabelOwnerNONE,
				}

				if existing, exists := prepRequirements[sku]; exists {
					requirements = existing
				}

				if strings.Contains(err.Error(), sku+" requires prepOwner") {
					requirements.PrepOwner = fba_inbound.SELLER
				}
				if strings.Contains(err.Error(), sku+" requires labelOwner") {
					requirements.LabelOwner = fba_inbound.LabelOwnerSELLER
				}

				prepRequirements[sku] = requirements
			}

			savePrepRequirements(prepRequirements)
			for i, item := range items {
				if _, exists := prepErrors[item.Msku]; exists {
					if strings.Contains(err.Error(), item.Msku+" requires prepOwner") {
						items[i].PrepOwner = fba_inbound.SELLER
					}

					if strings.Contains(err.Error(), item.Msku+" requires labelOwner") {
						items[i].LabelOwner = fba_inbound.LabelOwnerSELLER
					}
				}
			}

			status, err = app.Amazon.Client.CreateFBAInboundPlan(cmd.Context(), params)
			if err != nil {
				log.Error().Err(err).Msg("error occurred while creating inbound shipment plan after prep update")
				return
			}
		} else {
			log.Error().Err(err).Msg("error occurred while creating inbound shipment plan")
			return
		}
	}
	result := status.JSON202
	log.Info().Str("inbound_plan_id", result.InboundPlanId).Str("operation_id", result.OperationId).Msg("success!")
	shouldOpenPlan, err := internal.PromptFor("Open plan with default browser? [y/N]")
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	if strings.TrimSpace(strings.ToLower(strings.ToLower(shouldOpenPlan))) == "y" {
		// todo: change .com, I dont know how other marketplaces works.
		url := fmt.Sprintf("https://sellercentral.amazon.com/fba/sendtoamazon/confirm_content_step?wf=%s", result.InboundPlanId)
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
	PrepOwner fba_inbound.PrepOwner `json:"prep_owner"`

	// LabelOwner indicates the entity responsible for labeling the item for shipment.
	LabelOwner fba_inbound.LabelOwner `json:"label_owner"`
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
			prepErrors[sku] = string(fba_inbound.SELLER)
		}
	}

	labelMatches := reLabelOwner.FindAllStringSubmatch(errorStr, -1)
	for _, match := range labelMatches {
		if len(match) >= 2 {
			sku := match[1]
			prepErrors[sku] = string(fba_inbound.LabelOwnerSELLER)
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
	status, err := app.Amazon.Client.GetInboundOperationStatus(cmd.Context(), operationId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	response := status.JSON200
	logger := log.With().
		Str("id", response.OperationId).
		Str("status", string(response.OperationStatus)).
		Logger()
	if response.OperationStatus == fba_inbound.FAILED {
		logger.Warn().Send()
	} else {
		logger.Info().Send()
	}
	for i, problem := range response.OperationProblems {
		ev := log.Warn().
			Str("code", problem.Code).
			Str("message", problem.Message).
			Str("severity", problem.Severity)
		if problem.Details != nil {
			ev.Str("details", *problem.Details)
		}
		ev.Msgf("problem %d", i+1)
	}
}
