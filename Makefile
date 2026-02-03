BINARY := bin/harvest
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w \
	-X github.com/dedene/harvest-cli/internal/cmd.Version=$(VERSION) \
	-X github.com/dedene/harvest-cli/internal/cmd.Commit=$(COMMIT) \
	-X github.com/dedene/harvest-cli/internal/cmd.Date=$(DATE)

# Enable CGO on macOS for Keychain support, disable elsewhere for portability
CGO_ENABLED ?= $(shell [ "$$(uname)" = "Darwin" ] && echo 1 || echo 0)

.PHONY: build test lint install clean fmt fmt-check ci

build:
	@mkdir -p $(dir $(BINARY))
	CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/harvest

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/harvest

clean:
	rm -f $(BINARY)

fmt:
	gofumpt -w .
	goimports -w .

fmt-check:
	@test -z "$$(gofumpt -l .)" || (echo "Run 'make fmt'" && exit 1)
	@test -z "$$(goimports -l .)" || (echo "Run 'make fmt'" && exit 1)

ci: fmt-check lint test
