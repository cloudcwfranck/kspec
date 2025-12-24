.PHONY: build test lint clean install help

# Variables
BINARY_NAME=kspec
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
VERSION?=1.0.0

# Default target
.DEFAULT_GOAL := help

## help: Display this help message
help:
	@echo "Available targets:"
	@grep -E '^## ' Makefile | sed 's/^## /  /'

## build: Build the kspec binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/kspec
	@echo "Built: ./$(BINARY_NAME)"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GO) test ./... -v

## test-cover: Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	$(GO) test ./... -cover -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run go vet and other linters
lint:
	@echo "Running linters..."
	$(GO) vet ./...
	$(GO) fmt ./...
	@echo "Linting complete"

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

## install: Install kspec to /usr/local/bin
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo mv $(BINARY_NAME) /usr/local/bin/
	@echo "Installed: /usr/local/bin/$(BINARY_NAME)"

## mod-tidy: Run go mod tidy
mod-tidy:
	@echo "Running go mod tidy..."
	$(GO) mod tidy

## validate: Validate example specs
validate:
	@echo "Validating example specs..."
	./$(BINARY_NAME) validate --spec specs/examples/minimal.yaml
