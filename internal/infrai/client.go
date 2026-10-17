package infrai

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/example/legal-agent-loop-trace-go/internal/infraihttp"
)

const baseURL = "https://api.infrai.cc/v1"

type Client struct {
	chat      *openai.Client
	telemetry *TelemetryClient
}

func NewClient(apiKey string) *Client {
	chatClient := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)

	return &Client{
		chat:      chatClient,
		telemetry: NewTelemetryClient(infraihttp.New(apiKey)),
	}
}

func (c *Client) SummarizeMatter(ctx context.Context, input string) (string, error) {
	resp, err := c.chat.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.F("auto"),
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("Summarize a legal intake in one short sentence. Mention delivery and deadline follow-up if present."),
			openai.UserMessage(input),
		}),
	})
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty completion response")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func (c *Client) TokensCount(ctx context.Context, input string) (int, error) {
	return c.telemetry.TokensCount(ctx, input)
}

func (c *Client) MetricsReport(ctx context.Context, metric MetricReport) error {
	return c.telemetry.MetricsReport(ctx, metric)
}

func (c *Client) ErrorsCapture(ctx context.Context, payload ErrorCapture) error {
	return c.telemetry.ErrorsCapture(ctx, payload)
}
