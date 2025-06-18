# Stage 1: Build UI (Node.js)
FROM node:18-alpine AS ui-builder
WORKDIR /app-ui

# Copy package.json and package-lock.json (if available)
# Using '*' for package-lock.json to handle cases where it might not exist initially
COPY ui-panel/package.json ui-panel/package-lock.json* ./

# Install dependencies - use 'npm ci' if package-lock.json exists for deterministic builds
# For now, assuming it might not exist or for flexibility using 'npm install'
# Consider changing to 'RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi' for robustness
RUN npm install

# Copy the rest of the UI source code
COPY ui-panel/ ./

# Build the UI. VUE_APP_BASE_URL should be set for production builds.
# The .env.production file should set VUE_APP_BASE_URL=/ui/
# If not, explicitly setting it here is a fallback.
RUN npm run build
# Default build output is expected in /app-ui/dist

# Stage 2: Build the Go application
FROM golang:1.21-alpine AS builder

# Install build tools
RUN apk add --no-cache git build-base

WORKDIR /app

# Copy go module files
COPY go.mod go.sum ./
# Download Go modules
RUN go mod download

# Copy the Go source code
COPY main.go .
# If an 'internal' directory exists or is created for proto files, ensure it's copied:
# COPY internal/ ./internal/

# Build the Go application
RUN CGO_ENABLED=0 go build -o /server -ldflags="-w -s" ./main.go

# Stage 3: Create the final image
FROM teddysun/xray:latest

# Install ca-certificates for HTTPS communication by the Go app if needed
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the built Go application binary from the builder stage
COPY --from=builder /server /app/server

# Copy the built static UI files from the ui-builder stage
COPY --from=ui-builder /app-ui/dist /app/ui_static_files

# Expose V2Ray port and API port (values will be provided by Cloud Run via env vars)
EXPOSE $PORT
EXPOSE $API_PORT

# Set the entrypoint for the container
ENTRYPOINT ["/app/server"]
