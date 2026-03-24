.PHONY: dev dev-watch start build clean test deps

BINARY_NAME ?= envious-server
DIST_DIR ?= dist
GOOS  ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

ifeq ($(GOOS),windows)
  OUTPUT := $(DIST_DIR)/$(BINARY_NAME).exe
else
  OUTPUT := $(DIST_DIR)/$(BINARY_NAME)
endif

dev:
	@go run ./cmd/server

dev-watch:
	@air -c .air.toml

start:
	@go run ./cmd/server

build:
	@echo "create dist directory if it doesn't exist"
	@mkdir -p $(DIST_DIR)
	@echo "build the binary"
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(OUTPUT) ./cmd/server
	@echo "copy the templates directory"
	@cp -r internal/api/templates $(DIST_DIR)/templates

test:
	@go test ./...

deps:
	@go mod download
	@go mod tidy

clean:
	@rm -rf $(DIST_DIR)
