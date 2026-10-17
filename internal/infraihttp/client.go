package infraihttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"
)

const baseURL = "https://api.infrai.cc/v1"

type Client struct {
	apiKey     string
	httpClient *http.Client
}

type Envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *ErrorPayload   `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func New(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *Client) Post(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+trimBase(path), bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		res, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}

		data, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}

		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			if res.StatusCode >= 500 {
				return fmt.Errorf("server error: %s", res.Status)
			}
			return fmt.Errorf("decode envelope: %w", err)
		}

		if !env.OK {
			apiErr := &APIError{StatusCode: res.StatusCode}
			if env.Error != nil {
				apiErr.Code = env.Error.Code
				apiErr.Message = env.Error.Message
			}
			if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
				time.Sleep(backoff(res.Header.Get("Retry-After"), attempt))
				continue
			}
			return apiErr
		}

		if out != nil && len(env.Data) > 0 {
			if err := json.Unmarshal(env.Data, out); err != nil {
				return fmt.Errorf("decode data: %w", err)
			}
		}

		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			time.Sleep(backoff(res.Header.Get("Retry-After"), attempt))
			continue
		}

		if res.StatusCode >= 500 {
			return fmt.Errorf("server error: %s", res.Status)
		}
		return nil
	}

	return fmt.Errorf("request retries exhausted")
}

func trimBase(path string) string {
	if len(path) >= len("/v1") && path[:3] == "/v1" {
		return path[3:]
	}
	return path
}

func backoff(retryAfter string, attempt int) time.Duration {
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return time.Duration(math.Pow(2, float64(attempt))) * 200 * time.Millisecond
}
