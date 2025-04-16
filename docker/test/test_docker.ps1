# PowerShell script to run TailPost tests in Docker environment

Write-Host "=== Building and running TailPost tests in Docker environment ===" -ForegroundColor Cyan

# Ensure log directory exists
if (-not (Test-Path logs)) {
    New-Item -Path logs -ItemType Directory
}

# Build and start the Docker test environment
Write-Host "Starting Docker test environment..." -ForegroundColor Yellow
docker-compose -f docker-compose.test.yml build
docker-compose -f docker-compose.test.yml up -d

# Wait for the mock server to be ready
Write-Host "Waiting for mock server to be ready..." -ForegroundColor Yellow
$mockServerReady = $false
for ($i = 1; $i -le 10; $i++) {
    $result = docker-compose -f docker-compose.test.yml exec -T mock-server wget -q --spider http://localhost:8081/health 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Mock server is ready!" -ForegroundColor Green
        $mockServerReady = $true
        break
    }
    
    if ($i -eq 10) {
        Write-Host "Error: Mock server failed to start" -ForegroundColor Red
        docker-compose -f docker-compose.test.yml logs
        docker-compose -f docker-compose.test.yml down
        exit 1
    }
    
    Write-Host "Waiting for mock server... ($i/10)" -ForegroundColor Yellow
    Start-Sleep -Seconds 2
}

# Run the test agent with the test configuration
Write-Host "Running agent with test configuration..." -ForegroundColor Yellow
Start-Job -ScriptBlock {
    docker-compose -f docker-compose.test.yml exec -T app sh -c "cp /app/config.test.yaml /app/config.yaml && ./tailpost -config /app/config.yaml"
} | Out-Null

# Wait for logs to be collected
Write-Host "Waiting for logs to be collected (30 seconds)..." -ForegroundColor Yellow
Start-Sleep -Seconds 30

# Check the mock server logs
Write-Host "Checking mock server logs..." -ForegroundColor Yellow
$logs = docker-compose -f docker-compose.test.yml logs mock-server
Write-Host $logs

# Verify logs were received
if ($logs -match "Received logs") {
    Write-Host "✅ Test passed! Logs were successfully sent to the mock server." -ForegroundColor Green
} else {
    Write-Host "❌ Test failed! No logs were received by the mock server." -ForegroundColor Red
    docker-compose -f docker-compose.test.yml logs
    docker-compose -f docker-compose.test.yml down
    exit 1
}

# Clean up
Write-Host "Cleaning up..." -ForegroundColor Yellow
Get-Job | Stop-Job
Get-Job | Remove-Job
docker-compose -f docker-compose.test.yml down

Write-Host "=== Test completed successfully ===" -ForegroundColor Cyan 