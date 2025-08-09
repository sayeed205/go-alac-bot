# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o go-alac-bot .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/go-alac-bot .

# Create downloads directory
RUN mkdir -p downloads

# Set proper permissions
RUN chmod +x go-alac-bot

# Create non-root user
RUN addgroup -g 1001 botuser && \
    adduser -D -s /bin/sh -u 1001 -G botuser botuser

# Change ownership of app directory
RUN chown -R botuser:botuser /app

# Switch to non-root user
USER botuser

# Expose port (optional, for health checks)
EXPOSE 8080

# Health check (optional)
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep go-alac-bot || exit 1

# Run the application
CMD ["./go-alac-bot"]
