.PHONY: all build build-server build-client proto run-server run-client test test-cover lint fmt vet clean help

BINARY_DIR := bin
SERVER_BINARY := $(BINARY_DIR)/server
CLIENT_BINARY := $(BINARY_DIR)/client
PROTO_DIR := proto
GO_FILES := $(shell find . -name '*.go' -type f)

# Default target
all: proto build

# Build all binaries
build: build-server build-client

# Build server
build-server:
	@echo "Building server..."
	@mkdir -p $(BINARY_DIR)
	go build -o $(SERVER_BINARY) ./cmd/server

# Build client
build-client:
	@echo "Building client..."
	@mkdir -p $(BINARY_DIR)
	go build -o $(CLIENT_BINARY) ./cmd/client

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/shell.proto

# Run server
run-server: build-server
	@echo "Starting server..."
	./$(SERVER_BINARY)

# Run client
run-client: build-client
	@echo "Starting client..."
	./$(CLIENT_BINARY)

# Run server with custom config
run-server-config: build-server
	@echo "Starting server with config..."
	./$(SERVER_BINARY) -config configs/server.yaml

# Run client with custom config
run-client-config: build-client
	@echo "Starting client with config..."
	./$(CLIENT_BINARY) -config configs/client.yaml

# Run all tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	gofmt -s -w $(GO_FILES)

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html
	rm -f $(PROTO_DIR)/*.pb.go

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Install protobuf tools
install-proto-tools:
	@echo "Installing protobuf tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Help
help:
	@echo "Available targets:"
	@echo "  all              - Generate proto and build all binaries"
	@echo "  build            - Build server and client"
	@echo "  build-server     - Build server only"
	@echo "  build-client     - Build client only"
	@echo "  proto            - Generate protobuf code"
	@echo "  run-server       - Run server"
	@echo "  run-client       - Run client"
	@echo "  test             - Run all tests"
	@echo "  test-cover       - Run tests with coverage"
	@echo "  lint             - Run linter"
	@echo "  fmt              - Format code"
	@echo "  vet              - Run go vet"
	@echo "  clean            - Clean build artifacts"
	@echo "  deps             - Install dependencies"
	@echo "  install-proto-tools - Install protobuf compiler plugins"
	@echo "  help             - Show this help"
