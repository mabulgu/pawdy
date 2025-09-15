// Package app provides the main application logic for Pawdy.
package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mabulgu/pawdy/internal/backend/llamacpp"
	"github.com/mabulgu/pawdy/internal/backend/ollama"
	"github.com/mabulgu/pawdy/internal/config"
	"github.com/mabulgu/pawdy/internal/document"
	"github.com/mabulgu/pawdy/internal/prompt"
	"github.com/mabulgu/pawdy/internal/rag"
	"github.com/mabulgu/pawdy/internal/safety"
	"github.com/mabulgu/pawdy/pkg/types"
)

// App represents the main Pawdy application.
type App struct {
	Config        *types.Config
	LLMClient     types.LLMClient
	SafetyGate    types.SafetyGate
	Retriever     types.Retriever
	PromptBuilder *prompt.Builder
}

// Source represents a document source with metadata.
type Source struct {
	ID       string         `json:"id"`
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata"`
	Score    float64        `json:"score"`
}

// EvaluationResults contains evaluation metrics.
type EvaluationResults struct {
	Total             int     `json:"total"`
	AvgResponseTime   float64 `json:"avg_response_time"`
	AvgRelevanceScore float64 `json:"avg_relevance_score"`
	SafetyBlocks      int     `json:"safety_blocks"`
}

// New creates a new Pawdy application instance.
func New() (*App, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize LLM client
	var llmClient types.LLMClient
	switch cfg.Backend {
	case "llamacpp":
		llmClient, err = llamacpp.NewClient(cfg.ModelPath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize llama.cpp client: %w", err)
		}
	case "ollama":
		llmClient = ollama.NewClient(cfg.OllamaURL, cfg.OllamaModel)
	default:
		return nil, fmt.Errorf("unsupported backend: %s", cfg.Backend)
	}

	// Initialize safety gate
	var safetyClient types.LLMClient
	if cfg.Safety == "on" {
		switch cfg.Backend {
		case "llamacpp":
			// For llamacpp, we'd need a separate guard model - for now use the same client
			safetyClient = llmClient
		case "ollama":
			safetyClient = ollama.NewClient(cfg.OllamaURL, cfg.GuardModel)
		}
	}

	safetyGate := safety.NewGuard(safetyClient, cfg.Safety == "on")

	// Initialize embeddings
	var embeddings types.EmbeddingProvider
	switch cfg.Embeddings {
	case "ollama-nomic":
		embeddings = rag.NewOllamaEmbeddings(cfg.OllamaURL, cfg.EmbeddingModel)
	case "fastembed":
		return nil, fmt.Errorf("fastembed not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported embeddings provider: %s", cfg.Embeddings)
	}

	// Initialize retriever
	retriever, err := rag.NewQdrantRetriever(cfg.QdrantURL, cfg.Collection, embeddings)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize retriever: %w", err)
	}

	// Initialize prompt builder
	promptBuilder := prompt.NewBuilder(cfg.SystemPrompt)

	return &App{
		Config:        cfg,
		LLMClient:     llmClient,
		SafetyGate:    safetyGate,
		Retriever:     retriever,
		PromptBuilder: promptBuilder,
	}, nil
}

// Ask processes a question and returns a response with sources.
func (a *App) Ask(ctx context.Context, question string, temperature float64) (string, []*Source, error) {
	// Check input safety
	if a.SafetyGate.IsEnabled() {
		safetyResult, err := a.SafetyGate.CheckInput(ctx, question)
		if err != nil {
			return "", nil, fmt.Errorf("safety check failed: %w", err)
		}

		if !safetyResult.IsSafe {
			refusal := safety.GetRefusalMessage(safetyResult.Category)
			return refusal, nil, nil
		}
	}

	// Retrieve relevant documents
	documents, err := a.Retriever.Search(ctx, question, a.Config.TopK)
	if err != nil {
		return "", nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}

	// Build prompt with context
	prompt := a.PromptBuilder.BuildRAGPrompt(question, documents)

	// Get system prompt
	systemPrompt, err := a.PromptBuilder.BuildSystemPrompt()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build system prompt: %w", err)
	}

	// Configure generation options
	opts := types.GenerateOptions{
		Temperature:  temperature,
		MaxTokens:    a.Config.MaxTokens,
		TopP:         a.Config.TopP,
		SystemPrompt: systemPrompt,
	}

	if temperature == 0 {
		opts.Temperature = a.Config.Temperature
	}

	// Generate response
	response, err := a.LLMClient.Generate(ctx, prompt, opts)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate response: %w", err)
	}

	// Check output safety
	if a.SafetyGate.IsEnabled() {
		safetyResult, err := a.SafetyGate.CheckOutput(ctx, response)
		if err != nil {
			return "", nil, fmt.Errorf("output safety check failed: %w", err)
		}

		if !safetyResult.IsSafe {
			refusal := safety.GetRefusalMessage(safetyResult.Category)
			return refusal, nil, nil
		}
	}

	// Convert documents to sources
	sources := make([]*Source, len(documents))
	for i, doc := range documents {
		sources[i] = &Source{
			ID:       doc.ID,
			Content:  doc.Content,
			Metadata: doc.Metadata,
			Score:    doc.Score,
		}
	}

	return response, sources, nil
}

