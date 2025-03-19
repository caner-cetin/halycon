package cmd

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/product_type_definitions"
	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/valyala/fastjson"
)

type getProductTypeDefinitionConfig struct {
	Params      product_type_definitions.GetDefinitionsProductTypeParams
	ProductType string
}

var (
	searchProductTypeDefinitionCmd = &cobra.Command{
		Use: "search",
		Run: WrapCommandWithResources(searchProductTypeDefinition, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceProductTypeDefinitions}}),
	}
	searchProductTypeDefinitionCfg   product_type_definitions.SearchDefinitionsProductTypesParams
	getProductTypeDefinitionDetailed bool
	getProductTypeDefinitionCmd      = &cobra.Command{
		Use: "get",
		Run: WrapCommandWithResources(getProductTypeDefinition, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceProductTypeDefinitions}}),
	}
	getProductTypeDefinitionCfg getProductTypeDefinitionConfig

	definitionCmd = &cobra.Command{
		Use: "definition",
	}
)

func getDefinitionsCmd() *cobra.Command {
	searchProductTypeDefinitionCfg.ItemName = internal.Ptr("")
	searchProductTypeDefinitionCfg.Keywords = internal.Ptr([]string{})
	searchProductTypeDefinitionCmd.PersistentFlags().StringArrayVar(
		searchProductTypeDefinitionCfg.Keywords,
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
	getProductTypeDefinitionCmd.PersistentFlags().BoolVar(&getProductTypeDefinitionDetailed, "detailed", false, "complete property information")

	definitionCmd.AddCommand(searchProductTypeDefinitionCmd)
	definitionCmd.AddCommand(getProductTypeDefinitionCmd)
	return definitionCmd
}

func searchProductTypeDefinition(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	var keywords_set = len(*searchProductTypeDefinitionCfg.Keywords) != 0
	var item_name_set = *searchProductTypeDefinitionCfg.ItemName != ""
	if keywords_set && item_name_set {
		log.Error().Msg("keywords and item name cannot be used together")
		return
	}
	if !keywords_set && !item_name_set {
		log.Error().Msg("keywords or item name must be set")
		return
	}
	searchProductTypeDefinitionCfg.MarketplaceIds = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID
	status, err := app.Amazon.Client.SearchProductTypeDefinitions(cmd.Context(), &searchProductTypeDefinitionCfg)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	result := status.JSON200
	fmt.Printf("%-40s %-40s\n", "Display Name", "Amazon Name")
	fmt.Println(strings.Repeat("-", 80))
	for _, ptype := range result.ProductTypes {
		fmt.Printf("%-40s %-40s\n", ptype.DisplayName, ptype.Name)
		fmt.Println(strings.Repeat("-", 80))
	}
}
func getProductTypeDefinition(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	getProductTypeDefinitionCfg.Params.MarketplaceIds = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID
	status, err := app.Amazon.Client.GetProductTypeDefinition(cmd.Context(), getProductTypeDefinitionCfg.ProductType, &getProductTypeDefinitionCfg.Params)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	result := status.JSON200
	displayProductSummary(result)
	schemaURL := result.Schema.Link.Resource

	schema, err := fetchAndParseSchema(schemaURL)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}

	displayAllSchemaDetails(schema, 0)
}
func displayProductSummary(payload *product_type_definitions.ProductTypeDefinition) {
	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("%s: %s  |  %s: %s  |  %s: %s | %s: %s\n\n",
		bold("Product"), cyan(payload.DisplayName),
		bold("Requirements"), yellow(payload.Requirements),
		bold("Locale"), green(payload.Locale),
		bold("Schema"), cyan(payload.Schema.Link.Resource),
	)
}

func fetchAndParseSchema(schemaURL string) (*fastjson.Value, error) {
	resp, err := http.DefaultClient.Get(schemaURL) //nolint:bodyclose
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schema from %s: %w", schemaURL, err)
	}
	defer internal.CloseResponseBody(resp)
	schema_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Str("url", schemaURL).Err(err).Msg("error while reading schema")
		return nil, fmt.Errorf("failed to read schema body: %w", err)
	}
	return fastjson.MustParseBytes(schema_bytes), nil
}

