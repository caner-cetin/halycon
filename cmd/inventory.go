package cmd

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/catalog"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inventory"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const customFilter = "custom"

type BuildInventoryConfig struct {
	ForceRebuild bool
}

type QueryInventoryConfig struct {
	Keyword string
	Output  string
}

type InventoryFilter struct {
	Keyword        string
	QuantityFilter string
	MinQuantityStr string
	MaxQuantityStr string
	MinQuantity    int
	MaxQuantity    int
	OutputFormat   string
	ShowPreview    bool
	SortBy         string
	SortOrder      string
}

var (
	queryInventoryCmd = &cobra.Command{
		Use: "count",
		Run: WrapCommandWithResources(queryInventory, ResourceConfig{Resources: []ResourceType{ResourceAmazon, ResourceDB}, Services: []ServiceType{}}),
	}
	queryInventoryCfg QueryInventoryConfig
	buildInventoryCmd = &cobra.Command{
		Use: "build",
		Run: WrapCommandWithResources(buildInventory, ResourceConfig{Resources: []ResourceType{ResourceAmazon, ResourceDB}, Services: []ServiceType{ServiceFBAInventory, ServiceCatalog}}),
	}
	buildInventoryCfg BuildInventoryConfig
	inventoryCmd      = &cobra.Command{
		Use: "inventory",
	}
)

func getInventoryCmd() *cobra.Command {
	queryInventoryCmd.PersistentFlags().StringVarP(&queryInventoryCfg.Keyword, "keyword", "k", "", "keyword to query in product names (optional, will prompt if not provided)")
	queryInventoryCmd.PersistentFlags().StringVarP(&queryInventoryCfg.Output, "output", "o", "", "output format (csv, json, table), optional, will prompt if not provided")
	buildInventoryCmd.PersistentFlags().BoolVarP(&buildInventoryCfg.ForceRebuild, "force-rebuild", "f", false, "forces to rebuild table even if inventory is already built")
	inventoryCmd.AddCommand(queryInventoryCmd)
	inventoryCmd.AddCommand(buildInventoryCmd)
	return inventoryCmd
}

func getUPCsBatch(app AppCtx, asins []string) (map[string]string, error) {
	upcMap := make(map[string]string)

	batchSize := 10
	for i := 0; i < len(asins); i += batchSize {
		end := i + batchSize
		if end > len(asins) {
			end = len(asins)
		}

		batch := asins[i:end]
		batchUPCs, err := searchCatalogItemsForUPCs(app, batch)
		if err != nil {
			log.Warn().Err(err).Interface("batch", batch).Msg("failed to get UPC data for batch")
			continue
		}

		for asin, upc := range batchUPCs {
			if upc != "" {
				upcMap[asin] = upc
			}
		}

		time.Sleep(600 * time.Millisecond)
	}

	return upcMap, nil
}

func searchCatalogItemsForUPCs(app AppCtx, asins []string) (map[string]string, error) {
	upcMap := make(map[string]string)

	params := catalog.SearchCatalogItemsParams{
		MarketplaceIds:  cfg.Amazon.Auth.DefaultMerchant.MarketplaceID,
		IdentifiersType: internal.Ptr(catalog.ASIN),
		Identifiers:     &asins,
		IncludedData:    &[]catalog.SearchCatalogItemsParamsIncludedData{"identifiers"},
	}

	response, err := app.Amazon.Client.SearchCatalogItems(app.Ctx, &params)
	if err != nil {
		return nil, err
	}

	if response.JSON200 == nil || response.JSON200.Items == nil {
		return upcMap, nil
	}

	for _, item := range response.JSON200.Items {
		if item.Asin == "" || item.Identifiers == nil {
			continue
		}

		asin := item.Asin

		for _, marketplace := range *item.Identifiers {
			for _, identifier := range marketplace.Identifiers {
				if identifier.IdentifierType == "UPC" {
					upcMap[asin] = identifier.Identifier
					break
				}
			}
		}
	}

	return upcMap, nil
}

func checkAndSetForceRebuild(app AppCtx) error {
	_, err := app.Query.FbaInventoryCount(app.Ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			buildInventoryCfg.ForceRebuild = true
		} else {
			return fmt.Errorf("failed to get inventory count: %w", err)
		}
	}
	return nil
}

