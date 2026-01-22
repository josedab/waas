# Webhook Service Platform

A comprehensive webhook-as-a-service platform built with Go microservices architecture.

## Project Structure

```
webhook-platform/
├── cmd/                    # Application entry points
│   ├── api-service/        # Webhook API service
│   ├── delivery-engine/    # Webhook delivery engine
│   └── analytics-service/  # Analytics and monitoring service
├── internal/               # Private application code
│   ├── api/               # API service implementation
│   ├── delivery/          # Delivery engine implementation
│   └── analytics/         # Analytics service implementation
├── pkg/                   # Shared packages
│   ├── models/           # Data models
│   ├── database/         # Database utilities
│   └── utils/            # Common utilities
├── migrations/           # Database migrations
├── sdk/                  # Client SDKs (Go, Python, Node.js, Java, Ruby, PHP)
├── web/dashboard/        # React web dashboard
├── docker-compose.yml    # Local development environment
└── Makefile             # Development commands (run `make help`)
```

## Quick Start

### Prerequisites

**Required:**

- **Go 1.24+** - [Download](https://go.dev/dl/)
- **Docker and Docker Compose** - [Install Docker](https://docs.docker.com/get-docker/)

**Optional (for development):**

- **golang-migrate CLI** - Local database migration tool (auto-detected; falls back to Docker image)
  ```bash
  # macOS
  brew install golang-migrate

  # Linux
  curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
  sudo mv migrate /usr/local/bin/

  # Go install
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  ```
- **golangci-lint** - Linter
  ```bash
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  ```
- **swag** - Swagger doc generation
  ```bash
  go install github.com/swaggo/swag/cmd/swag@latest
  ```

Verify your setup with:
```bash
make validate-setup
```

### Development Setup

1. Clone the repository
2. Start the development environment:
   ```bash
   make dev-setup
   ```

This will create `.env` from the template (if needed), start PostgreSQL and Redis containers, wait for them to be healthy, and run database migrations.

### Running Services

Start all services with Docker:
```bash
make docker-up
```

Or run individual services locally:
```bash
make run-api        # API service on :8080
make run-delivery   # Delivery engine
make run-analytics  # Analytics service on :8082
make run-all        # All three services at once
```

### Verify It Works

After starting the API service:

```bash
# Health check
curl -s http://localhost:8080/health

# Create a tenant (returns your API key)
curl -s -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name": "my-test-tenant", "email": "test@example.com"}'

# Send a test webhook (replace <your-api-key> with the key from above)
curl -s -X POST http://localhost:8080/api/v1/webhooks/test \
  -H "Content-Type: application/json" \
  -H "X-API-Key: <your-api-key>" \
  -d '{"url": "https://httpbin.org/post", "payload": {"hello": "world"}}'

# Browse the interactive API docs
open http://localhost:8080/docs/
```

Or run the automated smoke test:
```bash
make smoke-test
```

### Database Migrations

```bash
make migrate-up     # Apply all migrations (core + enterprise features)
make migrate-core   # Apply core-only migrations (tenants, endpoints, deliveries, analytics, quotas)
make migrate-down   # Rollback migrations
```

> **Tip:** `migrate-core` runs only the first 5 migrations — enough for the core webhook workflow. Use `migrate-up` when you need enterprise features (transformations, billing, etc.).

### Building

```bash
make build
```

### Testing

```bash
make test                          # Core tests (fast — internal + key packages)
make test-all                      # All tests including enterprise packages
make -f Makefile.test test-unit    # Unit tests with coverage
make -f Makefile.test test-all     # All test suites (integration, e2e, perf, chaos)
make -f Makefile.test help         # See all test commands
```

### Code Quality

```bash
make lint           # Run golangci-lint
make fmt            # Check formatting
make vet            # Run go vet
```

### Available Commands

Run `make help` to see all development targets, or `make -f Makefile.test help` for test commands, or `make -f Makefile.prod help` for production operations.

## Services

### API Service (Port 8080)
- Webhook endpoint management
- Authentication and rate limiting
- Webhook sending API
- Swagger docs at `/docs/`

### Delivery Engine
- Webhook delivery processing
- Retry logic with exponential backoff
- Queue management

### Analytics Service (Port 8082)
- Metrics collection and aggregation
- Dashboard APIs
- Real-time monitoring via WebSocket

## Environment Variables

See [`.env.example`](.env.example) for the full list with defaults. Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://postgres:password@localhost:5432/webhook_platform?sslmode=disable` | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection string |
| `API_PORT` | `8080` | API service port |
| `ANALYTICS_PORT` | `8082` | Analytics service port |
| `JWT_SECRET` | - | JWT signing secret |
| `ENVIRONMENT` | `development` | `development` or `production` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

## Documentation

- [API Documentation](docs/README.md) - REST API reference and SDK usage
- [Architecture](ARCHITECTURE.md) - System design and package structure
- [Deployment Guide](docs/deployment-guide.md) - Production Kubernetes deployment
- [Contributing](CONTRIBUTING.md) - Development workflow and guidelines
- [SDK Development](sdk/CONTRIBUTING.md) - Building and maintaining SDKs
- [Dashboard](web/dashboard/README.md) - Frontend development
