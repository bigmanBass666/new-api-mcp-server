# new-api-mcp-server Docker image
# Multi-stage build for minimal image size

# ---- Build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build with optimizations
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /build/new-api-mcp-server ./cmd/server

# ---- Run stage ----
FROM alpine:3.21

# CA certificates for HTTPS upstream requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/new-api-mcp-server .

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/new-api-mcp-server"]