package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/smithy-go/ptr"
	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/product_type_definitions/client/definitions"
	"github.com/caner-cetin/halycon/internal/config"
	"github.com/jedib0t/go-pretty/v6/list"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/valyala/fastjson"
)

var (
	searchProductTypeDefinitionCmd = &cobra.Command{
		Use: "search",
		Run: WrapCommandWithResources(searchProductTypeDefinition, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceProductTypeDefinitions}}),
	}
	searchProductTypeDefinitionCfg definitions.SearchDefinitionsProductTypesParams
	getProductTypeDefinitionCmd    = &cobra.Command{
		Use: "get",
		Run: WrapCommandWithResources(getProductTypeDefinition, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceProductTypeDefinitions}}),
	}
	getProductTypeDefinitionCfg definitions.GetDefinitionsProductTypeParams

	definitionCmd = &cobra.Command{
		Use: "definition",
	}
)

func getDefinitionsCmd() *cobra.Command {
	searchProductTypeDefinitionCfg.ItemName = ptr.String("")
	searchProductTypeDefinitionCmd.PersistentFlags().StringArrayVar(
		&searchProductTypeDefinitionCfg.Keywords,
		"keywords",
		[]string{},
		"A comma-delimited list of keywords to search product types. Note: Cannot be used with itemName.",
	)
	searchProductTypeDefinitionCmd.PersistentFlags().StringVar(
		searchProductTypeDefinitionCfg.ItemName,
		"item",
		"",
		"The title of the ASIN to get the product type recommendation. Note: Cannot be used with keywords",
	)

	getProductTypeDefinitionCmd.PersistentFlags().StringVarP(&getProductTypeDefinitionCfg.ProductType, "type", "t", "", "The Amazon product type name.")

	definitionCmd.AddCommand(searchProductTypeDefinitionCmd)
	definitionCmd.AddCommand(getProductTypeDefinitionCmd)
	return definitionCmd
}

func searchProductTypeDefinition(cmd *cobra.Command, args []string) {
	var keywords_set = len(searchProductTypeDefinitionCfg.Keywords) != 0
	var item_name_set = *searchProductTypeDefinitionCfg.ItemName != ""
	if keywords_set && item_name_set {
		log.Fatal().Msg("keywords and item name cannot be used together")
	}
	if !keywords_set && !item_name_set {
		log.Fatal().Msg("keywords or item name must be set")
	}
	ctx := cmd.Context()
	app := ctx.Value(internal.APP_CONTEXT).(AppCtx)
	searchProductTypeDefinitionCfg.MarketplaceIds = viper.GetStringSlice(config.AMAZON_MARKETPLACE_ID.Key)
	result, err := app.Amazon.Client.SearchProductTypeDefinitions(&searchProductTypeDefinitionCfg)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Display Name", "Amazon Name"})
	for _, ptype := range result.Payload.ProductTypes {
		t.AppendRow(table.Row{*ptype.DisplayName, *ptype.Name})
		t.AppendSeparator()
	}
	t.Render()
}

func getProductTypeDefinition(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	app := ctx.Value(internal.APP_CONTEXT).(AppCtx)
	getProductTypeDefinitionCfg.MarketplaceIds = viper.GetStringSlice(config.AMAZON_MARKETPLACE_ID.Key)
	result, err := app.Amazon.Client.GetProductTypeDefinition(&getProductTypeDefinitionCfg)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	log.Info().
		Str("requirements", *result.Payload.Requirements).
		Str("display_name", *result.Payload.DisplayName).
		Str("locale", *result.Payload.Locale).
		Msg("basic info")
	resp, err := http.DefaultClient.Get(*result.Payload.Schema.Link.Resource)
	if err != nil {
		log.Fatal().Str("url", *result.Payload.Schema.Link.Resource).Err(err).Msg("error while querying schema")
	}
	schema_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().Str("url", *result.Payload.Schema.Link.Resource).Err(err).Msg("error while reading schema")
	}
	schema := fastjson.MustParseBytes(schema_bytes)
	properties := schema.GetObject("properties")

	l := list.NewWriter()
	l.SetOutputMirror(os.Stdout)
	for _, prop := range schema.GetArray("required") {
		prop_name, err := strconv.Unquote(string(prop.MarshalTo(nil)))
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		property := properties.Get(prop_name)
		if property == nil {
			log.Warn().Str("name", prop_name).Msg("prop not found in properties schema")
			continue
		}
		description := string(property.GetStringBytes("description"))
		l.AppendItem(fmt.Sprintf("%s - %s", prop, description))
		l.Indent()
		l.AppendItem("Examples")
		l.Indent()
		for _, example := range property.GetArray("examples") {
			l.AppendItem(example)
		}
		l.UnIndent()
		l.AppendItem(fmt.Sprintf("Type - %s", string(property.GetStringBytes("type"))))
		enums := property.GetArray("items", "properties", "value", "enum")
		if enums != nil {
			enum_names := property.GetArray("items", "properties", "value", "enumNames")
			l.AppendItem("Enums:")
			l.Indent()
			for i, enum := range enums {
				l.AppendItem(fmt.Sprintf("%s - %s", enum, enum_names[i]))
			}
			l.UnIndent()
		}
		l.UnIndent()
	}
	l.Render()
}
