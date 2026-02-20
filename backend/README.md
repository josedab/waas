# Webhook Service Platform

A comprehensive webhook-as-a-service platform built with Go microservices architecture.

## Project Structure

```
waas/
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
- **Node.js 18+** *(for dashboard)* — [Download](https://nodejs.org/). Uses pnpm (preferred) or npm.
  ```bash
  # Install pnpm (optional, npm also works)
  npm install -g pnpm
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

3. *(Optional)* Seed sample data after starting the API:
   ```bash
   make run-api      # Start the API first
   make seed         # Seed sample tenants, endpoints, and deliveries
   ```

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
  -H "Authorization: Bearer <your-api-key>" \
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
make migrate-up       # Apply all migrations (core + enterprise features)
make migrate-core     # Apply core-only migrations (tenants, endpoints, deliveries, analytics, quotas)
make migrate-down     # Rollback ALL migrations (caution: destructive)
make migrate-status   # Show current migration version
```

> **Tip:** `migrate-core` runs only the first 5 migrations — enough for the core webhook workflow. Use `migrate-up` when you need enterprise features (transformations, billing, etc.).
>
> **Rollback safety:** `migrate-down` rolls back **all** migrations, not just the last one. There are 83 migrations total. To roll back a single step, use `migrate -path migrations -database "$DATABASE_URL" down 1` directly.

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

This project uses three Makefiles. Run `make help` in any of them to see the full list.

| Makefile | Purpose | Help command |
|----------|---------|--------------|
| `Makefile` | Day-to-day development (build, run, test, lint, migrate, docker) | `make help` |
| `Makefile.test` | Extended test suites (integration, e2e, performance, chaos, benchmarks) | `make -f Makefile.test help` |
| `Makefile.prod` | Production operations (build images, deploy, scale, rollback) | `make -f Makefile.prod help` |

**Quick reference — most-used targets:**

```bash
# Setup & diagnostics
make dev-setup          # One-command environment setup
make doctor             # Full environment health check
make validate-setup     # Check prerequisites

# Run services
make dev                # API with hot-reload (auto-installs Air)
make dev-all            # All services with hot-reload
make run-all            # All services without hot-reload
make dev-logs           # All services with colored, prefixed log output

# Quality
make check              # fmt + vet + lint + test in one shot
make build-check        # Compile-check all packages without binaries
make lint-fast          # Lint only changed packages (fast feedback)
make test-watch         # Re-run tests on file changes

# Database
make migrate-up         # Apply all migrations
make migrate-core       # Apply core-only migrations (5 tables)
make migrate-status     # Show current version
make migrate-rollback-last  # Rollback only the last migration

# Dependencies & auditing
make deps               # Download and tidy Go modules
make audit-deps         # Check for outdated/vulnerable dependencies

# Git hooks
make install-hooks      # Install pre-commit hooks (gofmt + go vet)
make uninstall-hooks    # Remove pre-commit hooks
```

> **Tip:** All targets in `backend/Makefile` are also available from the repository root — just run `make <target>` from the top-level directory.

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
| `ADMIN_TENANT_IDS` | - | Comma-separated tenant IDs with admin privileges |
| `ALLOW_INSECURE_TLS` | `false` | Skip TLS certificate verification for webhook delivery |
| `CORS_ALLOWED_ORIGINS` | - | Comma-separated allowed CORS origins |
| `APP_ENV` | `development` | Application environment (`development`, `production`) |

## Documentation

- [API Documentation](docs/README.md) - REST API reference and SDK usage
- [Architecture](ARCHITECTURE.md) - System design and package structure
- [Deployment Guide](docs/deployment-guide.md) - Production Kubernetes deployment
- [Error Catalog](docs/ERROR_CATALOG.md) - Complete reference for all API error codes
- [Quick Reference](docs/QUICK_REFERENCE.md) - Cheat sheet for common development tasks
- [Contributing](CONTRIBUTING.md) - Development workflow and guidelines
- [SDK Development](sdk/CONTRIBUTING.md) - Building and maintaining SDKs
- [Dashboard](web/dashboard/README.md) - Frontend development

## Troubleshooting

<details>
<summary><strong>Port conflicts — "address already in use"</strong></summary>

Another process is occupying a port the platform needs (5432, 6379, 8080, or 8082).

```bash
# Identify what's using the port (e.g. 8080)
lsof -i :8080

