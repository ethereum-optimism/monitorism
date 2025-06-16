# Makefile

# Binary name
BINARY=monitorism

# Build directory
BUILD_DIR=./bin

# Source directory
CMD_DIR=./cmd/monitorism

# Go commands
GOBUILD=go build
GOCLEAN=go clean
GORUN=go run
GOTEST=go test

# Default target
.PHONY: all
all: build

# Build binary
.PHONY: build
build:
	@echo "Building..."
	$(GOBUILD) -tags all -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR) 

# Run program
.PHONY: run
run:
	@echo "Running..."
	$(GORUN) $(CMD_DIR) $(ARGS)

# Clean up binaries
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BUILD_DIR)/$(BINARY)

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@if [ "$(filter-out $@,$(MAKECMDGOALS))" != "" ]; then \
		$(GOTEST) -v -count=1 ./$(filter-out $@,$(MAKECMDGOALS))/...; \
	else \
		$(GOTEST) ./... -v; \
	fi

# This allows passing arguments to the test target
%:
	@:

#include tests that require live resources
#these resources are meant to be real and not mocked
.PHONY: test-live
test-live:
	@echo "Running live_tests..."
	$(GOTEST) ./... -v -tags live

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
	@echo "  make test [module]    # Run all tests or tests for a specific module (e.g. make test faultproof_withdrawals)"
	@echo "  make test-live"
	@echo "  make tidy"
	@echo "  make help"
