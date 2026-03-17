GO      ?= go
BINARY  := spotpilot
CMD     := ./cmd/spotpilot

VERSION   ?= dev
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%d)

LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)

.PHONY: fmt build run test lint ci clean

fmt:
	@if [ -n "$(gofmt -l .)" ]; then \
		echo "The following files are not formatted:"; \
		gofmt -l .; \
		exit 1; \
	fi

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

run:
	$(GO) run -ldflags "$(LDFLAGS)" $(CMD) $(ARGS)

test:
	$(GO) test ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found; running via go run (first run may take a while)"; \
		$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run; \
	fi

ci: fmt lint test build

clean:
	rm -f $(BINARY)
