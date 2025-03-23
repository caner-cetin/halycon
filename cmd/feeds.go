package cmd

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/caner-cetin/halycon/internal"
	"github.com/caner-cetin/halycon/internal/amazon/feeds"
	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type uploadFeedConfig struct {
	FeedType    string
	FeedPath    string
	ContentType string
}

type getFeedConfig struct {
	FeedId string
}

type getFeedReportConfig struct {
	FeedId string
}

var (
	uploadFeedCmd = &cobra.Command{
		Use: "upload",
		Run: WrapCommandWithResources(createFeedDocument, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceFeeds}}),
	}
	uploadFeedCfg uploadFeedConfig
	getFeedCmd    = &cobra.Command{
		Use: "get",
		Run: WrapCommandWithResources(getFeedDocument, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceFeeds}}),
	}
	getFeedCfg       getFeedConfig
	getFeedReportCmd = &cobra.Command{
		Use: "report",
		Run: WrapCommandWithResources(getFeedReport, ResourceConfig{Resources: []ResourceType{ResourceAmazon}, Services: []ServiceType{ServiceFeeds}}),
	}
	getFeedReportCfg getFeedReportConfig
	feedsCmd         = &cobra.Command{
		Use: "feeds",
	}
)

func getFeedsCmd() *cobra.Command {
	uploadFeedCmd.PersistentFlags().StringVarP(&uploadFeedCfg.FeedPath, "input", "i", "", "path for feed")
	uploadFeedCmd.PersistentFlags().StringVar(&uploadFeedCfg.FeedType, "feed-type", "", "feed type")
	uploadFeedCmd.PersistentFlags().StringVar(&uploadFeedCfg.ContentType, "content-type", "text/csv", "feed content type, ('application/json; charset=UTF-8', 'text/csv', etc...)")
	uploadFeedCmd.MarkFlagRequired("input")
	getFeedCmd.PersistentFlags().StringVarP(&getFeedCfg.FeedId, "id", "i", "", "feed id")
	getFeedCmd.MarkFlagRequired("id")
	getFeedReportCmd.PersistentFlags().StringVarP(&getFeedReportCfg.FeedId, "id", "i", "", "feed id")
	getFeedReportCmd.MarkFlagRequired("id")
	feedsCmd.AddCommand(uploadFeedCmd)
	feedsCmd.AddCommand(getFeedCmd)
	feedsCmd.AddCommand(getFeedReportCmd)
	return feedsCmd
}

func createFeedDocument(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	status, err := app.Amazon.Client.CreateFeedDocument(cmd.Context(), feeds.CreateFeedDocumentJSONRequestBody{ContentType: uploadFeedCfg.ContentType})
	if err != nil {
		log.Error().Err(err).Msg("failed to create feed document")
		return
	}
	create_feed_document_response := status.JSON201
	log.Info().
		Str("feed_document_id", create_feed_document_response.FeedDocumentId).
		Str("url", create_feed_document_response.Url).
		Msg("created feed document")

	feed, err := internal.ReadFile(uploadFeedCfg.FeedPath)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	if err = uploadFeedDocument(feed, create_feed_document_response.Url); err != nil {
		log.Error().Str("url", create_feed_document_response.Url).Err(err).Msg("failed to upload feed document")
		return
	}
	var params feeds.CreateFeedJSONRequestBody
	params.MarketplaceIds = cfg.Amazon.Auth.DefaultMerchant.MarketplaceID
	params.FeedType = uploadFeedCfg.FeedType
	params.InputFeedDocumentId = create_feed_document_response.FeedDocumentId
	create_feed_status, err := app.Amazon.Client.CreateFeed(cmd.Context(), params)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	create_feed_response := create_feed_status.JSON202
	log.Info().Str("id", create_feed_response.FeedId).Msg("created feed")
}

