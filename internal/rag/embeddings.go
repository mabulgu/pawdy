// Package rag provides retrieval-augmented generation functionality.
package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mabulgu/pawdy/pkg/types"
)

// OllamaEmbeddings implements embeddings using Ollama.
type OllamaEmbeddings struct {
	baseURL string
	model   string
	client  *http.Client
}

// Ensure OllamaEmbeddings implements the EmbeddingProvider interface
var _ types.EmbeddingProvider = (*OllamaEmbeddings)(nil)

// NewOllamaEmbeddings creates a new Ollama embeddings provider.
func NewOllamaEmbeddings(baseURL, model string) *OllamaEmbeddings {
	return &OllamaEmbeddings{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Embed generates vector embeddings for the given texts.
func (e *OllamaEmbeddings) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	
	for i, text := range texts {
		req := embeddingRequest{
			Model:  e.model,
			Prompt: text,
		}

		body, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/api/embeddings", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create embedding request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := e.client.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("failed to make embedding request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("ollama embedding API error (status %d)", resp.StatusCode)
		}

		var response embeddingResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode embedding response: %w", err)
		}

		embeddings[i] = response.Embedding
	}

	return embeddings, nil
}

// GetDimensions returns the dimensionality of the embeddings.
func (e *OllamaEmbeddings) GetDimensions() int {
	// nomic-embed-text produces 768-dimensional embeddings
	return 768
}

// IsHealthy checks if the embedding service is available.
func (e *OllamaEmbeddings) IsHealthy(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", e.baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama embedding service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama embedding service unhealthy (status %d)", resp.StatusCode)
	}

	return nil
}

// embeddingRequest represents a request to the Ollama embeddings API.
type embeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// embeddingResponse represents a response from the Ollama embeddings API.
type embeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}
