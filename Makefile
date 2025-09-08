# Makefile for TV-Forward

# Variables
BINARY_NAME=tv-forward
BUILD_DIR=build
MAIN_FILE=./cmd/main.go
CONFIG_FILE=config.yaml

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags="-s -w"

.PHONY: all build clean test deps run dev docker-build docker-run help

# Default target
all: clean deps test build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...
	@echo "Tests complete"

# Get dependencies
deps:
	@echo "Getting dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies complete"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run in development mode
dev:
	@echo "Running in development mode..."
	$(GOCMD) run $(MAIN_FILE)

# Install dependencies
install: deps
	@echo "Installing dependencies complete"

# Create default config if it doesn't exist
config:
	@if [ ! -f $(CONFIG_FILE) ]; then \
		echo "Creating default config file..."; \
		./$(BUILD_DIR)/$(BINARY_NAME) -config $(CONFIG_FILE); \
	fi

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME) .
	@echo "Docker build complete"

# Docker run
docker-run:
	@echo "Running Docker container..."
	docker run -p 9006:9006 -v $(PWD)/$(CONFIG_FILE):/app/$(CONFIG_FILE) $(BINARY_NAME)

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Clean, get deps, test, and build"
	@echo "  build        - Build the application"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  deps         - Get and tidy dependencies"
	@echo "  run          - Build and run the application"
	@echo "  dev          - Run in development mode"
	@echo "  install      - Install dependencies"
	@echo "  config       - Create default config file"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  help         - Show this help message"

# Create build directory
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Ensure build directory exists before building
build: $(BUILD_DIR)
