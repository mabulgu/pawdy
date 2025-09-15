// Package ollama provides an Ollama HTTP API backend for LLM operations.
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mabulgu/pawdy/pkg/types"
)

// Client represents an Ollama HTTP API client.
type Client struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewClient creates a new Ollama client.
func NewClient(baseURL, model string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		model:   model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Generate produces a complete response for the given prompt.
func (c *Client) Generate(ctx context.Context, prompt string, opts types.GenerateOptions) (string, error) {
	req := generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": opts.Temperature,
			"top_p":       opts.TopP,
			"num_predict": opts.MaxTokens,
		},
	}

	if opts.SystemPrompt != "" {
		req.System = opts.SystemPrompt
	}

	if len(opts.StopSequences) > 0 {
		req.Options["stop"] = opts.StopSequences
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Response, nil
}

// GenerateStream produces a streaming response for the given prompt.
func (c *Client) GenerateStream(ctx context.Context, prompt string, opts types.GenerateOptions) (<-chan types.StreamToken, error) {
	req := generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: true,
		Options: map[string]interface{}{
			"temperature": opts.Temperature,
			"top_p":       opts.TopP,
			"num_predict": opts.MaxTokens,
		},
	}

	if opts.SystemPrompt != "" {
		req.System = opts.SystemPrompt
	}

	if len(opts.StopSequences) > 0 {
		req.Options["stop"] = opts.StopSequences
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	tokens := make(chan types.StreamToken, 10)

	go func() {
		defer close(tokens)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				tokens <- types.StreamToken{Error: ctx.Err()}
				return
			default:
			}

			line := scanner.Text()
			if line == "" {
				continue
			}

			var response generateResponse
			if err := json.Unmarshal([]byte(line), &response); err != nil {
				tokens <- types.StreamToken{Error: fmt.Errorf("failed to decode streaming response: %w", err)}
				return
			}

			tokens <- types.StreamToken{
				Text: response.Response,
				Done: response.Done,
			}

			if response.Done {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			tokens <- types.StreamToken{Error: fmt.Errorf("failed to scan response: %w", err)}
		}
	}()

	return tokens, nil
}

// IsHealthy checks if the Ollama service is ready to serve requests.
func (c *Client) IsHealthy(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama service unhealthy (status %d)", resp.StatusCode)
	}

	// Check if the specific model is available
	var response struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode models response: %w", err)
	}

	for _, model := range response.Models {
		if strings.HasPrefix(model.Name, c.model) {
			return nil
		}
	}

	return fmt.Errorf("model '%s' not found in ollama", c.model)
}

// Close cleans up any resources used by the client.
func (c *Client) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}

// generateRequest represents a request to the Ollama generate API.
type generateRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	System  string                 `json:"system,omitempty"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// generateResponse represents a response from the Ollama generate API.
type generateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
}
