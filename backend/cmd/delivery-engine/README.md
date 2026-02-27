# Delivery Engine

The delivery engine is a standalone worker process that consumes webhook delivery jobs from the Redis queue, dispatches HTTP requests to configured endpoints, and manages retry logic with exponential backoff.

## Build

```bash
cd backend
go build -o delivery-engine ./cmd/delivery-engine/
```

## Run

```bash
# Directly
./delivery-engine

# Via Makefile (from backend/)
make run-delivery

# Via Docker Compose
docker compose up delivery-engine
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | — | PostgreSQL connection string (required) |
| `REDIS_URL` | — | Redis connection string (required) |
| `ALLOW_INSECURE_TLS` | `false` | Skip TLS verification for webhook delivery |
| `ENVIRONMENT` | `development` | `development` or `production` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

See [`.env.example`](../../.env.example) for the full list of supported variables.

## Graceful Shutdown

The engine listens for `SIGINT` and `SIGTERM` signals and drains in-flight deliveries before exiting.
