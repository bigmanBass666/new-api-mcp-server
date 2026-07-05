.PHONY: build test lint run clean docker-build docker-up docker-down

BINARY=new-api-mcp-server
# Windows needs .exe extension for MCP servers to spawn correctly
ifeq ($(OS),Windows_NT)
BINARY:=new-api-mcp-server.exe
endif
VERSION?=dev
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/server

test:
	go test ./... -v -race -count=1

lint:
	golangci-lint run ./...

run:
	go run ./cmd/server

clean:
	rm -rf bin/

# Docker targets
docker-build:
	docker build -t new-api-mcp-server:$(VERSION) .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

# E2E test targets
test-e2e:
	@if [ -f scripts/test-e2e.sh ]; then bash scripts/test-e2e.sh; else echo "scripts/test-e2e.sh not found — run 'make build' first"; fi

test-int:
	@echo "Running integration tests (requires MCP + New API)..."
	go test -tags=integration -v -count=1 ./internal/hightools/