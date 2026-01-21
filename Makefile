.PHONY: help build build-all test clean

# Variables
GIT_TAG := $(shell git describe --tags --always 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u +%Y-%m-%d)
LDFLAGS := -ldflags "-X github.com/khanglvm/tool-hub-mcp/internal/version.Version=$(GIT_TAG) -X github.com/khanglvm/tool-hub-mcp/internal/version.Commit=$(GIT_COMMIT) -X github.com/khanglvm/tool-hub-mcp/internal/version.Date=$(BUILD_DATE)"
BINARY_NAME=tool-hub-mcp
MAIN_PATH=./cmd/tool-hub-mcp

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build for current platform
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) $(MAIN_PATH)

build-all: ## Build for all platforms
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	@rm -rf bin/

install: ## Install to GOPATH/bin
	go install $(LDFLAGS) $(MAIN_PATH)