func displayAllSchemaDetails(schema *fastjson.Value, indentLevel int) {
	indent := strings.Repeat("  ", indentLevel)
	bold := color.New(color.Bold).SprintFunc()

	// Print type-agnostic attributes
	for _, key := range []string{"type", "description", "title"} {
		if schema.Exists(key) {
			fmt.Printf("%s%s: %s\n", indent, bold(key), string(schema.Get(key).MarshalTo(nil)))
		}
	}
	// Iterate and print the constrains
	displayPropertyConstraintsAll(schema, indent, bold)
	requiredValues := schema.GetArray("required")
	var requiredKeys = make([]string, 0, len(requiredValues))
	for _, required := range requiredValues {
		requiredKeys = append(requiredKeys, string(required.GetStringBytes()))
	}

	// Recursively handle "properties"
	properties := schema.GetObject("properties")
	if properties != nil {
		fmt.Printf("%sProperties:\n", indent)
		properties.Visit(func(key []byte, value *fastjson.Value) {
			k := string(key)
			if slices.Contains(requiredKeys, k) {
				fmt.Printf("%s* %s:\n", indent, string(key))
			} else {
				fmt.Printf("%s  %s:\n", indent, string(key))
			}
			displayAllSchemaDetails(value, indentLevel+2) // Recurse!
		})
	}

	// Handle "items" for array types
	items := schema.Get("items")
	if items != nil && items.Type() == fastjson.TypeObject {
		fmt.Printf("%sItems:\n", indent)
		displayAllSchemaDetails(items, indentLevel+2) // Recurse!
	}
	oneOfs := schema.GetArray("oneOf")
	if len(oneOfs) > 0 {
		fmt.Printf("%sOneOf:\n", indent)
		for _, oneOf := range oneOfs {
			displayAllSchemaDetails(oneOf, indentLevel+2)
		}
	}
	// this seems unnecessary
	// allOfs := schema.GetArray("allOf")
	// if len(allOfs) > 0 {
	// 	fmt.Printf("%sAllOf:\n", indent)
	// 	for _, allOf := range allOfs {
	// 		displayAllSchemaDetails(allOf, indentLevel+2)
	// 	}
	// }
	// anyOfs := schema.GetArray("anyOf")
	// if len(anyOfs) > 0 {
	// 	fmt.Printf("%sAnyOf:\n", indent)
	// 	for _, anyOf := range anyOfs {
	// 		displayAllSchemaDetails(anyOf, indentLevel+2)
	// 	}
	// }
	if schema.Exists("not") {
		fmt.Printf("%sNot:\n", indent)
		displayAllSchemaDetails(schema.Get("not"), indentLevel+2)
	}
}

func displayPropertyConstraintsAll(prop *fastjson.Value, indent string, dim func(a ...interface{}) string) {
	minLength := prop.GetInt("minLength")
	maxLength := prop.GetInt("maxLength")
	minimum := prop.GetFloat64("minimum")
	maximum := prop.GetFloat64("maximum")
	multipleOf := prop.GetFloat64("multipleOf")
	format := ""
	formatBytes := prop.GetStringBytes("format")
	if formatBytes != nil {
		format = string(formatBytes)
	}
	unique := prop.GetBool("uniqueItems")

	constraints := []string{}

	if minLength > 0 {
		constraints = append(constraints, fmt.Sprintf("Min Length: %d", minLength))
	}
	if maxLength > 0 {
		constraints = append(constraints, fmt.Sprintf("Max Length: %d", maxLength))
	}
	if prop.Exists("minimum") {
		constraints = append(constraints, fmt.Sprintf("Minimum: %f", minimum))
	}
	if prop.Exists("maximum") {
		constraints = append(constraints, fmt.Sprintf("Maximum: %f", maximum))
	}
	if prop.Exists("multipleOf") {
		constraints = append(constraints, fmt.Sprintf("Multiple Of: %f", multipleOf))
	}
	if format != "" {
		constraints = append(constraints, fmt.Sprintf("Format: %s", format))
	}
	if unique {
		constraints = append(constraints, "Unique Items")
	}

	if len(constraints) > 0 {
		fmt.Printf("%s%s %s\n", indent, dim("Constraints:"), strings.Join(constraints, ", "))
	}
}
