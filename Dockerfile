# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /server ./cmd/server

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
# - ca-certificates: TLS verification
# - tzdata: timezone support
# - postgresql-client: pg_isready for health checks, psql for migrations
# - golang-migrate: database migration tool
RUN apk add --no-cache ca-certificates tzdata postgresql-client && \
    wget -q https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz && \
    tar -xzf migrate.linux-amd64.tar.gz && \
    mv migrate /usr/local/bin/migrate && \
    rm migrate.linux-amd64.tar.gz

# Create non-root user
RUN adduser -D -g '' appuser

# Copy binary from builder
COPY --from=builder /server /app/server

# Copy migrations and entrypoint
COPY migrations /app/migrations
COPY scripts/entrypoint.sh /app/entrypoint.sh

# Make entrypoint executable
RUN chmod +x /app/entrypoint.sh

# Use non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run via entrypoint (handles migrations and startup)
ENTRYPOINT ["/app/entrypoint.sh"]
