# Stage 1: Build the Go application
FROM golang:1.21-alpine AS builder

# Install build tools
RUN apk add --no-cache git build-base

WORKDIR /app

# Copy go module files
COPY go.mod go.sum ./
# Download Go modules
RUN go mod download

# Copy the Go source code
# Assuming main.go is in the root and any potential gRPC generated code
# would be in a subdirectory like 'internal'. If 'internal' or other dirs
# are created later for .pb.go files, they should be copied too.
# For now, only main.go is explicitly copied.
COPY main.go .
# If an 'internal' directory exists or is created for proto files, add:
# COPY internal/ ./internal/

# Build the Go application
# CGO_ENABLED=0 to build a statically-linked binary without C dependencies
# -ldflags="-w -s" to strip debug information and reduce binary size
RUN CGO_ENABLED=0 go build -o /server -ldflags="-w -s" ./main.go

# Stage 2: Create the final image
FROM teddysun/xray:latest

# Install ca-certificates for HTTPS communication by the Go app if needed
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the built Go application binary from the builder stage
COPY --from=builder /server /app/server

# Expose V2Ray port and API port (values will be provided by Cloud Run via env vars)
EXPOSE $PORT
EXPOSE $API_PORT

# Set the entrypoint for the container
ENTRYPOINT ["/app/server"]
