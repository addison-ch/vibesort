package vibesort

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultModel   = "gpt-4.1-mini"
	defaultBaseURL = "https://api.openai.com/v1/chat/completions"
)

// Client calls OpenAI to determine a "vibe-based" ordering.
type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// Option customizes the vibesort Client.
type Option func(*Client)

// WithModel sets the LLM model used for sorting.
func WithModel(model string) Option {
	return func(c *Client) {
		if strings.TrimSpace(model) != "" {
			c.model = model
		}
	}
}

// WithBaseURL overrides the OpenAI-compatible endpoint.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		if strings.TrimSpace(url) != "" {
			c.baseURL = url
		}
	}
}

// WithHTTPClient supplies a custom HTTP client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.httpClient = h
		}
	}
}

// NewClient creates a vibesort client using the given OpenAI API key.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("api key is required")
	}

	c := &Client{
		apiKey:  apiKey,
		model:   defaultModel,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// SortStrings returns a reordered copy of items according to the provided vibe.
//
// Example vibes:
//   - "most likely to survive a zombie apocalypse"
//   - "rank by chaotic neutral energy"
func (c *Client) SortStrings(ctx context.Context, items []string, vibe string) ([]string, error) {
	if len(items) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(vibe) == "" {
		return nil, errors.New("vibe prompt is required")
	}

	userPrompt := buildPrompt(items, vibe)
	reqBody := chatCompletionRequest{
		Model: c.model,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: "You are a vibe ranking engine. Return ONLY JSON with this shape: " +
					`{"order":[integer,...]}` +
					". The array must include every input index exactly once.",
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		Temperature: 0.9,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai error (%d): %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(respBytes, &completion); err != nil {
		return nil, fmt.Errorf("decode openai response: %w", err)
	}
	if len(completion.Choices) == 0 {
		return nil, errors.New("openai response had no choices")
	}

	raw := completion.Choices[0].Message.Content
	order, err := parseOrder(raw)
	if err != nil {
		return nil, fmt.Errorf("parse model order: %w", err)
	}
	if err := validateOrder(order, len(items)); err != nil {
		return nil, fmt.Errorf("invalid model order: %w", err)
	}

	out := make([]string, 0, len(items))
	for _, idx := range order {
		out = append(out, items[idx])
	}
	return out, nil
}

func buildPrompt(items []string, vibe string) string {
	var b strings.Builder
	b.WriteString("Sort these items by vibe.\n")
	b.WriteString("Vibe: ")
	b.WriteString(vibe)
	b.WriteString("\nItems (index: value):\n")
	for i, item := range items {
		b.WriteString(fmt.Sprintf("%d: %q\n", i, item))
	}
	b.WriteString("Respond with JSON only.")
	return b.String()
}

func parseOrder(raw string) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("empty content")
	}

	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end < start {
		return nil, errors.New("no JSON object found")
	}

	var data struct {
		Order []int `json:"order"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &data); err != nil {
		return nil, err
	}
	return data.Order, nil
}

func validateOrder(order []int, expectedLen int) error {
	if len(order) != expectedLen {
		return fmt.Errorf("expected %d indexes, got %d", expectedLen, len(order))
	}
	seen := make(map[int]bool, expectedLen)
	for _, v := range order {
		if v < 0 || v >= expectedLen {
			return fmt.Errorf("index out of range: %d", v)
		}
		if seen[v] {
			return fmt.Errorf("duplicate index: %d", v)
		}
		seen[v] = true
	}
	return nil
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
