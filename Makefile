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

## install-bin: Install kspec binary to /usr/local/bin
install-bin: build
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

## build-operator: Build the operator binary
build-operator:
	@echo "Building operator..."
	CGO_ENABLED=0 $(GO) build -o bin/manager ./cmd/manager
	@echo "Built: ./bin/manager"

## manager: Alias for build-operator (kubebuilder compatibility)
manager: build-operator

## manifests: Generate CRD manifests (kubebuilder compatibility)
manifests:
	@echo "Generating CRD manifests..."
	@if command -v controller-gen >/dev/null 2>&1; then \
		controller-gen crd paths="./api/..." output:crd:artifacts:config=config/crd; \
	else \
		echo "controller-gen not found, skipping manifest generation"; \
		echo "Install with: go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest"; \
	fi

## install: Install CRDs into cluster (kubebuilder compatibility)
install: manifests
	@echo "Installing CRDs..."
	kubectl apply -f config/crd/kspec.io_*.yaml

## uninstall: Uninstall CRDs from cluster (kubebuilder compatibility)
uninstall:
	@echo "Uninstalling CRDs..."
	kubectl delete -f config/crd/kspec.io_*.yaml --ignore-not-found=true

## build-dashboard: Build the web dashboard binary
build-dashboard:
	@echo "Building web dashboard..."
	CGO_ENABLED=0 $(GO) build -o bin/web-dashboard ./cmd/web-dashboard
	@echo "Built: ./bin/web-dashboard"

## docker-operator: Build operator Docker image
docker-operator:
	@echo "Building operator Docker image..."
	docker build -t kspec-operator:latest .
	@echo "Built: kspec-operator:latest"

## docker-dashboard: Build dashboard Docker image
docker-dashboard:
	@echo "Building dashboard Docker image..."
	docker build -f Dockerfile.dashboard -t kspec-dashboard:latest .
	@echo "Built: kspec-dashboard:latest"

## deploy-dashboard: Deploy web dashboard to cluster (GitOps-friendly)
deploy-dashboard:
	@echo "Deploying web dashboard..."
	kubectl apply -f config/dashboard/deployment.yaml
	@echo "Dashboard deployed! Access with:"
	@echo "  kubectl port-forward -n kspec-system svc/kspec-dashboard 8000:80"
	@echo "  Then open http://localhost:8000"

## quick-start: Run quick start installation script
quick-start:
	@echo "Running quick start..."
	./hack/quick-start.sh
