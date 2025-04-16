# TailPost Usage Guide

This document provides detailed instructions for installing, configuring, and using TailPost in various environments.

## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [Common Use Cases](#common-use-cases)
- [Security Best Practices](#security-best-practices)
- [Troubleshooting](#troubleshooting)

## Installation

### Linux

```bash
# Download the installation script
curl -O https://raw.githubusercontent.com/amirhossein-jamali/tailpost/main/install/install-linux.sh

# Make it executable
chmod +x install-linux.sh

# Run the installer
sudo ./install-linux.sh
```

The installer will:
1. Download the appropriate binary for your system
2. Install it to `/usr/local/bin/tailpost`
3. Create a configuration directory at `/etc/tailpost`
4. Set up a systemd service (if systemd is available)

### macOS

```bash
# Download the installation script
curl -O https://raw.githubusercontent.com/amirhossein-jamali/tailpost/main/install/install-macos.sh

# Make it executable
chmod +x install-macos.sh

# Run the installer
sudo ./install-macos.sh
```

The installer will:
1. Download the appropriate binary for your system
2. Install it to `/usr/local/bin/tailpost`
3. Create a configuration directory at `/etc/tailpost`
4. Set up a launchd service

### Windows

```powershell
# Download the installation script
Invoke-WebRequest -Uri https://raw.githubusercontent.com/amirhossein-jamali/tailpost/main/install/install-windows.ps1 -OutFile install-windows.ps1

# Run the installer as Administrator
.\install-windows.ps1
```

The installer will:
1. Download the appropriate binary for your system
2. Install it to `C:\Program Files\TailPost`
3. Create a configuration directory at `C:\ProgramData\TailPost`
4. Set up a Windows service

### Docker

```bash
# Pull the image
docker pull amirhossein-jamali/tailpost:latest

# Run with a custom configuration
docker run -v /path/to/config.yaml:/etc/tailpost/config.yaml amirhossein-jamali/tailpost:latest
```

### Kubernetes

```bash
# Apply the CRD
kubectl apply -f https://raw.githubusercontent.com/amirhossein-jamali/tailpost/main/deploy/kubernetes/tailpost_crd.yaml

# Deploy the operator
kubectl apply -f https://raw.githubusercontent.com/amirhossein-jamali/tailpost/main/deploy/kubernetes/operator_deployment.yaml

# Create a TailPost instance
kubectl apply -f https://raw.githubusercontent.com/amirhossein-jamali/tailpost/main/deploy/kubernetes/tailpost_example_cr.yaml
```

## Configuration

TailPost is configured using a YAML file. The default location varies by platform:

- Linux: `/etc/tailpost/config.yaml`
- macOS: `/etc/tailpost/config.yaml`
- Windows: `C:\ProgramData\TailPost\config.yaml`

### Basic Configuration

Here's a minimal configuration that collects logs from a file:

```yaml
log_source_type: file
log_path: /var/log/syslog
server_url: http://log-server:8080/logs
batch_size: 100
flush_interval: 10s
```

### Multiple Log Sources

You can configure multiple log sources:

```yaml
log_sources:
  - type: file
    path: /var/log/syslog
  - type: file
    path: /var/log/auth.log
  - type: windows_event
    windows_event_log_name: Application
    windows_event_log_level: Information
```

### Advanced Settings

```yaml
# General settings
batch_size: 100
flush_interval: 10s
max_retry_count: 5
retry_backoff: 30s
server_url: http://log-server:8080/logs

# Log sources
log_sources:
  - type: file
    path: /var/log/syslog
    include_pattern: ".*ERROR.*"
    exclude_pattern: ".*DEBUG.*"
    follow: true
    from_beginning: false

# Security settings
security:
  tls:
    enabled: true
    ca_cert: /path/to/ca.crt
    client_cert: /path/to/client.crt
    client_key: /path/to/client.key
    insecure_skip_verify: false
  auth:
    type: bearer
    token: ${TAILPOST_AUTH_TOKEN}
  encryption:
    enabled: true
    type: aes-gcm
    key_file: /path/to/encryption.key

# Telemetry
telemetry:
  enabled: true
  service_name: "tailpost-agent"
  service_version: "1.0.0"
  exporter_type: "http"
  exporter_endpoint: "http://otel-collector:4318"
  sampling_rate: 1.0
```

## Common Use Cases

### Collecting System Logs

```yaml
log_sources:
  - type: file
    path: /var/log/syslog
  - type: file
    path: /var/log/auth.log
  - type: file
    path: /var/log/kern.log
```

### Monitoring Application Logs

```yaml
log_sources:
  - type: file
    path: /var/log/app/myapp.log
    include_pattern: "\\[(ERROR|WARN)\\]"
```

### Kubernetes Container Logs

```yaml
log_sources:
  - type: container
    namespace: default
    pod_name: my-application
    container_name: app
```

### Windows Event Logs

```yaml
log_sources:
  - type: windows_event
    windows_event_log_name: Application
    windows_event_log_level: Warning
  - type: windows_event
    windows_event_log_name: System
    windows_event_log_level: Error
```

## Security Best Practices

1. **Use TLS**: Always enable TLS to secure communications
2. **Implement Authentication**: Set up proper authentication
3. **Manage Encryption Keys**: Rotate encryption keys regularly
4. **Limit Access**: Run TailPost with minimal privileges
5. **Validate Configurations**: Check configurations for security issues

## Troubleshooting

### Common Issues

#### Agent Can't Connect to Server

1. Check network connectivity
2. Verify server URL is correct
3. Ensure TLS certificates are valid
4. Check authentication credentials

#### Missing Logs

1. Verify file paths exist and are readable
2. Check include/exclude patterns
3. Ensure log rotation isn't affecting collection
4. Check for permission issues

#### High Resource Usage

1. Reduce batch size
2. Increase flush interval
3. Limit the number of log sources
4. Add more specific include/exclude patterns

### Debugging

Enable debug logging by setting the log level:

```yaml
log_level: debug
```

View logs:
- Linux/macOS: `journalctl -u tailpost`
- Windows: Event Viewer > Application and Services Logs > TailPost

Copyright © 2025 Amirhossein Jamali. All rights reserved. 