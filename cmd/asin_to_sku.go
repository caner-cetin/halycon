package cmd

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inventory/client/fba_inventory"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type LookupSkuFromAsinConfig struct {
	ForceRebuildCache bool
	Single            bool
	// may be a single Input string or a path to the text file
	// that contains ASINs depending on Single flag
	Input  string
	Output string
}

type FBAProduct struct {
	SKU         string
	ProductName string
}

var (
	lookupSkuFromAsinCmd = &cobra.Command{
		Use: "asin-to-sku",
		Run: WrapCommandWithResources(lookupSkuFromAsin, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceFBAInventory}}),
	}
	lookupSkuFromAsinCfg = LookupSkuFromAsinConfig{}
)

func getLookupSkuFromAsinCmd() *cobra.Command {
	lookupSkuFromAsinCmd.PersistentFlags().BoolVarP(&lookupSkuFromAsinCfg.Single, "single", "s", false, "query single ASIN, if given, [--asin/-a] flag must be the product ASIN, not the file")
	lookupSkuFromAsinCmd.PersistentFlags().StringVarP(&lookupSkuFromAsinCfg.Input, "input", "i", "", "newline delimited (one per line) text file that contains ASINs, or, a single ASIN (if so, --single flag must be provided)")
	lookupSkuFromAsinCmd.PersistentFlags().BoolVar(&lookupSkuFromAsinCfg.ForceRebuildCache, "force-rebuild", false, "forces rebuilding the ASIN / SKU mapping cache")
	lookupSkuFromAsinCmd.PersistentFlags().StringVarP(&lookupSkuFromAsinCfg.Output, "output", "o", "", "output for SKU list (*.csv), not required when single ASIN is queried")
	return lookupSkuFromAsinCmd
}

func lookupSkuFromAsin(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	app := ctx.Value(internal.APP_CONTEXT).(AppCtx)
	productMap, err := getAsinToMskuMap(app.Amazon.Client)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if lookupSkuFromAsinCfg.Single {
		product, ok := productMap[lookupSkuFromAsinCfg.Input]
		if !ok {
			log.Fatal().Str("asin", lookupSkuFromAsinCfg.Input).Msg("product not found")
		}
		log.Info().Str("name", product.ProductName).Str("sku", product.SKU).Msg("found")
	} else {
		var input []byte
		if input, err = os.ReadFile(lookupSkuFromAsinCfg.Input); err != nil {
			if os.IsNotExist(err) {
				log.Fatal().Err(err).Msgf("path %s does not exist", lookupAsinFromUpcConfig.Input)
			}
			log.Fatal().Err(err).Msg("unknown error while reading contents of text file")
		}
		scanner := bufio.NewScanner(bytes.NewReader(input))
		scanner.Split(bufio.ScanLines)
		var asins []string
		for scanner.Scan() {
			asins = append(asins, strings.TrimSpace(scanner.Text()))
		}

		output_tmp, err := os.CreateTemp(os.TempDir(), "halcyon-asin-to-sku-output-*.csv")
		if err != nil {
			log.Fatal().Err(err).Msg("error while creating temporary output file")
		}
		defer output_tmp.Close()
		defer os.Remove(output_tmp.Name())

		writer := csv.NewWriter(output_tmp)
		writer.Write([]string{"ASIN", "SKU", "Product Name", "Quantity"})
		for _, asin := range asins {
			product, ok := productMap[asin]
			if !ok {
				log.Warn().Str("asin", asin).Msg("cannot find a product")
			}
			writer.Write([]string{asin, product.SKU, product.ProductName, ""})
		}
		writer.Flush()

		output, err := os.Create(lookupSkuFromAsinCfg.Output)
		if err != nil {
			log.Fatal().Err(err).Msg("error while creating the output file")
		}
		_, err = output_tmp.Seek(0, io.SeekStart)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		_, err = io.Copy(output, output_tmp)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		log.Info().Str("file", lookupSkuFromAsinCfg.Output).Msg("saved csv")
	}

}

func getAsinToMskuMap(amazonClient *amazon.AuthorizedClient) (map[string]FBAProduct, error) {
	cacheFilePath := filepath.Join(os.TempDir(), "halycon_amazon_asin_msku_map.json")
	var asinSkuMap = make(map[string]FBAProduct)
	if !lookupSkuFromAsinCfg.ForceRebuildCache {
		if data, err := os.ReadFile(cacheFilePath); err == nil {
			if err := json.Unmarshal(data, &asinSkuMap); err == nil {
				log.Info().Msgf("loaded ASIN to MSKU mapping from cache (%d items)", len(asinSkuMap))
				return asinSkuMap, nil
			}
			log.Warn().Err(err).Msg("failed to parse cached ASIN to MSKU mapping, rebuilding...")
		}
	}

	var nextToken = new(string)

	var i = 0
	for {
		params := fba_inventory.GetInventorySummariesParams{}
		params.MarketplaceIds = viper.GetStringSlice(internal.CONFIG_KEY_AMAZON_MARKETPLACE_ID)
		params.GranularityType = "Marketplace"
		params.GranularityID = viper.GetStringSlice(internal.CONFIG_KEY_AMAZON_MARKETPLACE_ID)[0]
		params.Details = aws.Bool(true)

		if nextToken != nil {
			params.NextToken = nextToken
		}

		result, err := amazonClient.GetFBAInventorySummaries(&params)
		if err != nil {
			return nil, err
		}

		for _, summary := range result.Payload.Payload.InventorySummaries {
			if summary.Asin != "" {
				asinSkuMap[summary.Asin] = FBAProduct{SKU: summary.SellerSku, ProductName: summary.ProductName}
			}
		}
		log.Trace().Int("page", i).Int("map_length", len(asinSkuMap)).Msg("building map...")
		if result.Payload == nil || result.Payload.Pagination == nil {
			break
		}
		nextToken = &result.Payload.Pagination.NextToken
		if *nextToken == "" {
			break
		}
		i++
	}

	if data, err := json.Marshal(asinSkuMap); err == nil {
		if err := os.WriteFile(cacheFilePath, data, 0644); err != nil {
			log.Warn().Err(err).Msg("Failed to save ASIN to MSKU mapping cache")
		} else {
			log.Info().Int("total_products", len(asinSkuMap)).Str("saved_to", cacheFilePath).Msgf("built cache")
		}
	}

	return asinSkuMap, nil
}