func uploadFeedDocument(feed []byte, uri string) error {
	req, err := http.NewRequest(http.MethodPut, uri, bytes.NewBuffer(feed))
	if err != nil {
		return fmt.Errorf("error constructing request: %w", err)
	}

	req.Header.Set("Content-Type", uploadFeedCfg.ContentType)
	req.ContentLength = int64(len(feed))
	// amazon does not support chunked data transfer
	// so we have to let them know content length, and disable chunked encoding, or any encoding at all.
	// set the encoding to identity
	req.TransferEncoding = []string{}

	resp, err := http.DefaultClient.Do(req) //nolint: bodyclose
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer internal.CloseReader(resp.Body)

	if resp.StatusCode >= http.StatusBadRequest {
		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading body: %w", err)
		}
		return fmt.Errorf("unexpected status %d with response %s", resp.StatusCode, string(respBytes))
	}

	return nil
}

func getFeedDocument(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	status, err := app.Amazon.Client.GetFeed(cmd.Context(), getFeedCfg.FeedId)
	if err != nil {
		log.Error().Err(err).Send()
		return
	}
	resp := status.JSON200

	fmt.Printf("%s: %s\n", color.GreenString("Feed ID"), resp.FeedId)
	fmt.Printf("%s: %s\n", color.GreenString("Type"), resp.FeedType)
	fmt.Printf("%s: %s\n", color.GreenString("Created"), resp.CreatedTime.Format(time.RFC3339))
	fmt.Printf("%s: %s\n", color.GreenString("Status"), color.YellowString(string(resp.ProcessingStatus)))
	if resp.ProcessingStartTime != nil {
		fmt.Printf("%s: %s\n", color.GreenString("Started"), resp.ProcessingStartTime.Format(time.RFC3339))
	}
	if resp.ProcessingEndTime != nil {
		fmt.Printf("%s: %s\n", color.GreenString("Completed"), resp.ProcessingEndTime.Format(time.RFC3339))
	}
	if resp.MarketplaceIds != nil {
		fmt.Printf("%s: %s\n", color.GreenString("Marketplaces"), strings.Join(*resp.MarketplaceIds, ", "))
	}
	if resp.ResultFeedDocumentId != nil {
		fmt.Printf("%s: %s\n", color.GreenString("Result Document"), *resp.ResultFeedDocumentId)
	}
}

func getFeedReport(cmd *cobra.Command, args []string) {
	app := GetApp(cmd)
	get_feed_status, err := app.Amazon.Client.GetFeed(cmd.Context(), getFeedReportCfg.FeedId)
	if err != nil {
		log.Error().Err(err).Msg("failed to get feed details from id")
		return
	}
	get_feed_response := get_feed_status.JSON200
	if get_feed_response.ResultFeedDocumentId == nil {
		log.Error().Msg("result feed document does not exist, are you sure that feed processing finished?")
		return
	}
	get_feed_document_status, err := app.Amazon.Client.GetFeedDocument(cmd.Context(), *get_feed_response.ResultFeedDocumentId)
	if err != nil {
		log.Error().Err(err).Msg("failed to get feed document from id")
		return
	}
	get_feed_document_response := get_feed_document_status.JSON200

	req, err := http.NewRequest(http.MethodGet, get_feed_document_response.Url, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to construct request for feed document")
		return
	}
	if get_feed_document_response.CompressionAlgorithm != nil {
		// only possible value is gzip
		// https://developer-docs.amazon.com/sp-api/docs/feeds-api-v2021-06-30-reference#compressionalgorithm
		req.Header.Add("Accept-Encoding", "gzip")
	}
	resp, err := http.DefaultClient.Do(req) //nolint: bodyclose
	if err != nil {
		log.Error().Err(err).Msg("failed to send request for feed document")
		return
	}
	defer internal.CloseReader(resp.Body)
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
	default:
		reader = resp.Body
	}
	defer internal.CloseReader(reader)
	if _, err := io.Copy(os.Stdout, io.LimitReader(reader, int64(5*1024*1024))); err != nil {
		log.Error().Err(err).Send()
		return
	}

}
