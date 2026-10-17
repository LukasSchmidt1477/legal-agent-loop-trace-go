package infrai

import (
	"context"
	"fmt"

	"github.com/example/legal-agent-loop-trace-go/internal/infraihttp"
)

type TelemetryClient struct {
	http *infraihttp.Client
}

type MetricReport map[string]any

type ErrorCapture map[string]any

func NewTelemetryClient(httpClient *infraihttp.Client) *TelemetryClient {
	return &TelemetryClient{http: httpClient}
}

func (c *TelemetryClient) TokensCount(ctx context.Context, input string) (int, error) {
	var out struct {
		Count int `json:"count"`
	}
	if err := c.http.Post(ctx, "/v1/ai/tokens/count", map[string]any{
		"model": "auto",
		"messages": []map[string]string{{"role": "user", "content": input}},
	}, &out); err != nil {
		return 0, err
	}
	return out.Count, nil
}

func (c *TelemetryClient) MetricsReport(ctx context.Context, metric MetricReport) error {
	return c.http.Post(ctx, "/v1/metrics/report", metric, nil)
}

func (c *TelemetryClient) ErrorsCapture(ctx context.Context, payload ErrorCapture) error {
	if err := c.http.Post(ctx, "/v1/errors/capture", payload, nil); err != nil {
		return fmt.Errorf("capture error: %w", err)
	}
	return nil
}
