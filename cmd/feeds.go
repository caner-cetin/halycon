package cmd

// import (
// 	"github.com/caner-cetin/halycon/internal/amazon/feeds"
// 	"github.com/rs/zerolog/log"
// 	"github.com/spf13/cobra"
// )

// type uploadFeedConfig struct {
// 	Type string
// }

// var (
// 	uploadFeedCmd = &cobra.Command{
// 		Use: "upload",
// 		Run: WrapCommandWithResources(createFeedDocument, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceFeeds}}),
// 	}
// 	uploadFeedCfg uploadFeedConfig
// 	feedsCmd = &cobra.Command{
// 		Use: "feeds",
// 	}
// )

// func getFeedsCmd() *cobra.Command {
// 	flags := uploadFeedCmd.Flags()
// 	flags.StringVar(&uploadFeedCfg.Type, "type", "", "type of the feed")
// 	uploadFeedCmd.MarkFlagRequired("type")
// 	feedsCmd.AddCommand(uploadFeedCmd)
// 	return feedsCmd
// }

// func createFeedDocument(cmd *cobra.Command, args []string) {
// 	app := GetApp(cmd)
// 	var params *feeds.CreateFeedDocumentJSONRequestBody
// 	params.ContentType = "application/json; charset=UTF-8"
// 	status, err := app.Amazon.Client.CreateFeedDocument(cmd.Context(), *params)
// 	if err != nil {
// 		log.Error().Err(err).Msg("failed to create feed document")
// 		return
// 	}
// 	resp := status.JSON201
// 	log.Info().
// 		Str("feed_document_id", resp.FeedDocumentId).
// 		Str("url", resp.Url).
// 		Msg("created feed document")
// }
