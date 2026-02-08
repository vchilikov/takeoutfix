GO ?= go
BINARY ?= takeoutfix
GOFILES := $(shell find . -type f -name '*.go' -not -path './vendor/*')

.DEFAULT_GOAL := check

.PHONY: fmt fmt-check vet test test-race build lint goreleaser-check tools check check-all ci

fmt:
	gofmt -w $(GOFILES)

fmt-check:
	@files="$$(gofmt -l $(GOFILES))"; \
	if [ -n "$$files" ]; then \
		echo "Go files are not formatted:"; \
		echo "$$files"; \
		exit 1; \
	fi

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

build:
	$(GO) build -o $(BINARY) .

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint is not installed, skipping lint"; \
	fi

goreleaser-check:
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser check; \
	else \
		echo "goreleaser is not installed, skipping goreleaser-check"; \
	fi

tools:
	@if command -v brew >/dev/null 2>&1; then \
		brew install golangci-lint goreleaser; \
	else \
		echo "Homebrew is required to install tools on this machine."; \
		echo "Install Homebrew from https://brew.sh and rerun 'make tools'."; \
		exit 1; \
	fi

check: fmt-check vet test build

check-all: check test-race lint goreleaser-check

ci: check
