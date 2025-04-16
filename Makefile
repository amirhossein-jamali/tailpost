# TailPost Makefile
# Copyright © 2025 Amirhossein Jamali. All rights reserved.

.PHONY: build test clean docker-build docker-test docker-dev lint fmt help

# Build variables
BINARY_NAME=tailpost
LDFLAGS=-ldflags "-X github.com/amirhossein-jamali/tailpost/pkg/version.Version=1.0.0 -X github.com/amirhossein-jamali/tailpost/pkg/version.BuildTime=`date +%FT%T%z`"

# Default target
.DEFAULT_GOAL := help

# Help target
help:
	@echo "TailPost Makefile"
	@echo "Usage:"
	@echo "  make build        Build the TailPost binary"
	@echo "  make test         Run all tests"
	@echo "  make clean        Remove build artifacts"
	@echo "  make docker-build Build Docker image"
	@echo "  make docker-test  Run tests in Docker"
	@echo "  make docker-dev   Start development environment in Docker"
	@echo "  make lint         Run linters"
	@echo "  make fmt          Format code"
	@echo "  make release      Prepare a release"

# Build the binary
build:
	@echo "Building TailPost..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/agent.go

# Run all tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run specific tests
test-unit:
	@echo "Running unit tests..."
	go test -v -short ./...

test-integration:
	@echo "Running integration tests..."
	go test -v -run Integration ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	go clean

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t tailpost:latest .

docker-test:
	@echo "Running tests in Docker..."
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit

docker-dev:
	@echo "Starting development environment in Docker..."
	docker-compose up -d

# Code quality
lint:
	@echo "Running linters..."
	golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

# Installation
install: build
	@echo "Installing TailPost..."
	cp $(BINARY_NAME) /usr/local/bin/

# Release
release: clean build test
	@echo "Preparing release..."
	mkdir -p release
	cp $(BINARY_NAME) release/
	cp README.md LICENSE release/
	cp -r examples release/
	cp -r install release/
	tar -czf $(BINARY_NAME)-$(shell go run ./cmd/version.go).tar.gz release
	@echo "Release package created: $(BINARY_NAME)-$(shell go run ./cmd/version.go).tar.gz"

# Cross-platform builds
build-all: clean
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/agent.go
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/agent.go
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/agent.go
	@echo "Multi-platform builds complete" 