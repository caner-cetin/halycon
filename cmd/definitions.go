package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"github.com/caner-cetin/halycon/internal/amazon/product_type_definitions/client/definitions"
	"github.com/caner-cetin/halycon/internal/amazon/product_type_definitions/models"
	"github.com/caner-cetin/halycon/internal/config"
	"github.com/fatih/color"
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
	searchProductTypeDefinitionCfg   definitions.SearchDefinitionsProductTypesParams
	getProductTypeDefinitionDetailed bool
	getProductTypeDefinitionCmd      = &cobra.Command{
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
	getProductTypeDefinitionCmd.PersistentFlags().BoolVar(&getProductTypeDefinitionDetailed, "detailed", false, "complete property information")

	definitionCmd.AddCommand(searchProductTypeDefinitionCmd)
	definitionCmd.AddCommand(getProductTypeDefinitionCmd)
	return definitionCmd
}

func searchProductTypeDefinition(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	var keywords_set = len(searchProductTypeDefinitionCfg.Keywords) != 0
	var item_name_set = *searchProductTypeDefinitionCfg.ItemName != ""
	if keywords_set && item_name_set {
		log.Fatal().Msg("keywords and item name cannot be used together")
	}
	if !keywords_set && !item_name_set {
		log.Fatal().Msg("keywords or item name must be set")
	}
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
	app := GetApp(cmd)
	getProductTypeDefinitionCfg.MarketplaceIds = viper.GetStringSlice(config.AMAZON_MARKETPLACE_ID.Key)

	result, err := app.Amazon.Client.GetProductTypeDefinition(&getProductTypeDefinitionCfg)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	detailedMode := getProductTypeDefinitionDetailed
	displayProductSummary(result.Payload)
	if getProductTypeDefinitionDetailed {
		schema := fetchAndParseSchema(*result.Payload.Schema.Link.Resource)
		requiredProps := getRequiredProperties(schema)
		displayProperties(schema, requiredProps, detailedMode)
	}
}

func displayProductSummary(payload *models.ProductTypeDefinition) {
	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("%s: %s  |  %s: %s  |  %s: %s | %s: %s\n\n",
		bold("Product"), cyan(*payload.DisplayName),
		bold("Requirements"), yellow(*payload.Requirements),
		bold("Locale"), green(*payload.Locale),
		bold("Schema"), cyan(*payload.Schema.Link.Resource),
	)
}

func fetchAndParseSchema(schemaURL string) *fastjson.Value {
	resp, err := http.DefaultClient.Get(schemaURL)
	if err != nil {
		log.Fatal().Str("url", schemaURL).Err(err).Msg("error while querying schema")
	}
	schema_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().Str("url", schemaURL).Err(err).Msg("error while reading schema")
	}
	return fastjson.MustParseBytes(schema_bytes)
}

func getRequiredProperties(schema *fastjson.Value) map[string]bool {
	requiredProps := make(map[string]bool)
	for _, prop := range schema.GetArray("required") {
		prop_name, err := strconv.Unquote(string(prop.MarshalTo(nil)))
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		requiredProps[prop_name] = true
	}
	return requiredProps
}

