// Package config handles application configuration management.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/mabulgu/pawdy/pkg/types"
	"github.com/spf13/viper"
)

// Load reads configuration from files and environment variables.
func Load() (*types.Config, error) {
	// Set defaults
	setDefaults()

	// Configure viper
	viper.SetConfigName("pawdy")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.pawdy")
	viper.AddConfigPath("/etc/pawdy")

	// Environment variable support
	viper.SetEnvPrefix("PAWDY")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is acceptable - use defaults and env vars
	}

	var config types.Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults establishes default configuration values.
func setDefaults() {
	// LLM Backend Configuration
	viper.SetDefault("backend", "ollama")
	viper.SetDefault("model_path", "./models/Llama-3.1-8B-Instruct-Q4_K_M.gguf")
	viper.SetDefault("ollama_url", "http://localhost:11434")
	viper.SetDefault("ollama_model", "llama3.1:8b")
	viper.SetDefault("guard_model", "llama-guard3:1b")

	// Embeddings Configuration
	viper.SetDefault("embeddings", "ollama-nomic")
	viper.SetDefault("embedding_model", "nomic-embed-text")

	// Vector Database
	viper.SetDefault("qdrant_url", "http://localhost:6333")
	viper.SetDefault("collection", "pawdy_docs")

	// RAG Parameters
	viper.SetDefault("chunk_tokens", 1000)
	viper.SetDefault("chunk_overlap", 200)
	viper.SetDefault("top_k", 6)
	viper.SetDefault("rerank", true)

	// Generation Parameters
	viper.SetDefault("temperature", 0.6)
	viper.SetDefault("max_tokens", 1024)
	viper.SetDefault("top_p", 0.9)

	// System Configuration
	viper.SetDefault("system_prompt", "./assets/system_prompt.md")
	viper.SetDefault("safety", "on")
	viper.SetDefault("log_level", "info")

	// Performance
	viper.SetDefault("context_window", 8192)
	viper.SetDefault("batch_size", 512)
}

// validate checks that the configuration is valid.
func validate(config *types.Config) error {
	// Validate backend
	if config.Backend != "llamacpp" && config.Backend != "ollama" {
		return fmt.Errorf("backend must be 'llamacpp' or 'ollama', got '%s'", config.Backend)
	}

	// Validate model path for llamacpp
	if config.Backend == "llamacpp" {
		if config.ModelPath == "" {
			return fmt.Errorf("model_path is required when using llamacpp backend")
		}
		if _, err := os.Stat(config.ModelPath); os.IsNotExist(err) {
			return fmt.Errorf("model file not found: %s", config.ModelPath)
		}
	}

	// Validate embeddings provider
	if config.Embeddings != "ollama-nomic" && config.Embeddings != "fastembed" {
		return fmt.Errorf("embeddings must be 'ollama-nomic' or 'fastembed', got '%s'", config.Embeddings)
	}

	// Validate safety setting
	if config.Safety != "on" && config.Safety != "off" {
		return fmt.Errorf("safety must be 'on' or 'off', got '%s'", config.Safety)
	}

	// Validate numeric ranges
	if config.Temperature < 0.0 || config.Temperature > 2.0 {
		return fmt.Errorf("temperature must be between 0.0 and 2.0, got %f", config.Temperature)
	}

	if config.TopP < 0.0 || config.TopP > 1.0 {
		return fmt.Errorf("top_p must be between 0.0 and 1.0, got %f", config.TopP)
	}

	if config.TopK < 1 || config.TopK > 50 {
		return fmt.Errorf("top_k must be between 1 and 50, got %d", config.TopK)
	}

	if config.ChunkTokens < 100 || config.ChunkTokens > 4000 {
		return fmt.Errorf("chunk_tokens must be between 100 and 4000, got %d", config.ChunkTokens)
	}

	if config.ChunkOverlap < 0 || config.ChunkOverlap >= config.ChunkTokens {
		return fmt.Errorf("chunk_overlap must be between 0 and chunk_tokens, got %d", config.ChunkOverlap)
	}

	// Validate system prompt file
	if config.SystemPrompt != "" {
		if _, err := os.Stat(config.SystemPrompt); os.IsNotExist(err) {
			return fmt.Errorf("system prompt file not found: %s", config.SystemPrompt)
		}
	}

	return nil
}

// GetConfiguredPath returns the path to the active config file.
func GetConfiguredPath() string {
	return viper.ConfigFileUsed()
}

// WriteExample creates an example configuration file.
func WriteExample(path string) error {
	example := `# Pawdy Configuration File
# Backend configuration
backend: llamacpp                 # Options: llamacpp, ollama
model_path: ./models/Llama-3.1-8B-Instruct-Q4_K_M.gguf
ollama_url: http://localhost:11434
guard_model: llama-guard3

# Embeddings configuration  
embeddings: ollama-nomic          # Options: ollama-nomic, fastembed
embedding_model: nomic-embed-text

# Vector database
qdrant_url: http://localhost:6333
collection: pawdy_docs

# RAG parameters
chunk_tokens: 1000                # Tokens per chunk
chunk_overlap: 200                # Overlap between chunks
top_k: 6                         # Number of chunks to retrieve
rerank: true                     # Enable keyword re-ranking

# Generation parameters
temperature: 0.6                 # Creativity (0.0 = deterministic, 1.0 = creative)
max_tokens: 1024                 # Maximum response length
top_p: 0.9                       # Nucleus sampling

# System configuration
system_prompt: ./assets/system_prompt.md
safety: on                       # Options: on, off
log_level: info                  # Options: debug, info, warn, error

# Performance
context_window: 8192             # Model context window
batch_size: 512                  # Batch size for embeddings
`

	return os.WriteFile(path, []byte(example), 0644)
}
