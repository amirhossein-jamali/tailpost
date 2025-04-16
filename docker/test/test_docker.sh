#!/bin/bash
set -e

echo "=== Building and running TailPost tests in Docker environment ==="

# Ensure log directory exists
mkdir -p logs

# Build and start the Docker test environment
echo "Starting Docker test environment..."
docker-compose -f docker-compose.test.yml build
docker-compose -f docker-compose.test.yml up -d

# Wait for the mock server to be ready
echo "Waiting for mock server to be ready..."
for i in {1..10}; do
  if docker-compose -f docker-compose.test.yml exec -T mock-server wget -q --spider http://localhost:8081/health; then
    echo "Mock server is ready!"
    break
  fi
  
  if [ $i -eq 10 ]; then
    echo "Error: Mock server failed to start"
    docker-compose -f docker-compose.test.yml logs
    docker-compose -f docker-compose.test.yml down
    exit 1
  fi
  
  echo "Waiting for mock server... ($i/10)"
  sleep 2
done

# Run the test agent with the test configuration
echo "Running agent with test configuration..."
docker-compose -f docker-compose.test.yml exec -T app sh -c "cp /app/config.test.yaml /app/config.yaml && ./tailpost -config /app/config.yaml" &
AGENT_PID=$!

# Wait for logs to be collected
echo "Waiting for logs to be collected (30 seconds)..."
sleep 30

# Check the mock server logs
echo "Checking mock server logs..."
LOGS=$(docker-compose -f docker-compose.test.yml logs mock-server)
echo "$LOGS"

# Verify logs were received
if echo "$LOGS" | grep -q "Received logs"; then
  echo "✅ Test passed! Logs were successfully sent to the mock server."
else
  echo "❌ Test failed! No logs were received by the mock server."
  docker-compose -f docker-compose.test.yml logs
  docker-compose -f docker-compose.test.yml down
  exit 1
fi

# Clean up
echo "Cleaning up..."
kill $AGENT_PID 2>/dev/null || true
docker-compose -f docker-compose.test.yml down

echo "=== Test completed successfully ===" 