# TailPost Windows Installation Script
# Copyright © 2025 Amirhossein Jamali. All rights reserved.

[CmdletBinding()]
param (
    [string]$Version = "1.0.0",
    [string]$InstallDir = "$env:ProgramFiles\TailPost",
    [string]$ConfigDir = "$env:ProgramData\TailPost",
    [string]$LogDir = "$env:ProgramData\TailPost\logs",
    [switch]$NoService = $false
)

# Set error action preference to stop on error
$ErrorActionPreference = "Stop"

function Write-Banner {
    Write-Host "============================================"
    Write-Host "TailPost Installer (Windows)"
    Write-Host "Version: $Version"
    Write-Host "Copyright © 2025 Amirhossein Jamali"
    Write-Host "============================================"
}

function Test-Administrator {
    $currentUser = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    return $currentUser.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Install-TailPost {
    # Check for administrator privileges
    if (-not (Test-Administrator)) {
        Write-Error "This script must be run as Administrator"
        exit 1
    }

    # Create directories
    Write-Host "Creating directories..."
    $directories = @($InstallDir, $ConfigDir, $LogDir)
    foreach ($dir in $directories) {
        if (-not (Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }
    }

    # Download binary
    $binaryPath = Join-Path -Path $InstallDir -ChildPath "tailpost.exe"
    $downloadUrl = "https://github.com/amirhossein-jamali/tailpost/releases/download/v$Version/tailpost-windows-amd64.exe"
    
    Write-Host "Downloading TailPost binary..."
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $binaryPath
    }
    catch {
        Write-Error "Failed to download TailPost: $_"
        exit 1
    }

    # Create default configuration if it doesn't exist
    $configPath = Join-Path -Path $ConfigDir -ChildPath "config.yaml"
    if (-not (Test-Path $configPath)) {
        Write-Host "Creating default configuration..."
        $defaultConfig = @"
# TailPost Default Configuration
log_source_type: windows_event
windows_event_log_name: Application
windows_event_log_level: Information
server_url: http://log-server:8080/logs
batch_size: 100
flush_interval: 10s
log_level: info
"@
        $defaultConfig | Out-File -FilePath $configPath -Encoding utf8
    }

    # Create a Windows service if not disabled
    if (-not $NoService) {
        Write-Host "Creating Windows Service..."
        $serviceName = "TailPost"
        $serviceDisplayName = "TailPost Log Collection Agent"
        $serviceDescription = "Collects and forwards logs from various sources to a central server."
        
        # Check if service exists and remove it if it does
        if (Get-Service -Name $serviceName -ErrorAction SilentlyContinue) {
            Write-Host "Stopping and removing existing service..."
            Stop-Service -Name $serviceName -Force
            $service = New-Object -ComObject WbemScripting.SWbemLocator
            $service = $service.ConnectServer('.', 'root\cimv2')
            $service.Get("Win32_Service.Name='$serviceName'").Delete() | Out-Null
            Start-Sleep -Seconds 2
        }
        
        # Create the new service
        $binPath = "`"$binaryPath`" --config `"$configPath`""
        $service = New-Service -Name $serviceName -DisplayName $serviceDisplayName -Description $serviceDescription -BinaryPathName $binPath -StartupType Automatic
        
        # Start the service
        Write-Host "Starting TailPost service..."
        Start-Service -Name $serviceName
        
        # Check service status
        $status = (Get-Service -Name $serviceName).Status
        Write-Host "Service status: $status"
    }

    # Create environment variable for the binary path
    [Environment]::SetEnvironmentVariable("TAILPOST_PATH", $InstallDir, [System.EnvironmentVariableTarget]::Machine)

    Write-Host "TailPost $Version has been installed successfully!"
    Write-Host "Binary location: $binaryPath"
    Write-Host "Configuration file: $configPath"
    Write-Host "Log directory: $LogDir"
}

# Main execution
Write-Banner
Install-TailPost 