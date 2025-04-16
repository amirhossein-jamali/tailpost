#!/bin/bash
# Tailpost macOS Installer
# This script installs Tailpost as a LaunchDaemon on macOS

# Default values
INSTALL_PATH="/usr/local/tailpost"
CONFIG_PATH=""
SERVICE_NAME="com.tailpost.agent"
SERVICE_LABEL="Tailpost Log Collector"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --config)
      CONFIG_PATH="$2"
      shift 2
      ;;
    --install-path)
      INSTALL_PATH="$2"
      shift 2
      ;;
    --help)
      echo "Usage: install-macos.sh [options]"
      echo ""
      echo "Options:"
      echo "  --config PATH       Path to a custom config file"
      echo "  --install-path PATH Installation directory (default: /usr/local/tailpost)"
      echo "  --help              Display this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Use --help for usage information"
      exit 1
      ;;
  esac
done

# Check for root privileges
if [ "$(id -u)" -ne 0 ]; then
   echo "This script must be run as root" 
   exit 1
fi

# Create installation directory
echo "Creating installation directory: $INSTALL_PATH"
mkdir -p "$INSTALL_PATH"
mkdir -p "$INSTALL_PATH/logs"

# Download the latest release if not found
BINARY_PATH="$INSTALL_PATH/tailpost"
if [ ! -f "$BINARY_PATH" ]; then
    echo "Downloading Tailpost binary..."
    # Replace with actual download URL
    DOWNLOAD_URL="https://github.com/amirhossein-jamali/tailpost/releases/latest/download/tailpost-darwin-amd64"
    if ! curl -L -o "$BINARY_PATH" "$DOWNLOAD_URL"; then
        echo "Failed to download Tailpost. Please download manually and place in $BINARY_PATH"
        exit 1
    fi
    chmod +x "$BINARY_PATH"
fi

# Create default config if not provided
DEFAULT_CONFIG_PATH="$INSTALL_PATH/config.yaml"
if [ -z "$CONFIG_PATH" ] || [ ! -f "$CONFIG_PATH" ]; then
    echo "Creating default configuration for macOS logs..."
    cat > "$DEFAULT_CONFIG_PATH" << EOF
# Tailpost Configuration for macOS
log_source_type: macos_asl
macos_log_query: "process == \"kernel\" OR subsystem == \"com.apple.system\""
server_url: http://localhost:8081/logs
batch_size: 20
flush_interval: 5s
EOF
    CONFIG_PATH="$DEFAULT_CONFIG_PATH"
else
    # Copy provided config
    echo "Copying provided configuration from $CONFIG_PATH"
    cp "$CONFIG_PATH" "$DEFAULT_CONFIG_PATH"
    CONFIG_PATH="$DEFAULT_CONFIG_PATH"
fi

# Create LaunchDaemon plist
PLIST_PATH="/Library/LaunchDaemons/$SERVICE_NAME.plist"
echo "Creating LaunchDaemon plist at $PLIST_PATH"
cat > "$PLIST_PATH" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$SERVICE_NAME</string>
    <key>ProgramArguments</key>
    <array>
        <string>$BINARY_PATH</string>
        <string>--config</string>
        <string>$CONFIG_PATH</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardErrorPath</key>
    <string>$INSTALL_PATH/logs/error.log</string>
    <key>StandardOutPath</key>
    <string>$INSTALL_PATH/logs/service.log</string>
    <key>WorkingDirectory</key>
    <string>$INSTALL_PATH</string>
</dict>
</plist>
EOF

# Set appropriate permissions
chown root:wheel "$PLIST_PATH"
chmod 644 "$PLIST_PATH"

# Stop service if it's already running
if launchctl list | grep -q "$SERVICE_NAME"; then
    echo "Stopping existing Tailpost service..."
    launchctl unload "$PLIST_PATH" 2>/dev/null
fi

# Start the service
echo "Starting Tailpost service..."
launchctl load "$PLIST_PATH"

# Check status
if launchctl list | grep -q "$SERVICE_NAME"; then
    echo "Tailpost service is running."
else
    echo "Warning: Tailpost service failed to start. Check logs for details."
fi

echo "Installation complete!"
echo "Configuration file: $CONFIG_PATH"
echo "Log files: $INSTALL_PATH/logs"
echo "To view the service status: launchctl list | grep $SERVICE_NAME"
echo "To view logs: tail -f $INSTALL_PATH/logs/service.log" 