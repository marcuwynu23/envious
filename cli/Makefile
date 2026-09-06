.PHONY: build release-build test test-coverage clean install run deps version

BINARY_NAME ?= envious
DIST_DIR ?= dist
GOOS := $(shell go env GOOS)
ifeq ($(GOOS),windows)
  OUTPUT := $(DIST_DIR)/$(BINARY_NAME).exe
else
  OUTPUT := $(DIST_DIR)/$(BINARY_NAME)
endif
# Override on the command line: make build VERSION=v1.0.0
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "unknown")
AUTHOR ?= $(shell git config user.name 2>/dev/null || echo "unknown")
LDFLAGS = -ldflags "-X envious-cli/cmd.Version=$(VERSION) -X envious-cli/cmd.Commit=$(COMMIT) -X envious-cli/cmd.BuildDate=$(BUILD_DATE) -X envious-cli/cmd.Author=$(AUTHOR)"

# Build the CLI with version info (override with VERSION=v1.0.0)
build:
	@echo "create dist directory if it doesn't exist"
	@mkdir -p $(DIST_DIR)
	@echo "build the binary ($(VERSION))"
	go build -trimpath $(LDFLAGS) -o $(OUTPUT) .

# Alias kept for compatibility (same as build)
release-build: build

# Print the resolved version vars without building
version:
	@echo "VERSION=$(VERSION)"
	@echo "COMMIT=$(COMMIT)"
	@echo "BUILD_DATE=$(BUILD_DATE)"
	@echo "AUTHOR=$(AUTHOR)"

# Run tests
test:
	@echo "run tests"
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "run tests with coverage"
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Install dependencies
deps:
	@echo "install dependencies"
	@go mod download
	@go mod tidy

# Run the CLI (after build)
run: build
	@echo "run the binary"
	./$(OUTPUT) $(ARGS)

# Clean build artifacts
clean:
	@echo "clean build artifacts"
	@rm -rf $(DIST_DIR)
	@rm -f coverage.out coverage.html

# Install CLI to $GOPATH/bin (optional)
install: release-build
	@echo "install the binary to $GOPATH/bin"
	@go install $(LDFLAGS) .
