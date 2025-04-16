#!/bin/bash
# TailPost Linux Installation Script
# Copyright © 2025 Amirhossein Jamali. All rights reserved.

set -e

# Default variables
VERSION="1.0.0"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/tailpost"
LOG_DIR="/var/log/tailpost"
DOWNLOAD_URL="https://github.com/amirhossein-jamali/tailpost/releases/download/v${VERSION}/tailpost-linux-amd64"
USE_SYSTEMD=true

# Banner
echo "============================================"
echo "TailPost Installer (Linux)"
echo "Version: ${VERSION}"
echo "Copyright © 2025 Amirhossein Jamali"
echo "============================================"

# Check for root privileges
if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root" >&2
    exit 1
fi

# Parse command line arguments
while [ $# -gt 0 ]; do
    case "$1" in
        --version=*)
            VERSION="${1#*=}"
            DOWNLOAD_URL="https://github.com/amirhossein-jamali/tailpost/releases/download/v${VERSION}/tailpost-linux-amd64"
            ;;
        --no-systemd)
            USE_SYSTEMD=false
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --version=VERSION    Specify the version to install"
            echo "  --no-systemd         Do not install systemd service"
            echo "  --help               Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
    shift
done

echo "Installing TailPost ${VERSION}..."

# Create directories
echo "Creating directories..."
mkdir -p "${CONFIG_DIR}" "${LOG_DIR}"
chmod 755 "${CONFIG_DIR}"
chmod 755 "${LOG_DIR}"

# Download binary
echo "Downloading TailPost binary..."
curl -L -o "${INSTALL_DIR}/tailpost" "${DOWNLOAD_URL}"
chmod +x "${INSTALL_DIR}/tailpost"

# Create default configuration
if [ ! -f "${CONFIG_DIR}/config.yaml" ]; then
    echo "Creating default configuration..."
    cat > "${CONFIG_DIR}/config.yaml" << EOF
# TailPost Default Configuration
log_source_type: file
log_path: /var/log/syslog
server_url: http://log-server:8080/logs
batch_size: 100
flush_interval: 10s
log_level: info
EOF
    chmod 644 "${CONFIG_DIR}/config.yaml"
fi

# Create systemd service
if [ "${USE_SYSTEMD}" = true ] && command -v systemctl >/dev/null 2>&1; then
    echo "Creating systemd service..."
    cat > /etc/systemd/system/tailpost.service << EOF
[Unit]
Description=TailPost Log Collection Agent
After=network.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/tailpost --config ${CONFIG_DIR}/config.yaml
Restart=always
RestartSec=10
User=root
Group=root
WorkingDirectory=/
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

    echo "Enabling and starting TailPost service..."
    systemctl daemon-reload
    systemctl enable tailpost
    systemctl start tailpost
fi

echo "TailPost ${VERSION} has been installed successfully!"
echo "Configuration file: ${CONFIG_DIR}/config.yaml"
echo "Binary location: ${INSTALL_DIR}/tailpost"

if [ "${USE_SYSTEMD}" = true ] && command -v systemctl >/dev/null 2>&1; then
    echo "Service status:"
    systemctl status tailpost --no-pager
fi

echo "Installation complete!" 