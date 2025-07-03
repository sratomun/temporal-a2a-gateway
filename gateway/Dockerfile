# A2A Gateway Go Service Dockerfile
FROM golang:1.24-alpine AS builder

# Install necessary system dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Copy source code (needed for go mod tidy to analyze imports)
COPY . .

# Initialize go.sum and download dependencies
RUN go mod tidy && go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o a2a-gateway .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/a2a-gateway .

# Copy configuration files
COPY --from=builder /app/config ./config

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:3000/health || exit 1

# Expose port
EXPOSE 3000

# Run the binary
CMD ["./a2a-gateway"]