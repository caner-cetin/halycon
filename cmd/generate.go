package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/caner-cetin/halycon/internal"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type generateCfg struct {
	Input      string
	InputFile  string
	Model      string
	Prompt     string
	PromptFile string
	Timeout    float64
	MaxTokens  int
}

type groqResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Logprobs     any    `json:"logprobs"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		QueueTime        float64 `json:"queue_time"`
		PromptTokens     int     `json:"prompt_tokens"`
		PromptTime       float64 `json:"prompt_time"`
		CompletionTokens int     `json:"completion_tokens"`
		CompletionTime   float64 `json:"completion_time"`
		TotalTokens      int     `json:"total_tokens"`
		TotalTime        float64 `json:"total_time"`
	} `json:"usage"`
	SystemFingerprint string `json:"system_fingerprint"`
	XGroq             struct {
		ID string `json:"id"`
	} `json:"x_groq"`
}

type groqRequest struct {
	Messages            []groqMessages `json:"messages"`
	Model               string         `json:"model"`
	Temperature         int            `json:"temperature"`
	MaxCompletionTokens int            `json:"max_completion_tokens"`
	TopP                int            `json:"top_p"`
	Stream              bool           `json:"stream"`
	Stop                any            `json:"stop"`
}
type groqImageUrl struct {
	URL *string `json:"url"`
}
type groqContent struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *groqImageUrl `json:"image_url,omitempty"`
}
type groqMessages struct {
	Role    string        `json:"role"`
	Content []groqContent `json:"content"`
}

var (
	generateDetailsCfg generateCfg
	generateCmd        = &cobra.Command{
		Use:   "generate",
		Short: "image-text to text AI inference",
		Run:   WrapCommandWithResources(generateDetails, ResourceConfig{}),
	}
)

func getGenerateCmd() *cobra.Command {
	flags := generateCmd.PersistentFlags()
	flags.StringVarP(&generateDetailsCfg.Input, "input", "i", "", "input image link, maximum allowed size for a request containing an image URL as input is 20MB")
	flags.StringVar(&generateDetailsCfg.InputFile, "input-file", "", "input image filepath, maximum allowed size for a request containing a base64 encoded image is 4MB.")
	generateCmd.MarkFlagRequired("input")
	flags.StringVar(&generateDetailsCfg.Model, "model", "llama-3.2-11b-vision-preview", "refer to https://console.groq.com/docs/vision")
	flags.StringVar(&generateDetailsCfg.Prompt, "prompt", "", "prompt for model")
	flags.StringVar(&generateDetailsCfg.PromptFile, "prompt-file", "", "prompt text filepath for model")
	generateCmd.MarkFlagsOneRequired("prompt", "prompt-file")
	generateCmd.MarkFlagsMutuallyExclusive("prompt", "prompt-file")
	flags.Float64Var(&generateDetailsCfg.Timeout, "timeout", 120, "inference timeout (seconds)")
	flags.IntVar(&generateDetailsCfg.MaxTokens, "max-tokens", 500, "The maximum number of tokens that can be generated in the chat completion.")
	return generateCmd
}

func generateDetails(cmd *cobra.Command, args []string) {
	var prompt string
	var err error
	if generateDetailsCfg.PromptFile != "" {
		prompt_bytes, err := internal.ReadFile(generateDetailsCfg.PromptFile)
		if err != nil {
			log.Error().Err(err).Str("path", generateDetailsCfg.PromptFile).Msg("failed to read prompt text file")
			return
		}
		prompt = string(prompt_bytes)
	} else {
		prompt = generateDetailsCfg.Prompt
	}
	var image []byte
	var content_type string
	if generateDetailsCfg.InputFile != "" {
		image, err = internal.ReadFile(generateDetailsCfg.InputFile)
		if err != nil {
			log.Error().Err(err).Msg("error reading image input")
			return
		}
		ifn_split := strings.Split(generateDetailsCfg.InputFile, ",")
		image_ext := ifn_split[len(ifn_split)-1]
		if image_ext == "jpg" {
			image_ext = "jpeg"
		}
		content_type = fmt.Sprintf("image/%s", image_ext)
	} else {
		ev := log.With().Str("link", generateDetailsCfg.Input).Logger()
		req, err := http.NewRequest(http.MethodGet, generateDetailsCfg.Input, nil)
		if err != nil {
			ev.Error().Err(err).Msg("failed to construct request for image link")
			return
		}
		req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			ev.Error().Err(err).Msg("failed to send request to image link")
			return
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			ev.Error().Err(err).Msg("failed to read image bytes")
			return
		}
		if resp.StatusCode > 299 {
			ev.Error().Int("status", resp.StatusCode).Str("body", string(body)).Msg("unexpected response from remote")
			return
		}
		content_type = resp.Header.Get("content-type")
		if strings.Contains(content_type, "text/html") {
			ev.Error().Int("status", resp.StatusCode).Msg("invalid text/html response from remote")
			return
		}
		image = body
	}
	content_type = strings.TrimSpace(content_type)
	var payload groqRequest
	var message groqMessages
	message.Role = "user"
	message.Content = append(message.Content, groqContent{Type: "text", Text: prompt})
	message.Content = append(message.Content, groqContent{Type: "image_url", ImageURL: &groqImageUrl{URL: internal.Ptr(fmt.Sprintf("data:%s;base64,%s", content_type, base64.StdEncoding.EncodeToString(image)))}})
	payload.Messages = append(payload.Messages, message)
	payload.Model = generateDetailsCfg.Model
	payload.Stream = false
	payload.MaxCompletionTokens = 500
	payload.TopP = 1
	payload.Temperature = 1
	payload.Stream = false
	payload.Stop = nil
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal inference request payload")
		return
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Error().Err(err).Msg("failed to create request for inference")
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", strings.TrimSpace(cfg.Groq.Token)))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: time.Minute * 2,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("failed to send inference request")
		return
	}
	defer internal.CloseResponseBody(resp)

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Error().Int("status", resp.StatusCode).Str("url", resp.Request.URL.String()).Str("body", string(bodyBytes)).Msg("unexpected response from remote")
		return
	}
	var completion groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		log.Error().Err(err).Send()
		return
	}
	if len(completion.Choices) == 0 {
		log.Error().Msg("empty response from inference api")
		return
	}
	fmt.Println(completion.Choices[0].Message.Content)
}
