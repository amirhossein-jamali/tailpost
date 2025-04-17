FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy module files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o tailpost ./cmd/agent.go

# Use a minimal alpine image for the final container
FROM alpine:3.19

WORKDIR /

# Add CA certificates for HTTPS connections
RUN apk --no-cache add ca-certificates tzdata

# Create necessary directories
RUN mkdir -p /etc/tailpost /var/log/tailpost

# Copy the binary from the builder stage
COPY --from=builder /app/tailpost /usr/local/bin/tailpost

# Set executable permissions
RUN chmod +x /usr/local/bin/tailpost

# Default config file
ARG CONFIG_FILE=./docker/config.docker.yaml
COPY ${CONFIG_FILE} /etc/tailpost/config.yaml

# Set environment variables
ENV TAILPOST_CONFIG_PATH=/etc/tailpost/config.yaml
ENV TAILPOST_LOG_LEVEL=info

# Expose the health check port
EXPOSE 8080

# Run as non-root user
RUN addgroup -S tailpost && adduser -S -G tailpost tailpost
USER tailpost

# Command to run the executable
ENTRYPOINT ["/usr/local/bin/tailpost"]
CMD ["--config", "/etc/tailpost/config.yaml"]

# Metadata
LABEL maintainer="Amirhossein Jamali <jamaali.amirhossein@gmail.com>"
LABEL version="1.0.0"
LABEL description="TailPost - Cross-platform log collection agent"
LABEL org.opencontainers.image.source="https://github.com/amirhossein-jamali/tailpost"
LABEL org.opencontainers.image.licenses="Proprietary"
LABEL org.opencontainers.image.created="2025-04-16T12:00:00Z" 