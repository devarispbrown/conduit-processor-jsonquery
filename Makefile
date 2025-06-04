.PHONY: build test run clean lint fmt

# Build variables
BINARY_NAME=processor-jsonquery
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags="-s -w"

# Default target
all: build

# Build the processor plugin
build:
	@echo "Building json.query processor..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) .

# Run all tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -cover ./...

# Run the processor locally (for debugging)
run: build
	@echo "Running json.query processor..."
	./$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	$(GO) clean

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	goimports -w .