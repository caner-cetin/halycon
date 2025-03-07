package cmd

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/catalog/client/catalog"
	"github.com/go-openapi/strfmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type LookupAsinFromUpcConfig struct {
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
	lookupAsinFromUpcConfig = LookupAsinFromUpcConfig{}
)

func getlookupAsinFromUpcCmd() *cobra.Command {
	lookupAsinFromUpcCmd.PersistentFlags().BoolVarP(&lookupAsinFromUpcConfig.Single, "single", "s", false, "query single UPC, if given, upc flag must be the product upc, not the file")
	lookupAsinFromUpcCmd.PersistentFlags().StringVarP(&lookupAsinFromUpcConfig.Input, "input", "i", "", "newline delimited (one per line) text file that contains UPCs, or, a single UPC (if so, --single flag must be provided)")
	lookupAsinFromUpcCmd.PersistentFlags().StringVarP(&lookupAsinFromUpcConfig.Output, "output", "o", "", "output for ASIN list, not required when single UPC is queried")
	return lookupAsinFromUpcCmd
}

func lookupAsinFromUpc(cmd *cobra.Command, args []string) {
	if lookupAsinFromUpcConfig.Input == "" {
		cmd.Help()
		log.Fatal().Msg("[--upc / -u] flag is required")
	}
	var ctx = cmd.Context()
	var app = ctx.Value(internal.APP_CONTEXT).(AppCtx)

	var identifiersType = "UPC"
	var queryIdentifiers []string
	if lookupAsinFromUpcConfig.Single {
		queryIdentifiers = append(queryIdentifiers, strings.TrimSpace(lookupAsinFromUpcConfig.Input))
	} else {
		contents, err := os.ReadFile(lookupAsinFromUpcConfig.Input)
		if err != nil {
			if os.IsNotExist(err) {
				log.Fatal().Err(err).Msgf("path %s does not exist", lookupAsinFromUpcConfig.Input)
			}
			log.Fatal().Err(err).Msg("unknown error while reading contents of text file")
		}
		scanner := bufio.NewScanner(bytes.NewBuffer(contents))
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			log.Trace().Bytes("upc", scanner.Bytes()).Msg("reading")
			queryIdentifiers = append(queryIdentifiers, strings.TrimSpace(string(scanner.Bytes())))
		}
	}

	params := catalog.NewSearchCatalogItemsParams()
	params.SetContext(ctx)
	params.SetMarketplaceIds(viper.GetStringSlice(internal.CONFIG_KEY_AMAZON_MARKETPLACE_ID))
	params.SetIdentifiersType(&identifiersType)
	params.SetIdentifiers(queryIdentifiers)
	params.SetIncludedData([]string{"identifiers", "attributes", "summaries"})
	result, err := app.Amazon.Client.SearchCatalogItems(params)
	if err != nil {
		log.Fatal().Err(err).Msg("error while searching catalog items")
	}
	if result.Payload == nil || len(result.Payload.Items) == 0 {
		log.Warn().Msg("no items found for given UPC or UPC list")
		return
	}
	if lookupAsinFromUpcConfig.Single {
		item := result.Payload.Items[0]
		if err := item.Asin.Validate(strfmt.Default); err != nil {
			log.Fatal().Err(err).Msg("cannot validate the ASIN string")
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
		output_tmp, err := os.CreateTemp(os.TempDir(), "halcyon-upc-to-asin-output-*.txt")
		if err != nil {
			log.Fatal().Err(err).Msg("error while creating temporary output file")
		}
		defer output_tmp.Close()
		defer os.Remove(output_tmp.Name())
		writer := bufio.NewWriter(output_tmp)

		var found []string
		for _, item := range result.Payload.Items {
			ev := log.With().Str("asin", string(*item.Asin)).Logger()
			if err := item.Asin.Validate(strfmt.Default); err != nil {
				ev.Fatal().Err(err).Msg("cannot validate the ASIN string")
			}
			if err := item.Identifiers.Validate(strfmt.Default); err != nil {
				ev.Fatal().Err(err).Msg("cannot validate identifiers")
			}
			ev.Trace().Msg("writing")
			writer.WriteString(string(*item.Asin) + "\n")

			for _, id := range item.Identifiers {
				for _, identifier := range id.Identifiers {
					if *identifier.IdentifierType == "UPC" {
						found = append(found, *identifier.Identifier)
					}
				}
			}
		}

		writer.Flush()
		_, err = output_tmp.Seek(0, io.SeekStart)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		output, err := os.Create(lookupAsinFromUpcConfig.Output)
		if err != nil {
			log.Fatal().Err(err).Msg("error while creating output file")
		}
		defer output.Close()
		_, err = io.Copy(output, output_tmp)
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		var not_found []string
		for _, queryIdentifier := range queryIdentifiers {
			if !slices.Contains(found, queryIdentifier) {
				not_found = append(not_found, queryIdentifier)
			}
		}
		if len(not_found) != 0 {
			log.Warn().Interface("upc", not_found).Msg("queries with no results")
		}
	}
}
