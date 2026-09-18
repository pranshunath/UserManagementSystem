# =============================================================
# Stage 1: Build binary using official Go Alpine compiler
# =============================================================
FROM golang:alpine AS builder

WORKDIR /src

# Install CA certificates and build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Cache Go modules layer
COPY go.mod go.sum ./
RUN go mod download

# Copy source tree and compile static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o /bin/api-gateway ./cmd/api/main.go

# =============================================================
# Stage 2: Minimal runtime with healthcheck utility
# =============================================================
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /bin/api-gateway /app/api-gateway

# Create non-root system user for security
RUN adduser -D -u 10001 appuser && chown -R appuser:appuser /app
USER appuser

# REST API Gateway listens on port 8080
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/api-gateway"]
