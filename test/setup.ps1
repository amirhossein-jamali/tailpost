# Setup script for testing environment (Windows PowerShell)

# Create a logs directory if it doesn't exist
New-Item -ItemType Directory -Force -Path logs | Out-Null

# Create a sample log file for testing
"This is a test log entry 1" | Out-File -FilePath logs\test.log
"This is a test log entry 2" | Out-File -FilePath logs\test.log -Append
"This is a test log entry 3" | Out-File -FilePath logs\test.log -Append

# Start the mock server in the background
$mockServerJob = Start-Job -ScriptBlock {
    Set-Location $using:PWD
    go run test\mock\mock_server.go
}

# Wait for the mock server to start
Start-Sleep -Seconds 2

Write-Host "Mock server started as job $($mockServerJob.Id)"
Write-Host "Test environment is ready"
Write-Host "Sample logs created in logs\test.log"
Write-Host "To run tests, use: go test .\test\ or go test .\..."
Write-Host ""
Write-Host "Press Ctrl+C to stop the mock server"

try {
    # Keep the script running until Ctrl+C
    while ($true) {
        Start-Sleep -Seconds 1
    }
}
finally {
    # Stop the mock server when the script is interrupted
    Stop-Job -Job $mockServerJob
    Remove-Job -Job $mockServerJob
    Write-Host "Mock server stopped"
} 