func fetchInventorySummaries(app AppCtx) ([]fba_inventory.InventorySummary, []string, error) {
	var summaries []fba_inventory.InventorySummary
	var collectedASINs []string
	var nextToken *string

	for {
		params := fba_inventory.GetInventorySummariesParams{}
		params.MarketplaceIds = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID
		params.GranularityType = "Marketplace"
		params.GranularityId = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID[0]
		params.Details = internal.Ptr(true)

		startDate := time.Now().AddDate(0, 0, -365)
		params.StartDateTime = &startDate

		if nextToken != nil {
			params.NextToken = nextToken
		}

		status, err := app.Amazon.Client.GetFBAInventorySummaries(context.TODO(), &params)
		if err != nil {
			if strings.Contains(err.Error(), "parsing time") && strings.Contains(err.Error(), "cannot parse") {
				log.Warn().Err(err).Msg("timestamp parsing error in Amazon API response - this is likely due to empty timestamp fields in the response")
				log.Info().Msg("trying to continue with partial data...")
				return summaries, collectedASINs, nil
			}
			return nil, nil, fmt.Errorf("failed to get fba inventory summary: %w", err)
		}
		result := status.JSON200

		for _, summary := range result.Payload.InventorySummaries {
			if summary.Asin != nil {
				collectedASINs = append(collectedASINs, *summary.Asin)
			}
			summaries = append(summaries, summary)
		}
		if result.Payload == nil || result.Pagination == nil {
			break
		}
		nextToken = result.Pagination.NextToken
		if *nextToken == "" {
			break
		}
	}

	return summaries, collectedASINs, nil
}

