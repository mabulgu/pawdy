# Pawdy 🐾

*ʕ•ᴥ•ʔ  hi, I'm Pawdy — your bare-metal onboarding buddy*

Pawdy is a fully local command-line chat assistant designed to help engineers onboard to the OpenShift Bare Metal team. It runs entirely offline using Meta's Llama models and provides RAG (Retrieval-Augmented Generation) capabilities over your team documentation.

## Project Structure Rationale

This project follows Go's standard project layout with some key design decisions:

```
pawdy/
├── cmd/pawdy/           # Main application entry point
├── internal/            # Private application code
│   ├── app/             # Application orchestration layer
│   ├── backend/         # LLM backend implementations (llama.cpp, Ollama)
│   ├── cli/             # CLI command implementations
│   ├── config/          # Configuration management
│   ├── document/        # Document processing and chunking
│   ├── prompt/          # Prompt templates and builders
│   ├── rag/             # RAG pipeline (embeddings, retrieval)
│   └── safety/          # Llama Guard 3 safety gate implementation
├── pkg/                 # Public library code (interfaces, types)
├── assets/              # System prompts and static files
├── materials/           # Default directory for ingesting docs
├── models/              # Recommended directory for GGUF models
└── .github/workflows/   # CI/CD pipeline
```

**Rationale:**
- `internal/` ensures clean API boundaries and prevents external dependencies on implementation details
- `pkg/` contains stable interfaces that could be reused by other tools
- Separation of concerns: each internal package has a single responsibility
- `cmd/` pattern allows for future CLI expansion without architectural changes
- `internal/app/` orchestrates all components providing a clean facade for the CLI layer

## Features

- **🔒 Privacy-First**: Runs completely offline, no cloud API calls or telemetry
- **🧠 Local LLM**: Support for Meta Llama 3.x models via llama.cpp or Ollama
- **📚 RAG Pipeline**: Vector search over your documentation using Qdrant
- **🛡️ Safety**: Built-in Llama Guard 3 content filtering (toggleable)
- **⚡ Streaming**: Real-time token streaming with source citations
- **🔧 Configurable**: YAML config with environment variable overrides

## Project Structure

┌───────────────────────────────────────────────────────────────────────────┐
│                         Pawdy — High-Level Architecture                   │
├───────────────────────────────────────────────────────────────────────────┤
│                            (Terminal / Shell)                             │
│                                ┌─────────┐                                │
│                                │  CLI    │  pawdy chat / ask / ingest     │
│                                │ (Cobra) │───────────────────────────────┐│
│                                └────┬────┘                               ││
│                                     │                                    ││
│                                     │ commands/options                    ││
│                                     ▼                                    ▼│
│                            ┌───────────────────┐                ┌──────────┴─────┐
│                            │   Orchestrator   │                │   Config/Env   │
│                            │ (routing + glue) │◀──────────────▶│  pawdy.yaml    │
│                            └───────┬──────────┘                └────────────────┘
│                                    │
│        ┌───────────────────────────┼───────────────────────────┐
│        │                           │                           │
│        ▼                           ▼                           ▼
│  ┌───────────────┐          ┌──────────────┐            ┌───────────────┐
│  │   RAG Layer   │          │  LLM Client  │            │  Safety Gate  │
│  │ (retrieve+CI) │          │ (llama.cpp/  │            │ (Llama Guard) │
│  └──────┬────────┘          │   Ollama)    │            └──────┬────────┘
│         │                   └──────┬───────┘                   │
│         │                 generate/stream ▲                    │
│         │                                 │ classify in/out    │
│  ┌──────┴─────────┐                       │                    │
│  │  Vector Store  │<──── embeddings ──────┘                    │
│  │   (Qdrant)     │                                             │
│  └──────┬─────────┘                                             │
│         │ retrieve                                              │
│  ┌──────┴─────────┐                                      ┌──────┴─────────┐
│  │  Embeddings    │◀──── text chunks ───┐               │  System Prompt │
│  │ (Ollama/ONNX)  │                     │               │  + Few-Shots   │
│  └──────┬─────────┘                     │               └─────────────────┘
│         │                               │
│  ┌──────┴─────────┐                     │
│  │  Ingestors     │◀──── materials/ ────┘
│  │ (md/txt/pdf/   │
│  │  html loaders) │
│  └────────────────┘
└───────────────────────────────────────────────────────────────────────────┘
Legend: CI = citation inserter; materials/ = local docs folder


