.PHONY: build test clean install run version-build

BINARY_NAME ?= envious
DIST_DIR ?= dist
GOOS := $(shell go env GOOS)
ifeq ($(GOOS),windows)
  OUTPUT := $(DIST_DIR)/$(BINARY_NAME).exe
else
  OUTPUT := $(DIST_DIR)/$(BINARY_NAME)
endif
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "unknown")
LDFLAGS = -ldflags "-X envious-cli/cmd.Version=$(VERSION) -X envious-cli/cmd.Commit=$(COMMIT) -X envious-cli/cmd.BuildDate=$(BUILD_DATE)"

# Build the CLI (development: version=dev)
build:
	mkdir -p $(DIST_DIR)
	go build -o $(OUTPUT) .

# Build with version info for release
release-build:
	mkdir -p $(DIST_DIR)
	go build $(LDFLAGS) -o $(OUTPUT) .

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run the CLI (after build)
run: build
	./$(OUTPUT) $(ARGS)

# Clean build artifacts
clean:
	rm -rf $(DIST_DIR)
	rm -f coverage.out coverage.html

# Install CLI to $GOPATH/bin (optional)
install: version-build
	go install .
