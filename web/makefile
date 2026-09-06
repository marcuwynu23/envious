.PHONY: dev dev-watch start build release-build clean test deps version

BINARY_NAME ?= envious-server
DIST_DIR ?= dist
GOOS  ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

ifeq ($(GOOS),windows)
  OUTPUT := $(DIST_DIR)/$(BINARY_NAME).exe
else
  OUTPUT := $(DIST_DIR)/$(BINARY_NAME)
endif

# Override on the command line: make build VERSION=v1.0.0
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "unknown")
LDFLAGS = -ldflags "-X envious-web/internal/version.Version=$(VERSION) -X envious-web/internal/version.Commit=$(COMMIT) -X envious-web/internal/version.BuildDate=$(BUILD_DATE)"

dev:
	@go run ./cmd/server

dev-watch:
	@air -c .air.toml

start:
	@go run ./cmd/server

build:
	@echo "create dist directory if it doesn't exist"
	@mkdir -p $(DIST_DIR)
	@echo "build the binary ($(VERSION))"
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath $(LDFLAGS) -o $(OUTPUT) ./cmd/server
	@echo "copy the templates directory"
	@cp -r internal/api/templates $(DIST_DIR)/templates

# Alias kept for compatibility (same as build)
release-build: build

# Print the resolved version vars without building
version:
	@echo "VERSION=$(VERSION)"
	@echo "COMMIT=$(COMMIT)"
	@echo "BUILD_DATE=$(BUILD_DATE)"

test:
	@go test ./...

deps:
	@go mod download
	@go mod tidy

clean:
	@rm -rf $(DIST_DIR)