// IngestFile processes and indexes a single file.
func (a *App) IngestFile(ctx context.Context, filePath string, chunkTokens, chunkOverlap int) (int, error) {
	// Use config defaults if not specified
	if chunkTokens == 0 {
		chunkTokens = a.Config.ChunkTokens
	}
	if chunkOverlap == 0 {
		chunkOverlap = a.Config.ChunkOverlap
	}

	// Process the file
	documents, err := document.ProcessFile(ctx, filePath, chunkTokens, chunkOverlap)
	if err != nil {
		return 0, fmt.Errorf("failed to process file: %w", err)
	}

	// Add to retriever
	err = a.Retriever.AddDocuments(ctx, documents)
	if err != nil {
		return 0, fmt.Errorf("failed to add documents: %w", err)
	}

	return len(documents), nil
}

// HealthCheck checks the health of all services.
func (a *App) HealthCheck(ctx context.Context) ([]*types.HealthStatus, error) {
	var statuses []*types.HealthStatus

	// Check LLM backend
	start := time.Now()
	llmErr := a.LLMClient.IsHealthy(ctx)
	llmLatency := time.Since(start)

	llmStatus := &types.HealthStatus{
		Name:    fmt.Sprintf("LLM Backend (%s)", a.Config.Backend),
		Healthy: llmErr == nil,
		Latency: llmLatency.String(),
	}
	if llmErr != nil {
		llmStatus.Message = llmErr.Error()
	}
	statuses = append(statuses, llmStatus)

	// Check vector database
	start = time.Now()
	dbErr := a.Retriever.IsHealthy(ctx)
	dbLatency := time.Since(start)

	dbStatus := &types.HealthStatus{
		Name:    "Vector Database (Qdrant)",
		Healthy: dbErr == nil,
		Latency: dbLatency.String(),
	}
	if dbErr != nil {
		dbStatus.Message = dbErr.Error()
	}
	statuses = append(statuses, dbStatus)

	// Check embeddings
	if _, ok := a.Retriever.(*rag.QdrantRetriever); ok {
		// This is a bit of a hack to access the embeddings provider
		// In a real implementation, we'd have a better way to access this
		embeddingsStatus := &types.HealthStatus{
			Name:    fmt.Sprintf("Embeddings (%s)", a.Config.Embeddings),
			Healthy: true, // Assume healthy if we got this far
			Message: "Embedded in retriever",
		}
		statuses = append(statuses, embeddingsStatus)
	}

	// Check safety gate
	if a.SafetyGate.IsEnabled() {
		safetyStatus := &types.HealthStatus{
			Name:    "Safety Gate",
			Healthy: true,
			Message: "Enabled",
		}
		statuses = append(statuses, safetyStatus)
	} else {
		safetyStatus := &types.HealthStatus{
			Name:    "Safety Gate",
			Healthy: true,
			Message: "Disabled",
		}
		statuses = append(statuses, safetyStatus)
	}

	return statuses, nil
}

// Reset clears the vector database.
func (a *App) Reset(ctx context.Context, collection string) error {
	return a.Retriever.DeleteCollection(ctx)
}

// Evaluate runs evaluation against a test set.
func (a *App) Evaluate(ctx context.Context, testFile, outputFile string) (*EvaluationResults, error) {
	// Check if test file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("test file not found: %s", testFile)
	}

	// For now, return a placeholder implementation
	// In a real implementation, this would:
	// 1. Read JSONL test file
	// 2. Process each question
	// 3. Measure response time and quality
	// 4. Generate detailed results

	results := &EvaluationResults{
		Total:             0,
		AvgResponseTime:   0.0,
		AvgRelevanceScore: 0.0,
		SafetyBlocks:      0,
	}

	return results, fmt.Errorf("evaluation not yet implemented - placeholder for future development")
}

// Close cleans up application resources.
func (a *App) Close() error {
	if a.LLMClient != nil {
		return a.LLMClient.Close()
	}
	return nil
}
