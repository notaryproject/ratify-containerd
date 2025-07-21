FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/

# Build the monitor binary
RUN go build -o monitor ./cmd/monitor

# Use a minimal image for runtime
FROM alpine:3.19

WORKDIR /app

# Copy the monitor binary from builder
COPY --from=builder /app/monitor /app/monitor

# Create the shared-data directory (optional, for mount)
RUN mkdir -p /shared-data

# Set the entrypoint to the monitor binary
ENTRYPOINT ["/app/monitor"]