## Prerequisites

- Go 1.22+
- Docker (for Qdrant)
- Either:
  - llama.cpp with Go bindings, OR
  - Ollama

## Quick Start

### 1. Install Dependencies

```bash
# Clone the repository
git clone https://github.com/mabulgu/pawdy
cd pawdy

# Install Go dependencies
make deps

# Build the binary
make build
```

### 2. Choose Your Backend

#### Option A: Ollama (Recommended for beginners)

```bash
# Install Ollama
# macOS (recommended):
brew install ollama

# Or download from https://ollama.ai/download
# Linux:
# curl -fsSL https://ollama.ai/install.sh | sh

# Pull required models
ollama pull llama3.1:8b        # or llama3.1:8b-instruct-q4_0 for smaller size
ollama pull llama-guard3:1b    # Safety filtering model
ollama pull nomic-embed-text   # Text embeddings model
```

#### Option B: llama.cpp + GGUF files

Download GGUF models to `./models/`:
```bash
# Example for Llama 3.1 8B (Q4_K_M quantization)
wget https://huggingface.co/bartowski/Meta-Llama-3.1-8B-Instruct-GGUF/resolve/main/Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf \
  -O models/Llama-3.1-8B-Instruct-Q4_K_M.gguf
```

### 3. Start Qdrant Vector Database

```bash
# Create data directory for persistence
mkdir -p qdrant

# Start Qdrant with persistent storage
docker run -d -p 6333:6333 -v $(pwd)/qdrant:/qdrant/storage qdrant/qdrant
```

### 4. Configure Pawdy

Copy the example config:
```bash
cp pawdy.example.yaml pawdy.yaml
```

