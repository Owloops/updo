.DEFAULT_GOAL := help

BINARY_NAME := updo
LAMBDA_BINARY := aws/bootstrap
LAMBDA_ZIP := aws/bootstrap.zip
LAMBDA_SOURCE := lambda/lambda.go

GOOS := linux
GOARCH := arm64
CGO_ENABLED := 0
BUILD_TAGS := lambda.norpc

.PHONY: help doctor build-lambda build clean test lint vet format check install

help: ## Show usage and commands
	@printf "updo - Website monitoring tool with multi-region Lambda support\n\n"
	@printf "Usage: make <command>\n\n"
	@printf "Commands:\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'
	@printf "\nExamples:\n"
	@printf "  make doctor                    # Check prerequisites\n"
	@printf "  make build                     # Build updo binary with embedded Lambda\n"
	@printf "  make test                      # Build and test deployment\n"
	@printf "  make clean                     # Remove build artifacts\n"

doctor: ## Check required tools and environment
	@echo "Checking prerequisites..."
	@if ! command -v go >/dev/null 2>&1; then \
		echo "Error: Go not found in PATH"; exit 1; \
	fi
	@echo "OK: Go found ($$(go version))"
	@if ! command -v zip >/dev/null 2>&1; then \
		echo "Error: zip not found in PATH"; exit 1; \
	fi
	@echo "OK: zip found"
	@if ! go mod verify >/dev/null 2>&1; then \
		echo "Error: Go modules not valid - run 'go mod tidy'"; exit 1; \
	fi
	@echo "OK: Go modules verified"
	@if [ ! -f "$(LAMBDA_SOURCE)" ]; then \
		echo "Error: Lambda source not found at $(LAMBDA_SOURCE)"; exit 1; \
	fi
	@echo "OK: Lambda source found"
	@echo "All prerequisites satisfied!"

build-lambda: ## Build Lambda binary for embedding
	@echo "Building Lambda binary for ARM64..."
	@cd lambda && GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build -tags $(BUILD_TAGS) -ldflags="-s -w" -o ../$(LAMBDA_BINARY) $(notdir $(LAMBDA_SOURCE))
	@echo "Lambda binary built: $(LAMBDA_BINARY)"
	@echo "Creating ZIP archive for embedding..."
	@cd aws && zip -q bootstrap.zip bootstrap
	@echo "Lambda ZIP created: $(LAMBDA_ZIP)"

build: build-lambda ## Build updo binary with embedded Lambda
	@echo "Building $(BINARY_NAME) with embedded Lambda binary..."
	@go build -ldflags="-s -w" -o $(BINARY_NAME) .
	@echo "Binary built: $(BINARY_NAME)"

install: build ## Install updo binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME) to GOPATH/bin..."
	@go install .
	@echo "$(BINARY_NAME) installed successfully"

test: build ## Build and test deployment with dry-run
	@echo "Testing deployment with dry-run..."
	@./$(BINARY_NAME) aws deploy --dry-run
	@echo "Test completed successfully"

lint: ## Run golangci-lint if available
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "Running golangci-lint..."; \
		golangci-lint run; \
	else \
		echo "Warning: golangci-lint not found, skipping lint check"; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

format: ## Format code with gofmt
	@echo "Formatting code..."
	@gofmt -s -w .
	@cd lambda && gofmt -s -w .

check: vet lint ## Run all code quality checks
	@echo "All checks completed"

clean: ## Remove all build artifacts
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME) $(LAMBDA_BINARY) $(LAMBDA_ZIP)
	@echo "Clean completed"
