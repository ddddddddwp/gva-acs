# TR069 Core Library Makefile

# Variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=tr069
BINARY_UNIX=$(BINARY_NAME)_unix

# Default target
all: test build

# Build targets
build:
	$(GOBUILD) -o $(BINARY_NAME) -v

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v

# Test targets
test:
	$(GOTEST) -v ./...

test-coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

bench:
	$(GOTEST) -bench=.

# Dependency management
deps:
	$(GOGET) -u -t ./...

tidy:
	$(GOMOD) tidy

# Clean targets
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f coverage.out

# Run examples
run-example:
	$(GOCMD) run examples/main.go

run-custom-rpc-example:
	$(GOCMD) run examples/custom_rpc_example.go

# Documentation
docs:
	@echo "TR069 Core Library Documentation"
	@echo "================================"
	@echo "README.md - Main project documentation"
	@echo "PROJECT_SUMMARY.md - Project completion summary"
	@echo "docs/ - Additional documentation files"
	@echo "examples/ - Usage examples"

# Install
install:
	$(GOGET) -v ./...

# Help
help:
	@echo "TR069 Core Library Makefile"
	@echo "=========================="
	@echo "Available targets:"
	@echo "  all               - Run tests and build (default)"
	@echo "  build             - Build the project"
	@echo "  build-linux       - Build for Linux"
	@echo "  test              - Run all tests"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  bench             - Run benchmarks"
	@echo "  deps              - Install dependencies"
	@echo "  tidy              - Tidy go.mod file"
	@echo "  clean             - Clean build artifacts"
	@echo "  run-example       - Run the main example"
	@echo "  run-custom-rpc-example - Run the custom RPC example"
	@echo "  docs              - Show documentation information"
	@echo "  install           - Install the library"
	@echo "  help              - Show this help message"

.PHONY: all build build-linux test test-coverage bench deps tidy clean run-example run-custom-rpc-example docs install help