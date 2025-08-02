# ShipIt Client Daemon - Makefile

# Variables
BINARY_NAME=shipitd
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_UNIX=$(BINARY_NAME)_unix

# Build targets
.PHONY: all build clean test coverage lint format deps help install

all: clean build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/client

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME).linux ./cmd/client

build-macos:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME).darwin ./cmd/client
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME).darwin-arm64 ./cmd/client

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME).exe ./cmd/client

build-all: build-linux build-macos build-windows

install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin/"
	@sudo cp $(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installation complete!"

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe
	rm -f $(BINARY_NAME).linux
	rm -f $(BINARY_NAME).darwin
	rm -f $(BINARY_NAME).darwin-arm64

test:
	$(GOTEST) -v ./...

test-coverage:
	$(GOTEST) -v -coverprofile=coverage.txt -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.txt -o coverage.html

test-race:
	$(GOTEST) -race ./...

lint:
	golangci-lint run

format:
	gofmt -s -w .
	goimports -w .

deps:
	$(GOMOD) download
	$(GOMOD) tidy

install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Development helpers
dev:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/client
	./$(BINARY_NAME) --help

run:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/client
	./$(BINARY_NAME)

# Documentation
docs:
	@echo "Generating documentation..."
	@mkdir -p docs/_build

# Release helpers
release: clean build-all
	@echo "Building release binaries..."
	@mkdir -p dist
	@cp $(BINARY_NAME).linux dist/
	@cp $(BINARY_NAME).darwin dist/
	@cp $(BINARY_NAME).darwin-arm64 dist/
	@cp $(BINARY_NAME).exe dist/
	@echo "Release binaries created in dist/"

# Docker helpers
docker-build:
	docker build -t shipit-client:latest .

docker-run:
	docker run --rm -it shipit-client:latest

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary for current platform"
	@echo "  build-linux  - Build for Linux"
	@echo "  build-macos  - Build for macOS (Intel + Apple Silicon)"
	@echo "  build-windows- Build for Windows"
	@echo "  build-all    - Build for all platforms"
	@echo "  install      - Build and install to /usr/local/bin/"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  test-race    - Run tests with race detection"
	@echo "  lint         - Run linter"
	@echo "  format       - Format code"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  install-tools- Install development tools"
	@echo "  dev          - Build and show help"
	@echo "  run          - Build and run"
	@echo "  release      - Build release binaries"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  help         - Show this help" 