func insertInventorySummaries(app AppCtx, summaries []fba_inventory.InventorySummary, upcMap map[string]string) error {
	stmt, err := app.DB.PrepareContext(app.Ctx,
		`INSERT INTO fba_inventory
		(title, total_quantity, fulfillable_quantity, inbound_receiving_quantity, inbound_shipped_quantity, sku, asin, upc)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer internal.CloseStmt(stmt)

	for _, summary := range summaries {
		var upc string
		if summary.Asin != nil {
			upc = upcMap[*summary.Asin]
		}

		_, err := stmt.ExecContext(app.Ctx,
			*summary.ProductName,
			*summary.TotalQuantity,
			*summary.InventoryDetails.FulfillableQuantity,
			*summary.InventoryDetails.InboundReceivingQuantity,
			*summary.InventoryDetails.InboundShippedQuantity,
			*summary.SellerSku,
			*summary.Asin,
			upc)
		if err != nil {
			return fmt.Errorf("failed to insert inventory item: %w", err)
		}
	}

	return nil
}

func buildFBAInventoryTable(app AppCtx) ([]fba_inventory.InventorySummary, map[string]string, error) {
	cnt, err := app.Query.FbaInventoryCount(app.Ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, fmt.Errorf("failed to get inventory count: %w", err)
	}

	if cnt == 0 || buildInventoryCfg.ForceRebuild {
		if _, err := app.DB.ExecContext(app.Ctx, "DELETE FROM fba_inventory;"); err != nil {
			return nil, nil, fmt.Errorf("failed to clear inventory table: %w", err)
		}

		summaries, collectedASINs, err := fetchInventorySummaries(app)
		if err != nil {
			return nil, nil, err
		}

		log.Info().Int("asin_count", len(collectedASINs)).Msg("fetching UPC data for ASINs")
		upcMap, err := getUPCsBatch(app, collectedASINs)
		if err != nil {
			log.Error().Err(err).Msg("failed to fetch UPC data, continuing without UPC information")
		}

		if err := insertInventorySummaries(app, summaries, upcMap); err != nil {
			return nil, nil, err
		}

		color.Green("fba inventory table built successfully")
		return summaries, upcMap, nil
	} else {
		color.Magenta("inventory already built, use [--force-rebuild / -f] to force rebuilding process")
		return nil, make(map[string]string), nil
	}
}

func populateFTSFromDatabase(app AppCtx, stmt *sql.Stmt) error {
	rows, err := app.DB.QueryContext(app.Ctx, `SELECT
		title, total_quantity, fulfillable_quantity, inbound_receiving_quantity, inbound_shipped_quantity, upc
		FROM fba_inventory;`)
	if err != nil {
		return fmt.Errorf("failed to query inventory data: %w", err)
	}
	defer internal.CloseRows(rows)

	for rows.Next() {
		var title string
		var totalQty, fulfillableQty, inboundReceivingQty, inboundShippedQty int
		var upcNull sql.NullString

		if err := rows.Scan(&title, &totalQty, &fulfillableQty, &inboundReceivingQty, &inboundShippedQty, &upcNull); err != nil {
			log.Error().Err(err).Msg("failed to scan inventory row")
			continue
		}

		upc := ""
		if upcNull.Valid {
			upc = upcNull.String
		}

		if _, err := stmt.ExecContext(app.Ctx, title, totalQty, fulfillableQty, inboundReceivingQty, inboundShippedQty, upc); err != nil {
			return fmt.Errorf("failed to insert into FTS table: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error during rows iteration: %w", err)
	}

	return nil
}

func populateFTSFromSummaries(app AppCtx, stmt *sql.Stmt, summaries []fba_inventory.InventorySummary, upcMap map[string]string) error {
	for _, summary := range summaries {
		var upc string
		if summary.Asin != nil {
			upc = upcMap[*summary.Asin]
		}

		if _, err := stmt.ExecContext(app.Ctx,
			*summary.ProductName,
			*summary.TotalQuantity,
			*summary.InventoryDetails.FulfillableQuantity,
			*summary.InventoryDetails.InboundReceivingQuantity,
			*summary.InventoryDetails.InboundShippedQuantity,
			upc); err != nil {
			return fmt.Errorf("failed to insert into FTS table: %w", err)
		}
	}
	return nil
}

func buildFTSTable(app AppCtx, summaries []fba_inventory.InventorySummary, upcMap map[string]string) error {
	cntRow := app.DB.QueryRow(`SELECT COUNT(title) FROM fts_title_quantity;`)
	var vtCnt int
	if err := cntRow.Scan(&vtCnt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			vtCnt = 0
		} else {
			return fmt.Errorf("failed to query fts table count: %w", err)
		}
	}

	if vtCnt == 0 || buildInventoryCfg.ForceRebuild {
		tx, err := app.DB.BeginTx(app.Ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer internal.Rollback(tx)

		if _, err := tx.ExecContext(app.Ctx, "DELETE FROM fts_title_quantity;"); err != nil {
			return fmt.Errorf("failed to clear FTS table: %w", err)
		}

		stmt, err := tx.PrepareContext(app.Ctx, `INSERT INTO fts_title_quantity
    (title, total_quantity, fulfillable_quantity, inbound_receiving_quantity, inbound_shipped_quantity, upc)
    VALUES (?, ?, ?, ?, ?, ?);`)
		if err != nil {
			return fmt.Errorf("failed to prepare FTS insert statement: %w", err)
		}
		defer internal.CloseStmt(stmt)

		if len(summaries) == 0 {
			if err := populateFTSFromDatabase(app, stmt); err != nil {
				return err
			}
		} else {
			if err := populateFTSFromSummaries(app, stmt, summaries, upcMap); err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit FTS build transaction: %w", err)
		}
		color.Green("FTS title quantity table built successfully")
	} else {
		color.Magenta("FTS title quantity table already built")
	}

	return nil
}

func buildInventory(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)

	if err := checkAndSetForceRebuild(app); err != nil {
		log.Error().Err(err).Msg("failed to check inventory count")
		return
	}

	summaries, upcMap, err := buildFBAInventoryTable(app)
	if err != nil {
		log.Error().Err(err).Msg("failed to build FBA inventory table")
		return
	}

	if err := buildFTSTable(app, summaries, upcMap); err != nil {
		log.Error().Err(err).Msg("failed to build FTS table")
		return
	}
}

func queryInventory(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		MarginBottom(1)

	fmt.Printf("%s\n", headerStyle.Render("📦 Inventory Query Tool"))

	filter, err := configureInventoryFilter()
	if err != nil {
		log.Error().Err(err).Msg("failed to configure inventory filter")
		return
	}

	if queryInventoryCfg.Keyword != "" {
		filter.Keyword = queryInventoryCfg.Keyword
	}
	if queryInventoryCfg.Output != "" {
		filter.OutputFormat = queryInventoryCfg.Output
	}

	rows, err := queryInventoryWithFilter(app, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to query inventory")
		return
	}

	if len(rows) == 0 {
		noResultsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true)

		fmt.Printf("\n%s\n", noResultsStyle.Render("🔍 No matching inventory items found."))
		fmt.Println("💡 Try adjusting your search criteria or check if inventory data is built.")
		return
	}

	if filter.ShowPreview {
		showInventoryPreview(rows, filter)

		var proceed bool
		err = huh.NewConfirm().
			Title("Proceed with this query?").
			Description(fmt.Sprintf("Found %d items matching your criteria", len(rows))).
			Value(&proceed).
			Run()

		if err != nil || !proceed {
			return
		}
	}

	switch filter.OutputFormat {
	case "csv":
		outputToCSV(rows)
	case "json":
		outputToJSON(rows)
	default:
		outputToTable(rows, filter)
	}
}

func outputToCSV(table []FTSTitleQuantityRow) {
	fileName := fmt.Sprintf("inventory_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
	file, err := os.Create(fileName)
	if err != nil {
		log.Error().Err(err).Msg("failed to create CSV file")
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"Title",
		"Total Quantity",
		"Fulfillable Quantity",
		"Inbound Receiving Quantity",
		"Inbound Shipped Quantity",
		"UPC",
	}
	if err := writer.Write(header); err != nil {
		log.Error().Err(err).Msg("failed to write CSV header")
		return
	}

	for _, row := range table {
		record := []string{
			row.Title,
			strconv.Itoa(row.TotalQuantity),
			strconv.Itoa(row.FulfillableQuantity),
			strconv.Itoa(row.InboundReceivingQuantity),
			strconv.Itoa(row.InboundShippedQuantity),
			row.UPC,
		}
		if err := writer.Write(record); err != nil {
			log.Error().Err(err).Msg("failed to write CSV row")
			return
		}
	}

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#22C55E")).
		Bold(true)

	fmt.Printf("\n%s\n", successStyle.Render("✅ CSV Export Complete"))
	fmt.Printf("📄 File: %s\n", fileName)
	fmt.Printf("📊 Records: %d\n", len(table))
}

type FTSTitleQuantityRow struct {
	Title                    string
	TotalQuantity            int
	FulfillableQuantity      int
	InboundReceivingQuantity int
	InboundShippedQuantity   int
	UPC                      string
}

func configureInventoryFilter() (*InventoryFilter, error) {
	filter := &InventoryFilter{}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Search Keywords").
				Description("Enter keywords to search in product titles (use * for wildcard)").
				Value(&filter.Keyword).
				Placeholder("e.g., iPhone, Samsung Galaxy, *phone*"),

			huh.NewSelect[string]().
				Title("Quantity Filter").
				Description("How do you want to filter by quantity?").
				Options(
					huh.NewOption("Show items with zero quantity", "zero"),
					huh.NewOption("Show items with low stock (≤5)", "low"),
					huh.NewOption("Show items with normal stock (>5)", "normal"),
					huh.NewOption("Show items with high stock (>50)", "high"),
					huh.NewOption("Custom quantity range", "custom"),
					huh.NewOption("Show all items", "all"),
				).
				Value(&filter.QuantityFilter),
		),

		huh.NewGroup(
			huh.NewInput().
				Title("Minimum Quantity").
				Description("Minimum total quantity (leave empty for no minimum)").
				Value(&filter.MinQuantityStr).
				Placeholder("0").
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
					if i, err := strconv.Atoi(s); err != nil {
						return fmt.Errorf("must be a number")
					} else if i < 0 {
						return fmt.Errorf("minimum quantity cannot be negative")
					}
					return nil
				}),

			huh.NewInput().
				Title("Maximum Quantity").
				Description("Maximum total quantity (leave empty for no maximum)").
				Value(&filter.MaxQuantityStr).
				Placeholder("1000").
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
					if i, err := strconv.Atoi(s); err != nil {
						return fmt.Errorf("must be a number")
					} else if i < 0 {
						return fmt.Errorf("maximum quantity cannot be negative")
					}
					return nil
				}),
		).WithHideFunc(func() bool {
			return filter.QuantityFilter != customFilter
		}),

		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Sort By").
				Description("How do you want to sort the results?").
				Options(
					huh.NewOption("Product Title (A-Z)", "title_asc"),
					huh.NewOption("Product Title (Z-A)", "title_desc"),
					huh.NewOption("Total Quantity (Low to High)", "quantity_asc"),
					huh.NewOption("Total Quantity (High to Low)", "quantity_desc"),
					huh.NewOption("Fulfillable Quantity (Low to High)", "fulfillable_asc"),
					huh.NewOption("Fulfillable Quantity (High to Low)", "fulfillable_desc"),
				).
				Value(&filter.SortBy),

			huh.NewSelect[string]().
				Title("Output Format").
				Description("How do you want the results displayed?").
				Options(
					huh.NewOption("Interactive Table", "table"),
					huh.NewOption("CSV File", "csv"),
					huh.NewOption("JSON File", "json"),
				).
				Value(&filter.OutputFormat),

			huh.NewConfirm().
				Title("Preview Results").
				Description("Show a preview before generating final output?").
				Value(&filter.ShowPreview),
		),
	)

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("failed to run inventory filter form: %w", err)
	}

	if filter.MinQuantityStr != "" {
		if val, err := strconv.Atoi(filter.MinQuantityStr); err == nil {
			filter.MinQuantity = val
		}
	}
	if filter.MaxQuantityStr != "" {
		if val, err := strconv.Atoi(filter.MaxQuantityStr); err == nil {
			filter.MaxQuantity = val
		}
	}

	switch filter.QuantityFilter {
	case "zero":
		filter.MinQuantity = 0
		filter.MaxQuantity = 0
	case "low":
		filter.MinQuantity = 0
		filter.MaxQuantity = 5
	case "normal":
		filter.MinQuantity = 6
		filter.MaxQuantity = 50
	case "high":
		filter.MinQuantity = 51
		filter.MaxQuantity = 999999
	case "all":
		filter.MinQuantity = 0
		filter.MaxQuantity = 999999
	case "custom":
		if filter.MinQuantityStr == "" {
			filter.MinQuantity = 0
		}
		if filter.MaxQuantityStr == "" {
			filter.MaxQuantity = 999999
		}
	}

	return filter, nil
}

func queryInventoryWithFilter(app AppCtx, filter *InventoryFilter) ([]FTSTitleQuantityRow, error) {
	var query strings.Builder
	var args []interface{}

	query.WriteString(`SELECT title, total_quantity, fulfillable_quantity, inbound_receiving_quantity, inbound_shipped_quantity, upc
FROM fts_title_quantity`)

	var conditions []string

	if filter.Keyword != "" {
		conditions = append(conditions, `rowid IN (
			SELECT rowid FROM fts_title_quantity WHERE title MATCH ?
		)`)
		args = append(args, filter.Keyword)
	}

	if filter.MinQuantity > 0 || filter.MaxQuantity < 999999 {
		if filter.MinQuantity == filter.MaxQuantity {
			conditions = append(conditions, "total_quantity = ?")
			args = append(args, filter.MinQuantity)
		} else {
			if filter.MinQuantity > 0 {
				conditions = append(conditions, "total_quantity >= ?")
				args = append(args, filter.MinQuantity)
			}
			if filter.MaxQuantity < 999999 {
				conditions = append(conditions, "total_quantity <= ?")
				args = append(args, filter.MaxQuantity)
			}
		}
	}

	if len(conditions) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(strings.Join(conditions, " AND "))
	}

	switch filter.SortBy {
	case "title_asc":
		query.WriteString(" ORDER BY title ASC")
	case "title_desc":
		query.WriteString(" ORDER BY title DESC")
	case "quantity_asc":
		query.WriteString(" ORDER BY total_quantity ASC")
	case "quantity_desc":
		query.WriteString(" ORDER BY total_quantity DESC")
	case "fulfillable_asc":
		query.WriteString(" ORDER BY fulfillable_quantity ASC")
	case "fulfillable_desc":
		query.WriteString(" ORDER BY fulfillable_quantity DESC")
	default:
		query.WriteString(" ORDER BY title ASC")
	}

	rows, err := app.DB.QueryContext(app.Ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close rows")
		}
	}()

	var table []FTSTitleQuantityRow
	for rows.Next() {
		var row FTSTitleQuantityRow
		var upcNull sql.NullString
		if err := rows.Scan(&row.Title,
			&row.TotalQuantity,
			&row.FulfillableQuantity,
			&row.InboundReceivingQuantity,
			&row.InboundShippedQuantity,
			&upcNull); err != nil {
			return nil, fmt.Errorf("failed to scan inventory row: %w", err)
		}

		if upcNull.Valid {
			row.UPC = upcNull.String
		} else {
			row.UPC = ""
		}

		table = append(table, row)
	}

	return table, nil
}

func showInventoryPreview(rows []FTSTitleQuantityRow, filter *InventoryFilter) {
	previewStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B82F6")).
		Bold(true).
		MarginTop(1).
		MarginBottom(1)

	fmt.Printf("%s\n", previewStyle.Render("📋 Query Preview"))

	fmt.Printf("🔍 Search: %s\n", getSearchSummary(filter))
	fmt.Printf("📊 Results: %d items found\n", len(rows))

	maxPreview := 5
	if len(rows) > maxPreview {
		fmt.Printf("📝 Showing first %d results:\n\n", maxPreview)
		rows = rows[:maxPreview]
	} else {
		fmt.Printf("📝 All results:\n\n")
	}

	for i, row := range rows {
		fmt.Printf("  %d. %s\n", i+1, truncateString(row.Title, 60))
		upcDisplay := row.UPC
		if upcDisplay == "" {
			upcDisplay = "N/A"
		}
		fmt.Printf("     Total: %d | Fulfillable: %d | UPC: %s\n", row.TotalQuantity, row.FulfillableQuantity, upcDisplay)
		if i < len(rows)-1 {
			fmt.Println()
		}
	}

	fmt.Println()
}

func getSearchSummary(filter *InventoryFilter) string {
	var parts []string

	if filter.Keyword != "" {
		parts = append(parts, fmt.Sprintf("Keywords: \"%s\"", filter.Keyword))
	}

	switch filter.QuantityFilter {
	case "zero":
		parts = append(parts, "Zero quantity items")
	case "low":
		parts = append(parts, "Low stock items (≤5)")
	case "normal":
		parts = append(parts, "Normal stock items (6-50)")
	case "high":
		parts = append(parts, "High stock items (>50)")
	case "custom":
		parts = append(parts, fmt.Sprintf("Quantity range: %d-%d", filter.MinQuantity, filter.MaxQuantity))
	case "all":
		parts = append(parts, "All items")
	}

	if len(parts) == 0 {
		return "All items"
	}

	return strings.Join(parts, ", ")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func outputToTable(table []FTSTitleQuantityRow, filter *InventoryFilter) {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#22C55E")).
		Bold(true).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#06B6D4")).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F3F4F6"))

	dividerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	fmt.Printf("%s\n", titleStyle.Render("📦 Inventory Results"))
	fmt.Printf("🔍 Search: %s\n", getSearchSummary(filter))

	divider := dividerStyle.Render("────────────────────────────────────────────────────────────────────────────────")
	fmt.Println(divider)

	for i, row := range table {
		fmt.Printf("%s\n", titleStyle.Render(fmt.Sprintf("%d. %s", i+1, row.Title)))

		fmt.Printf("   %s %s\n", labelStyle.Render("Total Quantity:"), valueStyle.Render(fmt.Sprintf("%d", row.TotalQuantity)))
		fmt.Printf("   %s %s\n", labelStyle.Render("Fulfillable:"), valueStyle.Render(fmt.Sprintf("%d", row.FulfillableQuantity)))
		fmt.Printf("   %s %s\n", labelStyle.Render("Inbound Receiving:"), valueStyle.Render(fmt.Sprintf("%d", row.InboundReceivingQuantity)))
		fmt.Printf("   %s %s\n", labelStyle.Render("Inbound Shipped:"), valueStyle.Render(fmt.Sprintf("%d", row.InboundShippedQuantity)))
		upcDisplay := row.UPC
		if upcDisplay == "" {
			upcDisplay = "N/A"
		}
		fmt.Printf("   %s %s\n", labelStyle.Render("UPC:"), valueStyle.Render(upcDisplay))

		if i < len(table)-1 {
			fmt.Println(divider)
		}
	}

	fmt.Println(divider)

	summaryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A855F7")).
		Bold(true)

	fmt.Printf("%s\n", summaryStyle.Render(fmt.Sprintf("📋 Total items found: %d", len(table))))
}

func outputToJSON(table []FTSTitleQuantityRow) {
	fileName := fmt.Sprintf("inventory_%s.json", time.Now().Format("2006-01-02_15-04-05"))

	jsonData, err := json.MarshalIndent(table, "", "  ")
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal JSON")
		return
	}

	if err := os.WriteFile(fileName, jsonData, 0644); err != nil {
		log.Error().Err(err).Msg("failed to write JSON file")
		return
	}

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#22C55E")).
		Bold(true)

	fmt.Printf("\n%s\n", successStyle.Render("✅ JSON Export Complete"))
	fmt.Printf("📄 File: %s\n", fileName)
	fmt.Printf("📊 Records: %d\n", len(table))
}
