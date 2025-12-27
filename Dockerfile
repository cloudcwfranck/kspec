# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates

# Set working directory
WORKDIR /workspace

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the manager binary
# CGO_ENABLED=0 for static binary
# GOOS=linux for Linux target
# Stripping debug info and optimizing for size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s" \
    -o manager \
    ./cmd/manager/main.go

# Runtime stage - use distroless for minimal attack surface
FROM gcr.io/distroless/static:nonroot

WORKDIR /

# Copy the binary from builder
COPY --from=builder /workspace/manager /manager

# Use non-root user (distroless nonroot UID: 65532)
USER 65532:65532

# Expose ports for metrics and health probes
EXPOSE 8080 8081

# Set entrypoint
ENTRYPOINT ["/manager"]