# Kill the process (replace <PID> with the actual PID)
kill <PID>

# Or change the port via .env
echo "API_PORT=9090" >> .env
```

Run `make validate-setup` to see which ports are available at a glance.
</details>

<details>
<summary><strong>Docker not running or <code>docker-compose</code> errors</strong></summary>

Many make targets depend on Docker. If you see "Cannot connect to the Docker daemon" or similar:

```bash
# Verify Docker is running
docker info >/dev/null 2>&1 && echo "Docker OK" || echo "Docker not running"

# macOS: start Docker Desktop from Applications, or:
open -a Docker

# Linux: start the daemon
sudo systemctl start docker
```

If `docker-compose` is not found, Docker Compose V2 ships as a `docker` subcommand:

```bash
docker compose version   # V2 (preferred)
docker-compose --version # V1 (legacy)
```
</details>

<details>
<summary><strong>Migration errors — "dirty database" or version mismatch</strong></summary>

If a migration failed mid-way, the schema version can be left in a "dirty" state.

```bash
# Check current version and dirty flag
make migrate-status

# Force-set the version to the last successful migration (e.g. version 5)
migrate -path migrations -database "$DATABASE_URL" force 5

# Then re-run migrations
make migrate-up
```

If the `migrate` CLI isn't installed locally, use the Docker fallback:

```bash
docker run --rm --network host -v $(pwd)/migrations:/migrations \
  migrate/migrate -path=/migrations -database "$DATABASE_URL" force 5
```

As a last resort, reset everything (destroys all data):

```bash
make migrate-reset
```
</details>

<details>
<summary><strong>PostgreSQL fails to start or <code>pg_isready</code> hangs</strong></summary>

Port 5432 may already be in use by another PostgreSQL instance.

```bash
# Check what's using the port
lsof -i :5432

# Stop existing containers and retry
docker-compose down && docker-compose up -d
```
</details>

<details>
<summary><strong><code>make migrate-up</code> fails with "connection refused"</strong></summary>

PostgreSQL may not be ready yet. Wait a few seconds after `docker-compose up -d`:

```bash
docker-compose exec postgres pg_isready -U postgres
# Once it reports "accepting connections", retry:
make migrate-up
```
</details>

<details>
<summary><strong>API returns 401 Unauthorized</strong></summary>

You need a valid API key. Create a tenant first:

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name": "my-tenant", "email": "me@example.com"}'
```

The response includes your `api_key`. Pass it via the `Authorization` header:

```bash
curl -H "Authorization: Bearer YOUR_KEY_HERE" http://localhost:8080/api/v1/endpoints
```
</details>

<details>
<summary><strong><code>make dev-setup</code> shows "migrate: not found"</strong></summary>

This is expected — the Makefile automatically falls back to a Docker-based migration. If you see actual migration errors, ensure Docker is running:

```bash
docker info >/dev/null 2>&1 && echo "Docker OK" || echo "Docker not running"
```
</details>

<details>
<summary><strong><code>make lint</code> fails with "golangci-lint not found"</strong></summary>

Install golangci-lint:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

Ensure `$GOPATH/bin` (usually `~/go/bin`) is in your `PATH`.
</details>

<details>
<summary><strong>Tests fail with "database does not exist"</strong></summary>

Integration tests need a separate test database. Run:

```bash
make -f Makefile.test test-setup
```

This creates the test database and applies migrations using `docker-compose.test.yml`.
</details>
