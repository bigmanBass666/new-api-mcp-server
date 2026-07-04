.PHONY: build test lint run clean

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