func displayProperties(schema *fastjson.Value, requiredProps map[string]bool, detailedMode bool) {
	bold := color.New(color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	properties := schema.GetObject("properties")

	fmt.Println(bold("Properties:"))
	fmt.Println(strings.Repeat("-", 40))

	i := 0
	properties.Visit(func(key []byte, v *fastjson.Value) {
		propName := string(key)
		description := string(v.GetStringBytes("description"))
		propType := string(v.GetStringBytes("type"))

		reqMarker := " "
		if requiredProps[propName] {
			reqMarker = "*"
		}

		if len(description) > 50 && !detailedMode {
			description = description[:47] + "..."
		}

		fmt.Printf("%s %s (%s) - %s\n",
			yellow(reqMarker),
			bold(propName),
			cyan(propType),
			description)

		if detailedMode {
			displayPropertyDetails(v, dim, green)
		}

		i++
		if !detailedMode && i%5 == 0 {
			fmt.Println(dim(strings.Repeat("-", 40)))
		}
	})

	displayFooterTips(detailedMode, dim, yellow)
}

func displayPropertyDetails(prop *fastjson.Value, dim, green func(a ...interface{}) string) {
	indent := "    "

	displayPropertyConstraints(prop, indent, dim)
	displayPropertyExamples(prop, indent, dim, green)
	displayPropertyEnums(prop, indent, dim, green, true)
	displayPropertySelectors(prop, indent)
	displayRequiredKeys(prop, indent, dim, green)

	fmt.Println()
}

func displayPropertyConstraints(prop *fastjson.Value, indent string, dim func(a ...interface{}) string) {
	minItems := prop.GetInt("minUniqueItems")
	maxItems := prop.GetInt("maxUniqueItems")
	if minItems > 0 || maxItems > 0 {
		fmt.Printf("%s%s Min: %d, Max: %d items\n",
			indent, dim("Constraints:"), minItems, maxItems)
	}
}

func displayPropertyExamples(prop *fastjson.Value, indent string, dim, green func(a ...interface{}) string) {
	examples := prop.GetArray("examples")
	if len(examples) > 0 {
		fmt.Printf("%s%s ", indent, dim("Examples:"))
		for j, example := range examples {
			if j < 2 {
				fmt.Printf("%s  ", green(example))
			} else if j == 2 {
				fmt.Printf("...")
				break
			}
		}
		fmt.Println()
	}
}

func displayPropertyEnums(prop *fastjson.Value, indent string, dim, green func(a ...interface{}) string, isDetailed bool) {
	enums := prop.GetArray("items", "properties", "value", "enum")
	if len(enums) > 0 {
		enum_names := prop.GetArray("items", "properties", "value", "enumNames")
		fmt.Printf("%s%s ", indent, dim("Options:"))

		var displayCount int
		if isDetailed {
			displayCount = len(enums)
		} else {
			displayCount = min(3, len(enums))
		}

		for j := range displayCount {
			fmt.Printf("%s (%s)  ",
				green(enums[j]),
				dim(enum_names[j]))
		}

		if len(enums) > 3 && !isDetailed {
			fmt.Printf("... %d more", len(enums)-3)
		}

		fmt.Println()
	}
}

func displayPropertySelectors(prop *fastjson.Value, indent string) {
	selectors := prop.GetArray("selectors")
	if len(selectors) > 0 {
		selector_strings := []string{}
		for _, selector := range selectors {
			selector_strings = append(selector_strings, string(selector.GetStringBytes()))
		}
		fmt.Printf("%sRequired Selectors: %s\n", indent, strings.Join(selector_strings, ","))
	}
}

func displayRequiredKeys(prop *fastjson.Value, indent string, dim, green func(a ...interface{}) string) {
	required_keys := prop.GetArray("items", "required")
	if len(required_keys) == 0 {
		return
	}

	required_key_strings := []string{}
	for _, key := range required_keys {
		required_key_strings = append(required_key_strings, string(key.GetStringBytes()))
	}

	fmt.Printf("%sRequired Keys: %s\n", indent, strings.Join(required_key_strings, ","))

	for _, required_key := range required_key_strings {
		displayRequiredKeyDetails(prop, required_key, indent, dim, green)
	}

	fmt.Println()
}

func displayRequiredKeyDetails(prop *fastjson.Value, required_key, indent string, dim, green func(a ...interface{}) string) {
	obj := prop.GetObject("items", "properties", required_key)
	if obj == nil {
		return
	}

	fmt.Printf("%s%s\n", strings.Repeat(indent, 2), required_key)
	fmt.Printf("%sTitle: %s\n", strings.Repeat(indent, 3), string(obj.Get("title").GetStringBytes()))
	fmt.Printf("%sDescription: %s\n", strings.Repeat(indent, 3), string(obj.Get("description").GetStringBytes()))

	// display enumerations and required keys if present
	displayNestedEnums(obj, indent, 3, dim, green)
	displayNestedRequiredKeys(prop, required_key, obj, indent, dim, green)
}

func displayNestedEnums(obj *fastjson.Object, indent string, indentLevel int, dim, green func(a ...interface{}) string) {
	enums := obj.Get("enum").GetArray()
	if len(enums) == 0 {
		return
	}

	fmt.Printf("%sEnums:\n", strings.Repeat(indent, indentLevel))
	enum_name_vals := obj.Get("enumNames").GetArray()

	for i := range enum_name_vals {
		fmt.Printf("%s%s (%s)\n",
			strings.Repeat(indent, indentLevel+1),
			enums[i].GetStringBytes(),
			enum_name_vals[i].GetStringBytes())
	}
}

func displayNestedRequiredKeys(prop *fastjson.Value, parent_key string, obj *fastjson.Object, indent string, dim, green func(a ...interface{}) string) {
	obj_required := obj.Get("required").GetArray()
	if len(obj_required) == 0 {
		return
	}

	obj_keys := []string{}
	for j := range obj_required {
		obj_keys = append(obj_keys, string(obj_required[j].GetStringBytes()))
	}

	fmt.Printf("%sRequired Keys: %s\n", strings.Repeat(indent, 3), strings.Join(obj_keys, ","))

	for _, key := range obj_keys {
		displayNestedKeyDetails(prop, parent_key, key, indent, dim, green)
	}
}

func displayNestedKeyDetails(prop *fastjson.Value, parent_key, key, indent string, dim, green func(a ...interface{}) string) {
	fmt.Printf("%s%s\n", strings.Repeat(indent, 4), key)

	obj_detail := prop.GetObject("items", "properties", parent_key, "properties", key)

	fmt.Printf("%sTitle: %s\n", strings.Repeat(indent, 5), string(obj_detail.Get("title").GetStringBytes()))
	fmt.Printf("%sDescription: %s\n", strings.Repeat(indent, 5), string(obj_detail.Get("description").GetStringBytes()))

	enums := obj_detail.Get("enum").GetArray()
	if len(enums) > 0 {
		fmt.Printf("%sEnums:\n", strings.Repeat(indent, 5))
		enum_name_vals := obj_detail.Get("enumNames").GetArray()

		for i := range enum_name_vals {
			fmt.Printf("%s%s (%s)\n",
				strings.Repeat(indent, 6),
				enums[i].GetStringBytes(),
				enum_name_vals[i].GetStringBytes())
		}
	}
}

func displayFooterTips(detailedMode bool, dim, yellow func(a ...interface{}) string) {
	if !detailedMode {
		fmt.Printf("\n%s use --detailed flag for complete property information\n",
			dim("Tip:"))
	}

	fmt.Printf("%s Properties marked with %s are required\n",
		dim("Note:"), yellow("*"))
}
