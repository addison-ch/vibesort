// Package vibesort is a sorting library that sorts by vibes.
//
// Instead of writing a comparison function, you describe how you want things
// sorted in plain English and an OpenAI model figures out the order for you.
package vibesort

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	defaultModel    = "gpt-4o-mini"
	defaultEndpoint = "https://api.openai.com/v1/chat/completions"
)

// Client holds the configuration used to talk to the LLM.
type Client struct {
	APIKey     string
	Model      string
	Endpoint   string
	HTTPClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithAPIKey sets the OpenAI API key. If unset, OPENAI_API_KEY is used.
func WithAPIKey(key string) Option { return func(c *Client) { c.APIKey = key } }

// WithModel overrides the default model (gpt-4o-mini).
func WithModel(model string) Option { return func(c *Client) { c.Model = model } }

// New creates a Client. By default it reads the API key from OPENAI_API_KEY
// and uses the gpt-4o-mini model.
func New(opts ...Option) *Client {
	c := &Client{
		APIKey:     os.Getenv("OPENAI_API_KEY"),
		Model:      defaultModel,
		Endpoint:   defaultEndpoint,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Sort returns a new slice containing the items of the original list reordered
// according to the natural-language descriptor (the "key"), e.g. "alphabetical",
// "by release date, oldest first", or "most spicy to least spicy".
//
// The original slice is not modified.
func Sort[T any](items []T, descriptor string, opts ...Option) ([]T, error) {
	return SortContext(context.Background(), New(opts...), items, descriptor)
}

// SortContext is like Sort but uses a caller-supplied Client and context.
func SortContext[T any](ctx context.Context, c *Client, items []T, descriptor string) ([]T, error) {
	if len(items) <= 1 {
		out := make([]T, len(items))
		copy(out, items)
		return out, nil
	}
	if c.APIKey == "" {
		return nil, fmt.Errorf("vibesort: no API key (set OPENAI_API_KEY or use WithAPIKey)")
	}

	order, err := c.askForOrder(ctx, items, descriptor)
	if err != nil {
		return nil, err
	}
	if len(order) != len(items) {
		return nil, fmt.Errorf("vibesort: model returned %d indices, expected %d", len(order), len(items))
	}

	out := make([]T, 0, len(items))
	seen := make(map[int]bool, len(items))
	for _, idx := range order {
		if idx < 0 || idx >= len(items) || seen[idx] {
			return nil, fmt.Errorf("vibesort: model returned invalid index %d", idx)
		}
		seen[idx] = true
		out = append(out, items[idx])
	}
	return out, nil
}

// askForOrder asks the model for the sorted order of item indices.
func (c *Client) askForOrder(ctx context.Context, items any, descriptor string) ([]int, error) {
	itemsJSON, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("vibesort: marshal items: %w", err)
	}

	system := "You are Vibesort, a sorting engine. You are given a JSON array of " +
		"items (0-indexed) and a sorting instruction. Return ONLY a JSON object of " +
		"the form {\"order\": [...]} where order lists the original item indices " +
		"sorted according to the instruction. Include every index exactly once. " +
		"Do not include any prose."
	user := fmt.Sprintf("Sorting instruction: %s\n\nItems:\n%s", descriptor, itemsJSON)

	reqBody := map[string]any{
		"model":           c.Model,
		"temperature":     0,
		"response_format": map[string]string{"type": "json_object"},
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	}
	buf, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vibesort: request failed: %w", err)
	}
	defer resp.Body.Close()

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("vibesort: decode response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("vibesort: openai error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("vibesort: no choices in response (status %s)", resp.Status)
	}

	var result struct {
		Order []int `json:"order"`
	}
	if err := json.Unmarshal([]byte(parsed.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("vibesort: parse model output %q: %w", parsed.Choices[0].Message.Content, err)
	}
	return result.Order, nil
}
