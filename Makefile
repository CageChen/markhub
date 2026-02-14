.PHONY: build run clean test deps release-dry-run

# Binary name
BINARY=markhub

# Build directory
BUILD_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Default target
all: deps build

# Install dependencies
deps:
	$(GOMOD) tidy
	$(GOMOD) download

# Build the binary
build: deps
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/markhub

# Run the application
run: build
	./$(BUILD_DIR)/$(BINARY) serve --path . --open

# Run with hot reload (requires air)
dev:
	air -c .air.toml

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Build for multiple platforms
build-all: deps
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/markhub
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/markhub
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/markhub
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/markhub

# Docker build
docker-build:
	docker build -t $(BINARY) .

# Docker run
docker-run:
	docker run -p 8080:8080 -v $(PWD)/docs:/docs $(BINARY)

# Release dry-run (local preview, no publish)
release-dry-run:
	goreleaser release --snapshot --clean

# Install globally
install:
	$(GOCMD) install ./cmd/markhub

# Format code
fmt:
	gofmt -s -w .
	goimports -w .

# Lint code
lint:
	golangci-lint run

# Help
help:
	@echo "Available targets:"
	@echo "  make deps            - Install dependencies"
	@echo "  make build           - Build the binary"
	@echo "  make run             - Build and run the application"
	@echo "  make dev             - Run with hot reload (requires air)"
	@echo "  make test            - Run tests"
	@echo "  make clean           - Clean build artifacts"
	@echo "  make build-all       - Build for all platforms"
	@echo "  make docker-build    - Build Docker image"
	@echo "  make release-dry-run - Local release preview (no publish)"
	@echo "  make install         - Install globally"

