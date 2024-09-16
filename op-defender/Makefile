# Makefile

# Binary name
BINARY=defender

# Build directory
BUILD_DIR=./bin

# Source directory
CMD_DIR=./cmd/defender

# Go commands
GOBUILD=go build
GOCLEAN=go clean
GORUN=go run

# Default target
.PHONY: all
all: build

# Build binary
.PHONY: build
build:
	@echo "Building..."
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR)

# Run program
.PHONY: run
run:
	@echo "Running..."
	$(GORUN) $(CMD_DIR)/

# Clean up binaries
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BUILD_DIR)/$(BINARY)

lint-go: ## Lints Go code with specific linters
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint --timeout 5m -e "errors.As" -e "errors.Is" ./...
.PHONY: lint-go

lint-go-fix: ## Lints Go code with specific linters and fixes reported issues
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint --timeout 5m -e "errors.As" -e "errors.Is" ./... --fix
.PHONY: lint-go-fix
# Run program
.PHONY: tidy
tidy:
	@echo "Tidying..."
	go mod tidy

# Help
.PHONY: help
help:
	@echo "Makefile commands:"
	@echo "  make build"
	@echo "  make run"
	@echo "  make clean"
	@echo "  make help"
