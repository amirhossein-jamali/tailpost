# TailPost

A high-performance, cross-platform log collection agent designed for modern observability stacks. TailPost efficiently collects logs from diverse sources and securely forwards them to central processing systems.

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org/doc/go1.23)
[![License: Custom](https://img.shields.io/badge/License-Custom-blue.svg)](./LICENSE)

## Overview

TailPost addresses the challenges of log collection in hybrid and multi-platform environments. Built with a focus on reliability, security, and observability, it serves as a robust agent for enterprise logging solutions.

## Key Features

- **Cross-Platform Collection**: Native support for Linux, macOS, and Windows
- **Multiple Source Types**:
  - File-based logs with rotation support
  - Windows Event Logs
  - macOS ASL logs
  - Kubernetes container and pod logs
- **Advanced Batching**: Configurable batch sizes and flush intervals with backpressure handling
- **Enterprise Security**:
  - TLS with certificate validation
  - Multiple authentication methods
  - End-to-end encryption options
- **Native Kubernetes Integration**:
  - Kubernetes Operator for deployment
  - Pod and container log collection
  - StatefulSet management
- **Comprehensive Observability**:
  - OpenTelemetry integration
  - Prometheus metrics
  - Health check API

## Architecture

TailPost follows a modular architecture with these primary components:

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Log Readers │────▶│ Log Pipeline │────▶│ Log Senders  │
└──────────────┘     └──────────────┘     └──────────────┘
       │                     │                    │
       ▼                     ▼                    ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Security   │     │  Telemetry   │     │ HTTP Client  │
└──────────────┘     └──────────────┘     └──────────────┘
```

### Components

- **Log Readers**: Specialized components that collect logs from various sources (files, Windows events, etc.)
- **Log Pipeline**: Processes and prepares logs for transmission
- **Log Senders**: Transport logs to the destination with reliability guarantees
- **Security Layer**: Handles TLS, authentication, and encryption
- **Telemetry**: Provides insights into agent performance and health
- **HTTP Client**: Manages connections to log receivers

## Installation

Platform-specific installation scripts are available in the [install](./install) directory:

- Linux: [install-linux.sh](./install/install-linux.sh)
- macOS: [install-macos.sh](./install/install-macos.sh)
- Windows: [install-windows.ps1](./install/install-windows.ps1)

## Configuration

TailPost is configured via YAML. See the [examples](./examples) directory for sample configurations.

### Basic Configuration

```yaml
log_source_type: file
log_path: /var/log/syslog
server_url: http://log-server:8080/logs
batch_size: 100
flush_interval: 10s
```

### Security Configuration

```yaml
security:
  tls:
    enabled: true
    ca_cert: /path/to/ca.crt
    client_cert: /path/to/client.crt
    client_key: /path/to/client.key
    insecure_skip_verify: false
  auth:
    type: bearer
    token: ${ENV_TOKEN_VAR}
  encryption:
    enabled: true
    type: aes-gcm
    key_file: /path/to/encryption.key
```

### Telemetry Configuration

```yaml
telemetry:
  enabled: true
  service_name: "tailpost-agent"
  service_version: "1.0.0"
  exporter_type: "http"
  exporter_endpoint: "http://otel-collector:4318"
  sampling_rate: 1.0
  context_propagation: true
  attributes:
    environment: "production"
    region: "us-west"
```

## Kubernetes Integration

TailPost can be deployed in Kubernetes environments using the provided Operator:

```yaml
apiVersion: logging.tailpost.io/v1
kind: TailpostAgent
metadata:
  name: tailpost-agent
spec:
  replicas: 3
  image: tailpost/agent:latest
  config:
    log_sources:
      - type: container
namespace: default
        pod_label_selector: app=nginx
```

## Telemetry Integration

TailPost provides observability through a built-in telemetry system:

- **Distributed Tracing**: Trace log flow from collection to delivery using OpenTelemetry
- **Performance Metrics**: Monitor batch sizes, processing times, and delivery latency
- **Error Tracking**: Capture errors during log collection and delivery

### Supported Telemetry Exports:

- **OpenTelemetry**: Native support with both HTTP and gRPC protocols
- **Prometheus**: Built-in metrics exporters for Prometheus scraping

Through OpenTelemetry's ecosystem, you can further connect to:
- Jaeger
- Zipkin
- OpenTelemetry Collector
- Various cloud-based observability platforms

### Sample Telemetry Configuration:

```yaml
telemetry:
  enabled: true
  service_name: "tailpost-agent"
  service_version: "1.0.0"
  exporter_type: "http"     # "http", "grpc", or "none"
  exporter_endpoint: "http://otel-collector:4318"
  sampling_rate: 1.0
  attributes:
    environment: "production"
    region: "us-west"
```

## Development

### Build Requirements

- Go 1.23+
- Make

### Build from Source

```bash
# Build the binary
make build

# Run tests
make test

# Run with specific config
./tailpost --config=config.yaml
```

### Docker Development Environment

A complete Docker-based development environment is available:

```bash
# Start the development environment
make docker-dev

# Run tests in Docker
make docker-test
```

## Testing

TailPost includes a comprehensive test suite:

- Unit tests for all components
- Integration tests for end-to-end validation
- Performance benchmarks

### Docker Test Environment

```bash
# Linux/macOS
./test_docker.sh

# Windows
.\test_docker.ps1
```

The test environment includes:

1. TailPost Agent container
2. Mock server for receiving logs
3. Sidecar container generating test logs

## Performance

TailPost is designed for high-performance log collection:

- Efficient use of goroutines for concurrent log collection
- Batching to minimize network overhead
- Low memory footprint with configurable resource limits

## License

This project is licensed under a custom proprietary license - see the [LICENSE](./LICENSE) file for details.

## Copyright

Copyright © 2025 Amirhossein Jamali. All rights reserved.

## Contributing

Contributions are welcome but require signing a Contributor License Agreement. Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.