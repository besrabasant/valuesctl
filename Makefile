# ---- Configurable vars -------------------------------------------------------

GO          ?= go
BUILD_DIR   ?= bin

# Binary/module
MODULE      := $(shell awk '/^module /{print $$2}' go.mod 2>/dev/null)
BIN         ?= $(notdir $(MODULE))
ifeq ($(BIN),)
BIN := valuesctl
endif

# Flags
VALIDATE        ?= 1     # set 0 to skip validation
APPLY_DEFAULTS  ?= 0     # set 1 to apply schema defaults on missing keys
BACKUP          ?= 1     # set 0 to disable .bak when patching

# Go build flags
CGO_ENABLED ?= 0
GOFLAGS     ?=
LDFLAGS     ?=
TAGS        ?=

# Default goal shows help
.DEFAULT_GOAL := help

# ---- Helpers -----------------------------------------------------------------

BIN_PATH := $(BUILD_DIR)/$(BIN)

define newline


endef

# ---- Targets -----------------------------------------------------------------

## Show this help
.PHONY: help
help:
	@echo "Usage: make <target>"
	@echo ""
	@awk 'BEGIN {FS":.*##"; printf "Targets:\n"} /^[a-zA-Z0-9_.-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

## Build the binary to $(BIN_PATH)
.PHONY: build
build: $(BIN_PATH)

$(BIN_PATH): $(shell find . -type f -name '*.go' -not -path './vendor/*')
	@mkdir -p $(BUILD_DIR)
	@echo ">> building $(BIN) ..."
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -trimpath -o $@ .

## Run the binary (pass ARGS="..."), e.g. make run ARGS="--help"
.PHONY: run
run: build
	@$(BIN_PATH) $(ARGS)

## Install to GOPATH/bin
.PHONY: install
install:
	@$(GO) install $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' .

## Format code
.PHONY: fmt
fmt:
	@$(GO) fmt ./...

## Vet code
.PHONY: vet
vet:
	@$(GO) vet ./...

## Tidy modules
.PHONY: tidy
tidy:
	@$(GO) mod tidy

## Run tests
.PHONY: test
test:
	@$(GO) test ./...

## Lint (optional; skips if golangci-lint not installed)
.PHONY: lint
lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed; skipping"

## Clean build artifacts
.PHONY: clean
clean:
	@rm -rf $(BUILD_DIR) *.bak
