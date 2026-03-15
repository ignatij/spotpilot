GO      ?= go
BINARY  := spotpilot
CMD     := ./cmd/spotpilot

VERSION   ?= dev
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%d)

LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)

.PHONY: build run test lint clean

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

run:
	$(GO) run -ldflags "$(LDFLAGS)" $(CMD) $(ARGS)

test:
	$(GO) test ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY)
