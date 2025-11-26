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

# Create data directory
RUN mkdir -p /data && chown easyoss:easyoss /data

# Copy binary from builder
COPY --from=builder /easyoss /usr/local/bin/easyoss

# Switch to non-root user
USER easyoss

# Set working directory
WORKDIR /data

# Expose port
EXPOSE 9000

# Set default data path
ENV DATA_PATH=/data

# Run the application
ENTRYPOINT ["easyoss"]
CMD ["-port", "9000", "-data", "/data"]
