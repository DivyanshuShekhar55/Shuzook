# Shuzook

Small playground project for learning observability in Go.

This repo contains:
- a Go API service
- OpenTelemetry instrumentation (traces, logs, metrics)
- Fluent Bit log shipping
- ClickStack all-in-one for local observability UI + backend

## Blog

I wrote a full deep-dive blog that explains the implementation concepts.

Read it here: https://medium.com/@shekdivyanshu/what-i-learned-about-observability-in-10-days-a370c143d2ee

## Tech Used

- Go
- OpenTelemetry (SDK + otelhttp + otelzap)
- Zap logger
- Lumberjack log rotation
- Fluent Bit
- ClickStack all-in-one Docker image
- Docker Compose

## Quick Setup

1. Go to project folder:
	cd shazam

2. Start full stack:
	docker compose up -d --build

3. Open apps:
	- API: http://localhost:8081
	- ClickStack UI: http://localhost:18080

4. Stop stack:
	docker compose down

## Useful Commands

- Follow container logs:
  docker compose logs -f

- Rebuild and restart:
  docker compose up -d --build

- Remove containers and volumes:
  docker compose down -v

## Project Layout

- shazam/cmd/api: API service code
- shazam/internals/otel: OpenTelemetry SDK setup
- shazam/fluent-bit: Fluent Bit config
- shazam/docker-compose.yml: local full stack
