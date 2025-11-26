# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /easyoss ./cmd/easyoss

# Final stage
FROM alpine:3.19

# Install ca-certificates for HTTPS support
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN adduser -D -g '' easyoss

# Create data and meta directories and config directory
RUN mkdir -p /data /meta /etc/easyoss && chown easyoss:easyoss /data /meta /etc/easyoss

# Copy binary from builder
COPY --from=builder /easyoss /usr/local/bin/easyoss

# Copy config file
COPY --from=builder /app/config/config.yaml /etc/easyoss/config.yaml

# Switch to non-root user
USER easyoss

# Set working directory
WORKDIR /data

# Expose ports (S3 API and Web UI)
EXPOSE 9000 9001

# Run the application (uses config file by default)
ENTRYPOINT ["easyoss"]
CMD []
