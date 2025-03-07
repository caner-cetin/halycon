package cmd

import (
	"github.com/spf13/cobra"
)

var (
	createShipmentPlanCmd = &cobra.Command{
		Use: "create",
	}

	shipmentCmd = &cobra.Command{
		Use: "shipment",
	}
)

func getShipmentCmd() *cobra.Command {
	shipmentCmd.AddCommand(createShipmentPlanCmd)
	return shipmentCmd
}

// func createShipmentPlan(cmd *cobra.Command, args []string) {
// 	params := fba_inbound.NewCreateInboundPlanParams()
// 	var request models.CreateInboundPlanRequest
// 	var items []models.ItemInput
// 	items = append(items, models.ItemInput{})
// }