Edit `pawdy.yaml` for your setup (see [Configuration](#configuration) below).

### 5. Ingest Your Documentation

```bash
# Add your team docs to the materials directory
mkdir -p materials
cp /path/to/your/docs/* materials/

# Ingest and index the documents
./pawdy ingest ./materials
```

### 6. Start Chatting!

```bash
# Interactive chat mode
./pawdy chat

# One-shot questions
./pawdy ask "How do I gather initramfs logs?"

# Disable safety filtering (use with caution)
./pawdy chat --safety=off
```

## Model Selection Guide

Choose your model size based on your hardware:

| CPU Cores | RAM  | GPU VRAM | Recommended Model | Quantization | Performance |
|-----------|------|----------|-------------------|--------------|-------------|
| 4-8       | 8GB  | None     | Llama 3.1 3B     | Q4_K_M       | Fast        |
| 8-16      | 16GB | None     | Llama 3.1 8B     | Q4_K_M       | Good        |
| 16+       | 32GB | None     | Llama 3.1 8B     | Q5_K_M       | Better      |
| Any       | 16GB | 8GB+     | Llama 3.1 8B     | Q6_K         | Best        |

**Quantization Guide:**
- **Q4_K_M**: Good quality, moderate file size (~4.6GB for 8B)
- **Q5_K_M**: Better quality, larger file size (~5.7GB for 8B)  
- **Q6_K**: Excellent quality, largest file size (~6.8GB for 8B)

## Configuration

Create `pawdy.yaml` in your project root:

```yaml
# LLM Backend Configuration
backend: llamacpp                 # Options: llamacpp, ollama
model_path: ./models/Llama-3.1-8B-Instruct-Q4_K_M.gguf
ollama_url: http://localhost:11434
guard_model: llama-guard3

# Embeddings Configuration  
embeddings: ollama-nomic          # Options: ollama-nomic, fastembed
embedding_model: nomic-embed-text

# Vector Database
qdrant_url: http://localhost:6333
collection: pawdy_docs

# RAG Parameters
chunk_tokens: 1000                # Tokens per chunk
chunk_overlap: 200                # Overlap between chunks
top_k: 6                         # Number of chunks to retrieve
rerank: true                     # Enable keyword re-ranking

# Generation Parameters
temperature: 0.6                 # Creativity (0.0 = deterministic, 1.0 = creative)
max_tokens: 1024                 # Maximum response length
top_p: 0.9                       # Nucleus sampling

# System Configuration
system_prompt: ./assets/system_prompt.md
safety: on                       # Options: on, off
log_level: info                  # Options: debug, info, warn, error

# Performance
context_window: 8192             # Model context window
batch_size: 512                  # Batch size for embeddings
```

### Environment Variable Overrides

All config values can be overridden with environment variables using the `PAWDY_` prefix:

```bash
export PAWDY_BACKEND=ollama
export PAWDY_TEMPERATURE=0.8
export PAWDY_SAFETY=off
```

## Commands

### Core Commands

```bash
# Interactive chat with streaming responses
pawdy chat [--safety=on|off] [--temperature=0.6]

# One-shot question
pawdy ask "your question here" [--safety=on|off]

# Ingest documentation
pawdy ingest <directory> [--chunk-size=1000] [--overlap=200]

# Reset vector database
pawdy reset [--collection=pawdy_docs]
```

### Utility Commands

```bash
# Health check for all services
pawdy health

# Model evaluation against test set
pawdy eval [--test-file=eval.jsonl]

# Show configuration
pawdy config show

# Version information  
pawdy version
```

## Safety Features

Pawdy includes Llama Guard 3 for content safety:

- **Input filtering**: Blocks unsafe user prompts
- **Output filtering**: Filters potentially harmful responses
- **Categories**: Handles violence, hate speech, privacy violations, etc.
- **Configurable**: Can be disabled with `--safety=off` or config

⚠️ **Warning**: Disabling safety filtering may produce inappropriate content. Use responsibly in controlled environments only.

## Onboarding Seed Pack

For OpenShift Bare Metal teams, create a `materials/` directory with:

```
materials/
├── getting-started.md           # Team overview and first steps
├── architecture/                # System architecture docs
│   ├── baremetal-overview.md
│   └── networking-guide.md
├── troubleshooting/             # Common issues and solutions
│   ├── initramfs-logs.md
│   └── boot-issues.md
├── runbooks/                    # Operational procedures
└── apis/                        # API documentation
```

Supported formats: Markdown (`.md`), Plain text (`.txt`), HTML (`.html`), PDF (`.pdf`)

## Development

### Building

```bash
# Build binary
make build

# Run tests
make test

# Lint code
make lint

# Run with hot reload
make run
```

### Setup Development Environment

```bash
# Setup development environment (first time)
make dev-setup

# This will:
# - Install Go dependencies
# - Create config from template  
# - Display setup instructions
```

### Testing

```bash
# Unit tests
go test ./...

# Integration tests (requires Qdrant running)
go test ./tests/integration/...

# Benchmark tests
go test -bench=. ./internal/rag/
```

## Troubleshooting

### Common Issues

**Qdrant connection failed**
```bash
# Check if Qdrant is running
docker ps | grep qdrant

# Start Qdrant if not running
mkdir -p qdrant
docker run -d -p 6333:6333 -v $(pwd)/qdrant:/qdrant/storage qdrant/qdrant
```

**Model not found**
```bash
# Verify model path in config
pawdy config show | grep model_path

# For Ollama, check available models
ollama list
```

**Out of memory errors**
- Try a smaller model (3B instead of 8B)
- Use more aggressive quantization (Q4 instead of Q6)
- Reduce `chunk_tokens` and `batch_size` in config

**Slow responses**
- Increase `batch_size` for better GPU utilization
- Use GPU acceleration if available
- Consider Q4 quantization for speed vs quality tradeoff

### Debugging

Enable debug logging:
```bash
export PAWDY_LOG_LEVEL=debug
./pawdy chat
```

Or edit `pawdy.yaml`:
```yaml
log_level: debug
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Standards

- Follow Go conventions and `gofmt` formatting
- Add tests for new functionality
- Update documentation for user-facing changes
- Ensure CI passes (`make test lint`)

## License

Apache License 2.0 - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Meta AI](https://ai.meta.com/) for Llama models
- [Qdrant](https://qdrant.tech/) for vector search
- [Ollama](https://ollama.ai/) for local model serving
- [llama.cpp](https://github.com/ggerganov/llama.cpp) for efficient inference

---

*Made with ❤️ for the OpenShift Bare Metal team*
