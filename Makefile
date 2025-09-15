# Pawdy Makefile

.PHONY: build test lint run clean install deps help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=pawdy
BINARY_UNIX=$(BINARY_NAME)_unix

# Build directory
BUILD_DIR=./bin

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date +%Y-%m-%dT%H:%M:%S%z)

# LDFLAGS for version injection
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

all: test build

## Build the binary
build:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/pawdy

## Build for Linux
build-linux:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_UNIX) ./cmd/pawdy

## Run tests
test:
	$(GOTEST) -v ./...

## Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## Run linter
lint:
	golangci-lint run

## Format code
fmt:
	$(GOCMD) fmt ./...

## Run the application in development mode
run:
	$(GOCMD) run ./cmd/pawdy

## Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## Install dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## Install the binary to GOPATH/bin
install: build
	$(GOCMD) install $(LDFLAGS) ./cmd/pawdy

## Download and verify dependencies
deps-verify:
	$(GOMOD) verify

## Update dependencies
deps-update:
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

## Create example config
config:
	cp pawdy.example.yaml pawdy.yaml
	@echo "Created pawdy.yaml from example template"

## Setup development environment
dev-setup: deps config
	@echo "Setting up development environment..."
	@echo "1. Make sure Docker is running for Qdrant"
	@echo "2. Install Ollama or setup llama.cpp"
	@echo "3. Run 'make run' to start development server"

## Display help
help:
	@echo "Pawdy Development Commands:"
	@echo ""
	@awk '/^##.*/ { \
		helpMessage = substr($$0, 4); \
		if (helpMessage == "") { \
			print "" \
		} else { \
			print "  " helpMessage \
		} \
	} \
	/^[a-zA-Z_-]+:.*/ { \
		if (NF == 2) { \
			printf "  \033[36m%-15s\033[0m %s\n", $$1, helpMessage \
		} \
		helpMessage = "" \
	}' $(MAKEFILE_LIST)
