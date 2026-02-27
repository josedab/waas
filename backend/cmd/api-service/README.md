# API Service

The API service is the main HTTP gateway for the WaaS platform. It exposes the REST API for webhook management, tenant operations, and developer tools.

## Build

```bash
cd backend
go build -o api-service ./cmd/api-service/
```

## Run

```bash
# Directly
./api-service

# Via Makefile (from backend/)
make run-api

# Via Docker Compose
docker compose up api-service
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | `8080` | HTTP listen port |
| `DATABASE_URL` | — | PostgreSQL connection string (required) |
| `REDIS_URL` | — | Redis connection string (required) |
| `JWT_SECRET` | — | JWT signing key (required) |
| `ENVIRONMENT` | `development` | `development` or `production` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

See [`.env.example`](../../.env.example) for the full list of supported variables.

## API Documentation

Swagger annotations in this package and throughout `internal/api` generate the OpenAPI spec. Regenerate with:

```bash
make docs
```

Access the Swagger UI at `http://localhost:8080/swagger/index.html` when running in development mode.
