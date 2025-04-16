# Test Directory

This directory contains testing utilities and integration tests for the Tailpost project.

## Structure

- `mock/` - Contains mock servers for testing
  - `mock_server.go` - A simple HTTP server that receives logs for testing

## Running Tests

To run unit tests across the entire project:

```bash
make test
```

To run integration tests:

```bash
make test-integration
```

## Mock Server

The mock server is a simple HTTP server that receives logs and prints them to the console. It's useful for testing the Tailpost agent without a real log processing backend.

To run the mock server:

```bash
make run-mock
```

Or directly:

```bash
go run test/mock/mock_server.go
```

The mock server listens on port 8081 and exposes the following endpoints:

- `POST /logs` - Receives logs in JSON format
- `GET /health` - Health check endpoint 