PROJECT_NAME := "afittestide-asimi-cli"

# List all available recipes
default:
    @just --list

# Install Go dependencies
install:
    go mod download
    go mod vendor

# Build the binary
build:
    go build -o asimi .

# Run with debug logging
run:
    go run . --debug

# Run all tests
test:
    go test -v ./...

# Run tests with coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# Run linting
lint:
    golangci-lint run

# Format code
fmt:
    go fmt ./...
    goimports -w .

# Clean build artifacts
clean:
    rm -f asimi
    rm -f coverage.out coverage.html
    rm -f asimi.log
    rm -rf profiles/
    rm -rf test_tmp/

# Install development tools
bootstrap:
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    go install golang.org/x/tools/cmd/goimports@latest

# Measure shell performance
measure:
    ./profile_startup.sh

# Build the sandbox container
build-sandbox:
    podman machine init --disk-size 30 || true
    podman machine start || true
    podman build -t localhost/asimi-sandbox-{{PROJECT_NAME}}:latest -f .agents/sandbox/Dockerfile .

# Clean up the sandbox container
clean-sandbox:
    podman rmi localhost/asimi-sandbox-{{PROJECT_NAME}}:latest
