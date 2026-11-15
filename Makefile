BINARY_NAME=atlas
BUILD_DIR=bin
SANDBOX_DIR=dev/fake_root

.PHONY: all build run dev prod seed test clean clean-sandbox clean-config tidy fmt vet help

all: build

## build: Compile binary to ./bin/atlas
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .

## run: Run application in development sandbox mode (uses ./dev/fake_root/var/www)
run:
	@ATLAS_ROOT_DIR=$(SANDBOX_DIR) go run .

## dev: Build and run binary in development sandbox mode
dev: build
	@ATLAS_ROOT_DIR=$(SANDBOX_DIR) ./$(BUILD_DIR)/$(BINARY_NAME)

## prod: Run application pointing to real host filesystem (/)
prod: build
	@ATLAS_ENV=production ./$(BUILD_DIR)/$(BINARY_NAME)

## test: Execute automated unit tests
test:
	@go test -v ./...

## tidy: Tidy and verify go.mod dependencies
tidy:
	@go mod tidy

## fmt: Format Go source code
fmt:
	@go fmt ./...

## vet: Run static analysis with go vet
vet:
	@go vet ./...

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(BINARY_NAME)

## clean-sandbox: Reset the fake development root directory (./dev/fake_root)
clean-sandbox:
	@echo "Removing sandbox directory $(SANDBOX_DIR)..."
	@rm -rf dev

## clean-config: Remove the sandbox configuration store (domains.json)
clean-config:
	@echo "Removing sandbox configuration store..."
	@rm -rf $(SANDBOX_DIR)/.config/atlas

## help: Show available Makefile commands
help:
	@echo "Available commands:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
