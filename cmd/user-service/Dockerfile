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
    -o /bin/user-service ./cmd/user-service/main.go

# =============================================================
# Stage 2: Minimal distroless/alpine runtime
# =============================================================
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /bin/user-service /app/user-service

# Create non-root system user for security
RUN adduser -D -u 10001 appuser && chown -R appuser:appuser /app
USER appuser

# gRPC microservice listens on internal container port 50051
EXPOSE 50051

ENTRYPOINT ["/app/user-service"]
