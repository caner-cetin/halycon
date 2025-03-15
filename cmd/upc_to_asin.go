package cmd

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/catalog/client/catalog"
	"github.com/caner-cetin/halycon/internal/amazon/catalog/models"
	"github.com/caner-cetin/halycon/internal/config"
	"github.com/go-openapi/strfmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type lookupAsinFromUpcConfig struct {
	// to query a single upc
	Single bool
	// may be a single Input string or a path to the text file
	// that contains UPCs depending on Single flag
	Input string
	// path to output text file
	Output string
}

var (
	lookupAsinFromUpcCmd = &cobra.Command{
		Use:   "upc-to-asin",
		Short: "generates ASIN list from list of UPCs or a single upc",
		Run:   WrapCommandWithResources(lookupAsinFromUpc, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceCatalog, ServiceFBAInventory}}),
	}
	lookupAsinFromUpcCfg = lookupAsinFromUpcConfig{}
)

func getlookupAsinFromUpcCmd() *cobra.Command {
	lookupAsinFromUpcCmd.PersistentFlags().BoolVarP(&lookupAsinFromUpcCfg.Single, "single", "s", false, "query single UPC, if given, upc flag must be the product upc, not the file")
	lookupAsinFromUpcCmd.PersistentFlags().StringVarP(&lookupAsinFromUpcCfg.Input, "input", "i", "", "newline delimited (one per line) text file that contains UPCs, or, a single UPC (if so, --single flag must be provided)")
	lookupAsinFromUpcCmd.PersistentFlags().StringVarP(&lookupAsinFromUpcCfg.Output, "output", "o", "", "output for ASIN list, not required when single UPC is queried")
	return lookupAsinFromUpcCmd
}

func lookupAsinFromUpc(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)

	var queryIdentifiers []string
	if lookupAsinFromUpcCfg.Single {
		queryIdentifiers = append(queryIdentifiers, strings.TrimSpace(lookupAsinFromUpcCfg.Input))
	} else {
		contents, err := os.ReadFile(lookupAsinFromUpcCfg.Input)
		if err != nil {
			ev := log.With().Str("path", lookupAsinFromUpcCfg.Input).Err(err).Logger()
			if os.IsNotExist(err) {
				ev.Fatal().Msg("path does not exist")
			}
			ev.Fatal().Msg("unknown error while opening file")
		}
		scanner := bufio.NewScanner(bytes.NewBuffer(contents))
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			line := scanner.Text()
			cleaned := internal.RemoveAllNonDigit(line)
			log.Trace().Str("original", line).Str("cleaned", cleaned).Msg("reading")
			queryIdentifiers = append(queryIdentifiers, cleaned)
		}
	}
	batchCount := (len(queryIdentifiers) + 9) / 10
	results := make([]*models.ItemSearchResults, 0, batchCount)
	for identifiers := range slices.Chunk(queryIdentifiers, 10) {
		log.Trace().Interface("identifiers", identifiers).Msg("searching next batch")
		params := catalog.NewSearchCatalogItemsParams()
		params.SetContext(cmd.Context())
		params.SetMarketplaceIds(viper.GetStringSlice(config.AMAZON_MARKETPLACE_ID.Key))
		params.SetIdentifiersType(ptr.String("UPC"))
		params.SetIdentifiers(identifiers)
		params.SetIncludedData([]string{"identifiers", "attributes", "summaries"})
		result, err := app.Amazon.Client.SearchCatalogItems(params)
		if err != nil {
			log.Error().Err(err).Msg("error while searching catalog items")
			return
		}
		if result.Payload == nil || len(result.Payload.Items) == 0 {
			log.Warn().Interface("batch", identifiers).Msg("no items found for this batch of UPCs")
			continue
		}
		results = append(results, result.Payload)
	}
	if lookupAsinFromUpcCfg.Single {
		item := results[0].Items[0]
		if err := item.Asin.Validate(strfmt.Default); err != nil {
			log.Error().Err(err).Msg("cannot validate the ASIN string")
			return
		}
		ev := log.Info()
		for _, mplace := range item.Identifiers {
			for _, identifier := range mplace.Identifiers {
				ev.Str(*identifier.IdentifierType, *identifier.Identifier)
			}
		}
		ev.
			Str("ASIN", string(*item.Asin)).
			Msg("found!")
	} else {
		output_tmp, err := os.CreateTemp(os.TempDir(), "halycon-upc-to-asin-output-*.txt")
		if err != nil {
			log.Error().Err(err).Msg("error while creating temporary output file")
			return
		}
		defer output_tmp.Close()
		defer os.Remove(output_tmp.Name())
		writer := bufio.NewWriter(output_tmp)

		upcToAsin := make(map[string]string)

		for _, result := range results {
			for _, item := range result.Items {
				ev := log.With().Str("asin", string(*item.Asin)).Logger()
				if err := item.Asin.Validate(strfmt.Default); err != nil {
					ev.Fatal().Err(err).Msg("cannot validate the ASIN string")
				}
				if err := item.Identifiers.Validate(strfmt.Default); err != nil {
					ev.Fatal().Err(err).Msg("cannot validate identifiers")
				}
				ev.Trace().Msg("writing")

				for _, id := range item.Identifiers {
					for _, identifier := range id.Identifiers {
						if *identifier.IdentifierType == "UPC" {
							upc := *identifier.Identifier
							upcToAsin[upc] = string(*item.Asin)
							break
						}
					}
				}
			}
		}

		for _, upc := range queryIdentifiers {
			if asin, found := upcToAsin[upc]; found {
				log.Trace().Str("upc", upc).Str("asin", asin).Msg("writing matched pair")
				_, err := writer.WriteString(asin + "\n")
				if err != nil {
					log.Err(err).Str("asin", asin).Str("file_path", output_tmp.Name()).Msg("error writing asin")
					return
				}
			} else {
				log.Warn().Str("upc", upc).Msg("no ASIN found for UPC")
			}
		}
		err = writer.Flush()
		if err != nil {
			log.Error().Err(err).Send()
		}

		_, err = output_tmp.Seek(0, io.SeekStart)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
		output, err := os.Create(lookupAsinFromUpcCfg.Output)
		if err != nil {
			log.Error().Err(err).Msg("error while creating output file")
			return
		}
		defer output.Close()
		_, err = io.Copy(output, output_tmp)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
	}
}
