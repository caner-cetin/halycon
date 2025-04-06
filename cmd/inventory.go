// query inventory still lacks some flags, configurations, etc. todo.
package cmd

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/fba_inventory"
	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type BuildInventoryConfig struct {
	ForceRebuild bool
}

type QueryInventoryConfig struct {
	Keyword string
	Output  string
}

var (
	queryInventoryCmd = &cobra.Command{
		Use: "count",
		Run: WrapCommandWithResources(queryInventory, ResourceConfig{Resources: []ResourceType{ResourceAmazon, ResourceDB}, Services: []ServiceType{}}),
	}
	queryInventoryCfg QueryInventoryConfig
	buildInventoryCmd = &cobra.Command{
		Use: "build",
		Run: WrapCommandWithResources(buildInventory, ResourceConfig{Resources: []ResourceType{ResourceAmazon, ResourceDB}, Services: []ServiceType{ServiceFBAInventory}}),
	}
	buildInventoryCfg BuildInventoryConfig
	inventoryCmd      = &cobra.Command{
		Use: "inventory",
	}
)

func getInventoryCmd() *cobra.Command {
	queryInventoryCmd.PersistentFlags().StringVarP(&queryInventoryCfg.Keyword, "keyword", "k", "", "keyword to query in product names")
	queryInventoryCmd.MarkFlagRequired("keyword")
	queryInventoryCmd.PersistentFlags().StringVarP(&queryInventoryCfg.Output, "output", "o", "", "output format (csv), optional, results will be pretty printed if not given")
	buildInventoryCmd.PersistentFlags().BoolVarP(&buildInventoryCfg.ForceRebuild, "force-rebuild", "f", false, "forces to rebuild table even if inventory is already built")
	inventoryCmd.AddCommand(queryInventoryCmd)
	inventoryCmd.AddCommand(buildInventoryCmd)
	return inventoryCmd
}
func buildInventory(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	cnt, err := app.Query.FbaInventoryCount(app.Ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			buildInventoryCfg.ForceRebuild = true
		} else {
			log.Error().Err(err).Msg("failed to get inventory count")
			return
		}
	}
	var summaries []fba_inventory.InventorySummary
	if cnt == 0 || buildInventoryCfg.ForceRebuild {
		var nextToken *string
		if _, err := app.DB.ExecContext(app.Ctx, "DELETE FROM fba_inventory;"); err != nil {
			log.Error().Err(err).Msg("failed to clear inventory table")
			return
		}
		stmt, err := app.DB.PrepareContext(app.Ctx,
			`INSERT INTO fba_inventory
			(title, total_quantity, fulfillable_quantity, inbound_receiving_quantity, inbound_shipped_quantity, sku, asin)
			VALUES (?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			log.Error().Err(err).Msg("failed to prepare statement")
			return
		}
		defer internal.CloseStmt(stmt)
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

			status, err := app.Amazon.Client.GetFBAInventorySummaries(context.TODO(), &params)
			if err != nil {
				log.Error().Err(err).Msg("failed to get fba inventory summary")
				return
			}
			result := status.JSON200

			for _, summary := range result.Payload.InventorySummaries {
				_, err := stmt.ExecContext(app.Ctx,
					*summary.ProductName,
					*summary.TotalQuantity,
					*summary.InventoryDetails.FulfillableQuantity,
					*summary.InventoryDetails.InboundReceivingQuantity,
					*summary.InventoryDetails.InboundShippedQuantity,
					*summary.SellerSku,
					*summary.Asin)
				summaries = append(summaries, summary)
				if err != nil {
					log.Error().Err(err).Msg("failed to insert inventory item")
					return
				}
			}
			if result.Payload == nil || result.Pagination == nil {
				break
			}
			nextToken = result.Pagination.NextToken
			if *nextToken == "" {
				break
			}
			i++
		}
		color.Green("fba inventory table built successfully")
	} else {
		color.Magenta("inventory already built, use [--force-rebuild / -f] to force rebuilding process")
	}

	cntRow := app.DB.QueryRow(`SELECT COUNT(title) FROM fts_title_quantity;`)
	var vtCnt int
	if err := cntRow.Scan(&vtCnt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			vtCnt = 0
		} else {
			log.Error().Err(err).Msg("failed to query fts table count")
			return
		}
	}
	if vtCnt == 0 || buildInventoryCfg.ForceRebuild {
		tx, err := app.DB.BeginTx(app.Ctx, nil)
		if err != nil {
			log.Error().Err(err).Msg("failed to begin transaction")
			return
		}
		defer internal.Rollback(tx)

		if _, err := tx.ExecContext(app.Ctx, "DELETE FROM fts_title_quantity;"); err != nil {
			log.Error().Err(err).Msg("failed to clear FTS table within transaction")
			return
		}

		stmt, err := tx.PrepareContext(app.Ctx, `INSERT INTO fts_title_quantity
    (title, total_quantity, fulfillable_quantity, inbound_receiving_quantity, inbound_shipped_quantity)
    VALUES (?, ?, ?, ?, ?);`)
		if err != nil {
			log.Error().Err(err).Msg("failed to prepare FTS insert statement within transaction")
			return
		}
		defer internal.CloseStmt(stmt)

		// if summaries is empty, pull data from the database
		if len(summaries) == 0 {
			rows, err := app.DB.QueryContext(app.Ctx, `SELECT
				title, total_quantity, fulfillable_quantity, inbound_receiving_quantity, inbound_shipped_quantity
				FROM fba_inventory;`)
			if err != nil {
				log.Error().Err(err).Msg("failed to query inventory data")
				return
			}
			defer internal.CloseRows(rows)

			for rows.Next() {
				var title string
				var totalQty, fulfillableQty, inboundReceivingQty, inboundShippedQty int

				if err := rows.Scan(&title, &totalQty, &fulfillableQty, &inboundReceivingQty, &inboundShippedQty); err != nil {
					log.Error().Err(err).Msg("failed to scan inventory row")
					continue
				}

				if _, err := stmt.ExecContext(app.Ctx, title, totalQty, fulfillableQty, inboundReceivingQty, inboundShippedQty); err != nil {
					log.Error().Err(err).Msg("failed to insert into FTS table within transaction")
					return
				}
				if err = rows.Err(); err != nil {
					log.Error().Err(err).Msg("error during rows iteration")
					return
				}
			}
		} else { // or use the data we just fetched
			for _, summary := range summaries {
				if _, err := stmt.ExecContext(app.Ctx,
					*summary.ProductName,
					*summary.TotalQuantity,
					*summary.InventoryDetails.FulfillableQuantity,
					*summary.InventoryDetails.InboundReceivingQuantity,
					*summary.InventoryDetails.InboundShippedQuantity); err != nil {
					log.Error().Err(err).Msg("failed to insert into FTS table")
					return
				}
			}
		}
		if err := tx.Commit(); err != nil {
			log.Error().Err(err).Msg("failed to commit FTS build transaction")
			return
		}
		color.Green("FTS title quantity table built successfully")
	} else {
		color.Magenta("FTS title quantity table already built")
	}
}

func queryInventory(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	rows, err := app.DB.QueryContext(app.Ctx, `SELECT *
FROM fts_title_quantity
WHERE rowid IN (
	SELECT rowid
	FROM fts_title_quantity
	WHERE title MATCH ?
)
AND total_quantity = 0;`, queryInventoryCfg.Keyword) // todo: provide more ways to filter, for now, it just gets products with 0 quantity.
	if err != nil {
		log.Error().Err(err).Msg("failed to query fts table")
		return
	}
	var table []FTSTitleQuantityRow
	for rows.Next() {
		var row FTSTitleQuantityRow
		if err := rows.Scan(&row.Title,
			&row.TotalQuantity,
			&row.FulfillableQuantity,
			&row.InboundReceivingQuantity,
			&row.InboundShippedQuantity); err != nil {
			log.Error().Err(err).Msg("failed to scan row")
			return
		}
		table = append(table, row)
	}

	if len(table) == 0 {
		color.Yellow("No matching inventory items found.")
		return
	}

	if queryInventoryCfg.Output == "csv" {
		outputToCSV(table)
		return
	}

	titleColor := color.New(color.FgGreen, color.Bold)
	labelColor := color.New(color.FgCyan)
	valueColor := color.New(color.FgWhite)
	divider := color.New(color.FgBlue).Sprint("--------------------------------------------------")

	titleColor.Println("Matching inventory items with zero quantity:")
	fmt.Println(divider)
	for _, row := range table {
		titleColor.Printf("%s\n", row.Title)
		labelColor.Print("  Total Quantity: ")
		valueColor.Printf("%d\n", row.TotalQuantity)
		labelColor.Print("  Fulfillable Quantity: ")
		valueColor.Printf("%d\n", row.FulfillableQuantity)
		labelColor.Print("  Inbound Receiving Quantity: ")
		valueColor.Printf("%d\n", row.InboundReceivingQuantity)
		labelColor.Print("  Inbound Shipped Quantity: ")
		valueColor.Printf("%d\n", row.InboundShippedQuantity)
		fmt.Println(divider)
	}
	color.New(color.FgMagenta).Printf("Total items found: %d\n", len(table))
}

func outputToCSV(table []FTSTitleQuantityRow) {
	fileName := fmt.Sprintf("inventory_%s.csv", time.Now().Format("2006-01-02"))
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
		}
		if err := writer.Write(record); err != nil {
			log.Error().Err(err).Msg("failed to write CSV row")
			return
		}
	}

	color.Green("CSV file created: %s with %d records", fileName, len(table))
}

type FTSTitleQuantityRow struct {
	Title                    string
	TotalQuantity            int
	FulfillableQuantity      int
	InboundReceivingQuantity int
	InboundShippedQuantity   int
}
