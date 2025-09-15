// Package types defines core types and interfaces for the Pawdy application.
package types

import (
	"context"
	"io"
	"time"
)

// LLMClient defines the interface for language model backends.
type LLMClient interface {
	// Generate produces a complete response for the given prompt.
	Generate(ctx context.Context, prompt string, opts GenerateOptions) (string, error)

	// GenerateStream produces a streaming response for the given prompt.
	GenerateStream(ctx context.Context, prompt string, opts GenerateOptions) (<-chan StreamToken, error)

	// IsHealthy checks if the backend is ready to serve requests.
	IsHealthy(ctx context.Context) error

	// Close cleans up any resources used by the client.
	Close() error
}

// StreamToken represents a single token in a streaming response.
type StreamToken struct {
	Text  string
	Done  bool
	Error error
}

// GenerateOptions configures text generation parameters.
type GenerateOptions struct {
	Temperature   float64  `json:"temperature,omitempty"`
	TopP          float64  `json:"top_p,omitempty"`
	MaxTokens     int      `json:"max_tokens,omitempty"`
	StopSequences []string `json:"stop_sequences,omitempty"`
	SystemPrompt  string   `json:"system_prompt,omitempty"`
}

// SafetyGate defines the interface for content safety filtering.
type SafetyGate interface {
	// CheckInput validates user input for safety violations.
	CheckInput(ctx context.Context, text string) (*SafetyResult, error)

	// CheckOutput validates model output for safety violations.
	CheckOutput(ctx context.Context, text string) (*SafetyResult, error)

	// IsEnabled returns whether safety filtering is currently enabled.
	IsEnabled() bool
}

// SafetyResult contains the result of a safety check.
type SafetyResult struct {
	IsSafe   bool    `json:"is_safe"`
	Category string  `json:"category,omitempty"`
	Reason   string  `json:"reason,omitempty"`
	Score    float64 `json:"score,omitempty"`
}

// SafetyCategories defines known safety violation categories.
var SafetyCategories = map[string]string{
	"S1":  "Violent Crimes",
	"S2":  "Non-Violent Crimes",
	"S3":  "Sex Crimes",
	"S4":  "Child Exploitation",
	"S5":  "Defamation",
	"S6":  "Specialized Advice",
	"S7":  "Privacy",
	"S8":  "Intellectual Property",
	"S9":  "Indiscriminate Weapons",
	"S10": "Hate",
	"S11": "Self-Harm",
	"S12": "Sexual Content",
	"S13": "Elections",
	"S14": "Code Interpreter Abuse",
}

// Retriever defines the interface for document retrieval and RAG.
type Retriever interface {
	// Search finds the most relevant documents for a query.
	Search(ctx context.Context, query string, topK int) ([]*Document, error)

	// AddDocuments ingests and indexes new documents.
	AddDocuments(ctx context.Context, docs []*Document) error

	// DeleteCollection removes all documents from the collection.
	DeleteCollection(ctx context.Context) error

	// IsHealthy checks if the vector database is accessible.
	IsHealthy(ctx context.Context) error
}

// Document represents a document chunk with metadata.
type Document struct {
	ID       string         `json:"id"`
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata"`
	Score    float64        `json:"score,omitempty"`
}

// DocumentSource contains information about the original document.
type DocumentSource struct {
	Path     string    `json:"path"`
	Title    string    `json:"title,omitempty"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	Type     string    `json:"type"`
}

// EmbeddingProvider defines the interface for text embeddings.
type EmbeddingProvider interface {
	// Embed generates vector embeddings for the given texts.
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// GetDimensions returns the dimensionality of the embeddings.
	GetDimensions() int

	// IsHealthy checks if the embedding service is available.
	IsHealthy(ctx context.Context) error
}

// DocumentProcessor handles parsing and chunking of various document formats.
type DocumentProcessor interface {
	// Process extracts text content from a document and splits it into chunks.
	Process(ctx context.Context, reader io.Reader, source DocumentSource) ([]*Document, error)

	// SupportedTypes returns the file types this processor can handle.
	SupportedTypes() []string
}

// PromptBuilder constructs prompts with context and formatting.
type PromptBuilder interface {
	// BuildRAGPrompt creates a prompt with retrieved context.
	BuildRAGPrompt(query string, context []*Document) string

	// BuildSystemPrompt loads and formats the system prompt.
	BuildSystemPrompt() (string, error)

	// FormatResponse formats the final response with citations.
	FormatResponse(response string, sources []*Document) string
}

// ChatSession represents an interactive chat session.
type ChatSession struct {
	ID       string                 `json:"id"`
	Messages []Message              `json:"messages"`
	Config   map[string]interface{} `json:"config"`
	Created  time.Time              `json:"created"`
}

// Message represents a single message in a chat session.
type Message struct {
	Role      string      `json:"role"` // "user", "assistant", "system"
	Content   string      `json:"content"`
	Sources   []*Document `json:"sources,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// Config represents the application configuration.
type Config struct {
	// LLM Backend Configuration
	Backend     string `yaml:"backend" mapstructure:"backend"`
	ModelPath   string `yaml:"model_path" mapstructure:"model_path"`
	OllamaURL   string `yaml:"ollama_url" mapstructure:"ollama_url"`
	OllamaModel string `yaml:"ollama_model" mapstructure:"ollama_model"`
	GuardModel  string `yaml:"guard_model" mapstructure:"guard_model"`

	// Embeddings Configuration
	Embeddings     string `yaml:"embeddings" mapstructure:"embeddings"`
	EmbeddingModel string `yaml:"embedding_model" mapstructure:"embedding_model"`

	// Vector Database
	QdrantURL  string `yaml:"qdrant_url" mapstructure:"qdrant_url"`
	Collection string `yaml:"collection" mapstructure:"collection"`

	// RAG Parameters
	ChunkTokens  int  `yaml:"chunk_tokens" mapstructure:"chunk_tokens"`
	ChunkOverlap int  `yaml:"chunk_overlap" mapstructure:"chunk_overlap"`
	TopK         int  `yaml:"top_k" mapstructure:"top_k"`
	Rerank       bool `yaml:"rerank" mapstructure:"rerank"`

	// Generation Parameters
	Temperature float64 `yaml:"temperature" mapstructure:"temperature"`
	MaxTokens   int     `yaml:"max_tokens" mapstructure:"max_tokens"`
	TopP        float64 `yaml:"top_p" mapstructure:"top_p"`

	// System Configuration
	SystemPrompt string `yaml:"system_prompt" mapstructure:"system_prompt"`
	Safety       string `yaml:"safety" mapstructure:"safety"`
	LogLevel     string `yaml:"log_level" mapstructure:"log_level"`

	// Performance
	ContextWindow int `yaml:"context_window" mapstructure:"context_window"`
	BatchSize     int `yaml:"batch_size" mapstructure:"batch_size"`
}

// HealthStatus represents the health of a service component.
type HealthStatus struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}
