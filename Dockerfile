# ==============================================================================
# PLM Backend Dockerfile (Go)
# ==============================================================================
# Production-ready Go backend with security hardening
# ==============================================================================

FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o /build/plm-server ./cmd/server/main.go

# ==============================================================================
# Production Runtime
# ==============================================================================
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget && \
    rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1001 -S plm_group && \
    adduser -u 1001 -S plm_user -G plm_group

WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=plm_user:plm_group /build/plm-server /app/plm-server

# Switch to non-root user
USER plm_user

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

# Start server
CMD ["/app/plm-server"]
