# Tailpost Installation Scripts

This directory contains platform-specific installation scripts for Tailpost, a log collection agent that reads logs from files or containers and sends them to a specified server for processing.

## Available Installers

### Linux (systemd-based distributions)

The Linux installer sets up Tailpost as a systemd service.

```bash
sudo ./install-linux.sh [options]
```

Options:
- `--config PATH`: Path to a custom config file
- `--install-path PATH`: Installation directory (default: /opt/tailpost)
- `--help`: Display help message

### macOS

The macOS installer sets up Tailpost as a LaunchDaemon.

```bash
sudo ./install-macos.sh [options]
```

Options:
- `--config PATH`: Path to a custom config file
- `--install-path PATH`: Installation directory (default: /usr/local/tailpost)
- `--help`: Display help message

### Windows

The Windows installer sets up Tailpost as a Windows service using NSSM (Non-Sucking Service Manager).

```powershell
# Run as Administrator
.\install-windows.ps1 [options]
```

Parameters:
- `-ConfigPath PATH`: Path to a custom config file
- `-InstallPath PATH`: Installation directory (default: C:\Program Files\Tailpost)
- `-ServiceName NAME`: Name of the service (default: Tailpost)
- `-ServiceDisplayName NAME`: Display name of the service (default: Tailpost Log Collector)
- `-ServiceDescription DESC`: Description of the service (default: Collects logs from Windows events and files)

## Pre-built Binary Downloads

The scripts will download the appropriate binary for your platform automatically. If the download fails, you can manually download the binaries from the releases page and place them in the installation directory:

- Linux: https://github.com/amirhossein-jamali/tailpost/releases/latest/download/tailpost-linux-amd64
- macOS: https://github.com/amirhossein-jamali/tailpost/releases/latest/download/tailpost-darwin-amd64
- Windows: https://github.com/amirhossein-jamali/tailpost/releases/latest/download/tailpost-windows-amd64.exe

## Requirements

### Linux
- systemd-based Linux distribution
- root permissions
- curl (for downloading binaries)

### macOS
- macOS 10.12 Sierra or later
- root permissions
- curl (for downloading binaries)

### Windows
- Windows 10/11 or Windows Server 2016 or later
- Administrator permissions
- NSSM (Non-Sucking Service Manager) installed and in PATH
- PowerShell 5.0 or later

## Post-Installation

After installation, Tailpost will start automatically as a service. You can check the service status and view logs using the platform-specific commands shown at the end of each installer's output.

### Default Configuration

If no custom configuration is provided, the installers will create a default configuration appropriate for the platform:

- **Linux**: Monitors `/var/log/syslog`
- **macOS**: Monitors macOS system logs with a filter for kernel and system processes
- **Windows**: Monitors the Windows Application Event Log 