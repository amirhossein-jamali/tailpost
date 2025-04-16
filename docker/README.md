# Docker Configuration for TailPost

This directory contains Docker-related files for building, testing, and deploying TailPost.

## Files

- `Dockerfile`: Main Docker image definition for TailPost
- `docker-compose.yaml`: Configuration for running TailPost with supporting services
- `docker-compose.test.yml`: Configuration for running tests in Docker
- `config.docker.yaml`: Default configuration for Docker deployment
- `config.docker.encrypt.yaml`: Configuration with encryption enabled

## Usage

### Build the Docker Image

```bash
docker build -t tailpost:latest -f docker/Dockerfile .
```

### Run with Docker Compose

```bash
docker-compose -f docker/docker-compose.yaml up
```

### Run Tests in Docker

```bash
docker-compose -f docker/docker-compose.test.yml up --build --abort-on-container-exit
```

## Environment Variables

The Docker configuration supports the following environment variables:

- `TAILPOST_SERVER_URL`: The URL of the log server (default: http://log-server:8080/logs)
- `TAILPOST_LOG_LEVEL`: Log level (default: info)
- `TAILPOST_BATCH_SIZE`: Number of logs to batch before sending (default: 100)
- `TAILPOST_FLUSH_INTERVAL`: How often to flush logs, in seconds (default: 10s)

## Volumes

The Docker configuration mounts the following volumes:

- `/var/log:/var/log:ro`: Read-only access to logs directory
- `/etc/tailpost/config.yaml:/etc/tailpost/config.yaml`: Configuration file

Copyright © 2025 Amirhossein Jamali. All rights reserved. 