# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build the binary
RUN go build -o server ./cmd

# Stage 2: Final lightweight image
FROM alpine:latest

# Install ca-certificates for external API requests (e.g. Kakao API)
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder (all assets and templates are self-contained via go:embed)
COPY --from=builder /app/server .

# Create data directory for SQLite persistent storage
RUN mkdir -p /app/data

# Port the app listens on
EXPOSE 8080

# Set env variables
ENV APP_PORT=8080
ENV APP_HOST=0.0.0.0
ENV DATABASE_PATH=/app/data/restaurants.db

# Run the server
CMD ["./server"]
