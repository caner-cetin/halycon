package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inventory"
	sp_api "github.com/caner-cetin/halycon/internal/sp-api"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type lookupSkuFromAsinConfig struct {
	ForceRebuildCache bool
	Single            bool
	// may be a single Input string or a path to the text file
	// that contains ASINs depending on Single flag
	Input  string
	Output string
}

// FBAProduct represents a product in the Fulfillment by Amazon (FBA) system.
// It contains the product's unique stock keeping unit (SKU) and name.
type FBAProduct struct {
	SKU         string
	ProductName string
}

var (
	lookupSkuFromAsinCmd = &cobra.Command{
		Use: "asin-to-sku",
		Run: WrapCommandWithResources(lookupSkuFromAsin, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceFBAInventory}}),
	}
	lookupSkuFromAsinCfg = lookupSkuFromAsinConfig{}
)

func getLookupSkuFromAsinCmd() *cobra.Command {
	lookupSkuFromAsinCmd.PersistentFlags().BoolVarP(&lookupSkuFromAsinCfg.Single, "single", "s", false, "query single ASIN, if given, [--asin/-a] flag must be the product ASIN, not the file")
	lookupSkuFromAsinCmd.PersistentFlags().StringVarP(&lookupSkuFromAsinCfg.Input, "input", "i", "", "newline delimited (one per line) text file that contains ASINs, or, a single ASIN (if so, --single flag must be provided)")
	lookupSkuFromAsinCmd.PersistentFlags().BoolVar(&lookupSkuFromAsinCfg.ForceRebuildCache, "force-rebuild", false, "forces rebuilding the ASIN / SKU mapping cache")
	lookupSkuFromAsinCmd.PersistentFlags().StringVarP(&lookupSkuFromAsinCfg.Output, "output", "o", "", "output for SKU list (*.csv), not required when single ASIN is queried")
	return lookupSkuFromAsinCmd
}

func lookupSkuFromAsin(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	productMap, err := getAsinToMskuMap(app.Amazon.Client)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	if lookupSkuFromAsinCfg.Single {
		product, ok := productMap[lookupSkuFromAsinCfg.Input]
		if !ok {
			log.Error().Str("asin", lookupSkuFromAsinCfg.Input).Msg("product not found")
			return
		}
		log.Info().Str("name", product.ProductName).Str("sku", product.SKU).Msg("found")
	} else {
		var input []byte
		if input, err = os.ReadFile(lookupSkuFromAsinCfg.Input); err != nil {
			if os.IsNotExist(err) {
				log.Error().Err(err).Msgf("path %s does not exist", lookupAsinFromUpcCfg.Input)
				return
			}
			log.Error().Err(err).Msg("unknown error while reading contents of text file")
			return
		}
		scanner := bufio.NewScanner(bytes.NewReader(input))
		scanner.Split(bufio.ScanLines)
		var asins []string
		for scanner.Scan() {
			asins = append(asins, strings.TrimSpace(scanner.Text()))
		}

		output_tmp, err := os.CreateTemp(os.TempDir(), "halycon-asin-to-sku-output-*.csv")
		if err != nil {
			log.Error().Err(err).Msg("error while creating temporary output file")
		}
		defer output_tmp.Close()
		defer os.Remove(output_tmp.Name())

		writer := csv.NewWriter(output_tmp)
		err = writer.Write([]string{"ASIN", "SKU", "Product Name", "Quantity"})
		if err != nil {
			log.Error().
				Err(err).
				Msg("error while writing column names")
			return
		}
		for _, asin := range asins {
			product, ok := productMap[asin]
			if !ok {
				log.Warn().Str("asin", asin).Msg("cannot find a product")
			}
			err = writer.Write([]string{asin, product.SKU, product.ProductName, ""})
			if err != nil {
				log.Error().
					Err(err).
					Str("asin", asin).
					Str("sku", product.SKU).
					Str("name", product.ProductName).
					Msg("error while writing row")
				return
			}
		}
		writer.Flush()

		output, err := os.Create(lookupSkuFromAsinCfg.Output)
		if err != nil {
			log.Error().Err(err).Msg("error while creating the output file")
			return
		}
		_, err = output_tmp.Seek(0, io.SeekStart)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
		_, err = io.Copy(output, output_tmp)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
		log.Info().Str("file", lookupSkuFromAsinCfg.Output).Msg("saved csv")
	}

}

func getAsinToMskuMap(amazonClient *sp_api.Client) (map[string]FBAProduct, error) {
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
		params.MarketplaceIds = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID
		params.GranularityType = "Marketplace"
		params.GranularityId = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID[0]
		params.Details = internal.Ptr(true)

		if nextToken != nil {
			params.NextToken = nextToken
		}

		status, err := amazonClient.GetFBAInventorySummaries(context.TODO(), &params)
		if err != nil {
			return nil, err
		}
		result := status.JSON200

		for _, summary := range result.Payload.InventorySummaries {
			if summary.Asin != nil && *summary.Asin != "" {
				asinSkuMap[*summary.Asin] = FBAProduct{SKU: *summary.SellerSku, ProductName: *summary.ProductName}
			}
		}
		log.Trace().Int("page", i).Int("map_length", len(asinSkuMap)).Msg("building map...")
		if result.Payload == nil || result.Pagination == nil {
			break
		}
		nextToken = result.Pagination.NextToken
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
