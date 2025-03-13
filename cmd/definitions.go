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

	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	fmt.Printf("%s: %s  |  %s: %s  |  %s: %s\n\n",
		bold("Product"), cyan(*result.Payload.DisplayName),
		bold("Requirements"), yellow(*result.Payload.Requirements),
		bold("Locale"), green(*result.Payload.Locale))

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

	requiredProps := make(map[string]bool)
	for _, prop := range schema.GetArray("required") {
		prop_name, err := strconv.Unquote(string(prop.MarshalTo(nil)))
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		requiredProps[prop_name] = true
	}

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
			indent := "    "

			minItems := v.GetInt("minUniqueItems")
			maxItems := v.GetInt("maxUniqueItems")
			if minItems > 0 || maxItems > 0 {
				fmt.Printf("%s%s Min: %d, Max: %d items\n",
					indent, dim("Constraints:"), minItems, maxItems)
			}

			examples := v.GetArray("examples")
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

			enums := v.GetArray("items", "properties", "value", "enum")
			if len(enums) > 0 {
				enum_names := v.GetArray("items", "properties", "value", "enumNames")
				fmt.Printf("%s%s ", indent, dim("Options:"))
				var displayCount int
				if detailedMode {
					displayCount = len(enums)
				} else {
					displayCount = min(3, len(enums))
				}
				for j := range displayCount {
					fmt.Printf("%s (%s)  ",
						green(enums[j]),
						dim(enum_names[j]))
				}
				if len(enums) > 3 && !detailedMode {
					fmt.Printf("... %d more", len(enums)-3)
				}
				fmt.Println()
			}

			fmt.Println()
		}

		i++
		if !detailedMode && i%5 == 0 {
			fmt.Println(dim(strings.Repeat("-", 40)))
		}
	})

	if !detailedMode {
		fmt.Printf("\n%s use --detailed flag for complete property information\n",
			dim("Tip:"))
	}
	fmt.Printf("%s Properties marked with %s are required\n",
		dim("Note:"), yellow("*"))
}
