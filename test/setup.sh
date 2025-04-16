#!/bin/bash
# Setup script for testing environment

# Create a logs directory if it doesn't exist
mkdir -p logs

# Create a sample log file for testing
echo "This is a test log entry 1" > logs/test.log
echo "This is a test log entry 2" >> logs/test.log
echo "This is a test log entry 3" >> logs/test.log

# Start the mock server in the background
go run test/mock/mock_server.go &
MOCK_PID=$!

# Wait for the mock server to start
sleep 2

echo "Mock server started with PID $MOCK_PID"
echo "Test environment is ready"
echo "Sample logs created in logs/test.log"
echo "To run tests, use: make test or make test-integration"
echo ""
echo "Press Ctrl+C to stop the mock server"

# Wait for Ctrl+C
trap "kill $MOCK_PID; echo 'Mock server stopped'; exit 0" INT
wait 