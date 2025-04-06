package cmd

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/csv"
	"io"
	"os"
	"strings"

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

type FBAProduct struct {
	SKU   string
	Title string
}

var (
	lookupSkuFromAsinCmd = &cobra.Command{
		Use: "asin-to-sku",
		Run: WrapCommandWithResources(lookupSkuFromAsin, ResourceConfig{Resources: []ResourceType{ResourceAmazon, ResourceDB}, Services: []ServiceType{ServiceFBAInventory}}),
	}
	lookupSkuFromAsinCfg = lookupSkuFromAsinConfig{}
)

func getLookupSkuFromAsinCmd() *cobra.Command {
	lookupSkuFromAsinCmd.PersistentFlags().BoolVarP(&lookupSkuFromAsinCfg.Single, "single", "s", false, "query single ASIN, if given, [--asin/-a] flag must be the product ASIN, not the file")
	lookupSkuFromAsinCmd.PersistentFlags().StringVarP(&lookupSkuFromAsinCfg.Input, "input", "i", "", "newline delimited (one per line) text file that contains ASINs, or, a single ASIN (if so, --single flag must be provided)")
	lookupSkuFromAsinCmd.PersistentFlags().StringVarP(&lookupSkuFromAsinCfg.Output, "output", "o", "", "output for SKU list (*.csv), not required when single ASIN is queried")
	return lookupSkuFromAsinCmd
}

func lookupSkuFromAsin(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	var err error
	if lookupSkuFromAsinCfg.Single {
		product, err := app.Query.GetFBAProductFromAsin(cmd.Context(), sql.NullString{String: lookupAsinFromUpcCfg.Input, Valid: true})
		if err != nil {
			log.Error().Str("asin", lookupSkuFromAsinCfg.Input).Err(err).Msg("failed to lookup product")
			return
		}
		if !product.Title.Valid {
			log.Error().Str("asin", lookupSkuFromAsinCfg.Input).Msg("product not found")
			return
		}
		log.Info().Str("title", product.Title.String).Str("sku", product.Sku.String).Msg("found")
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

		productMap, err := app.buildAsinToSkuMap()
		if err != nil {
			log.Error().Err(err).Msg("failed to build asin to sku map")
			return
		}
		for _, asin := range asins {
			product, ok := productMap[asin]
			if !ok {
				log.Warn().Str("asin", asin).Msg("cannot find the product")
			}
			err = writer.Write([]string{asin, product.SKU, product.Title, ""})
			if err != nil {
				log.Error().
					Err(err).
					Str("asin", asin).
					Str("sku", product.SKU).
					Str("title", product.Title).
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

func (a *AppCtx) buildAsinToSkuMap() (map[string]FBAProduct, error) {
	var atsMap = make(map[string]FBAProduct)
	contents, err := a.Query.GetAsinToSkuMapContents(a.Ctx)
	if err != nil {
		return nil, err
	}
	for _, content := range contents {
		atsMap[content.Asin.String] = FBAProduct{SKU: content.Sku.String, Title: content.Title.String}
	}
	return atsMap, nil
}
