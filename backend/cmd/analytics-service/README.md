# Analytics Service

The analytics service collects delivery metrics, aggregates statistics, and exposes a dashboard API. It also provides real-time monitoring via WebSocket and runs background workers for periodic metric computation.

## Build

```bash
cd backend
go build -o analytics-service ./cmd/analytics-service/
```

## Run

```bash
# Directly
./analytics-service

# Via Makefile (from backend/)
make run-analytics

# Via Docker Compose
docker compose up analytics-service
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ANALYTICS_PORT` | `8082` | HTTP listen port |
| `DATABASE_URL` | — | PostgreSQL connection string (required) |
| `REDIS_URL` | — | Redis connection string (required) |
| `ENVIRONMENT` | `development` | `development` or `production` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

See [`.env.example`](../../.env.example) for the full list of supported variables.

## Graceful Shutdown

The service listens for `SIGINT` and `SIGTERM` signals, waits up to 15 seconds for in-flight HTTP requests, and stops background workers before exiting.
