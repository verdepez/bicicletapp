# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=1 is required for go-sqlite3
RUN CGO_ENABLED=1 GOOS=linux go build -o main ./cmd/server

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies (sqlite libs)
RUN apk add --no-cache sqlite-libs ca-certificates

# Copy binary from builder
COPY --from=builder /app/main .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

# Create data directory for SQLite
RUN mkdir -p /data

# Expose port
EXPOSE 8080

# Environment variables
ENV PORT=8080
ENV DATABASE_PATH=/data/bicicletapp.db
ENV GIN_MODE=release

# Run the binary
CMD ["./main"]
