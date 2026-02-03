# WaaS Quick Reference

> Cheat sheet for common development tasks. See [ARCHITECTURE.md](../ARCHITECTURE.md) for full details.

## Quick Start

```bash
# Clone & setup
git clone <repo-url> && cd waas/backend
make dev-setup          # Creates .env, starts containers, runs core migrations (5 tables)

# Full setup with all migrations (core + enterprise features):
# make dev-setup-full

# Faster alternative: only core tables (tenants, endpoints, deliveries, analytics, quotas)
# make ensure-env && docker-compose up -d && make migrate-core

# Run everything
make run-all            # API + delivery engine + analytics

# Run individually
make run-api            # API server on :8080
make run-delivery       # Delivery engine (processes queue)
make run-analytics      # Analytics collector
make run-dashboard      # React dashboard (Vite dev server)
```

## Testing

```bash
make test               # Core tests with coverage summary
make test-all           # All tests including enterprise packages
make test-coverage      # Per-package coverage breakdown
make test-integration   # Integration tests in Docker (isolated)

# Run a single package test
go test -v ./pkg/repository/...
go test -v -run TestCreateEndpoint ./pkg/repository/...
```

## Database

```bash
make migrate-up             # Run all pending migrations
make migrate-down           # Rollback all migrations
make migrate-rollback-last  # Rollback last migration only
make migrate-status         # Show current migration version
make migrate-core           # Run only core migrations (1-5)
make migrate-reset          # ⚠️ Drop all + re-migrate (interactive)

# Manual migration creation
migrate create -ext sql -dir migrations -seq <name>
```

## Code Quality

```bash
make lint               # Run golangci-lint
make fmt                # Format code (gofmt)
make vet                # Run go vet
make build-check        # Compile all packages (no binary output)
make audit-deps         # Check for outdated/vulnerable deps
```

## Docker

```bash
make docker-up          # Start postgres + redis + app services
make docker-down        # Stop all containers
make docker-build       # Build Docker images

# With observability stack
docker-compose -f docker-compose.yml -f docker-compose.observability.yml up
```

## Key Directories

```
internal/api/           # HTTP handlers, server setup, middleware
pkg/models/             # Core data models (Tenant, Endpoint, Delivery, Event)
pkg/repository/         # Database layer (PostgreSQL via sqlx)
pkg/queue/              # Message queue (Redis-backed)
pkg/auth/               # Authentication & API key management
pkg/errors/             # Structured error responses
pkg/delivery/           # Delivery engine (retry logic, scheduling)
migrations/             # SQL migration files
```

## Common Patterns

### Adding a new API endpoint
1. Define models in `pkg/<feature>/models.go`
2. Add repository in `pkg/<feature>/repository.go` (implement `Repository` interface)
3. Add service in `pkg/<feature>/service.go` (business logic)
4. Add handlers in `pkg/<feature>/handlers.go` (HTTP layer)
5. Wire into `internal/api/server.go` (import, field, init, routes)

### Error responses
```go
import pkgerrors "github.com/josedab/waas/pkg/errors"

// Use structured errors
pkgerrors.AbortWithNotFound(c, "endpoint", endpointID)
pkgerrors.AbortWithInternalError(c, err)
pkgerrors.RespondWithError(c, pkgerrors.HandleRepositoryError(err))
```

### Tenant isolation
```go
tenantID := c.GetString("tenant_id")  // Set by auth middleware
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | (required) | PostgreSQL connection string |
| `REDIS_URL` | `localhost:6379` | Redis connection string |
| `API_PORT` | `8080` | API server port |
| `JWT_SECRET` | (required) | JWT signing secret |
| `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |

## Useful Links

- [Architecture Guide](../ARCHITECTURE.md) — Package tiers, data flow, design decisions
- [Deployment Guide](deployment-guide.md) — Production deployment (K8s, Helm)
- [Contributing](../CONTRIBUTING.md) — Full contribution guide
- [Quick Contributing](../CONTRIBUTING_QUICK.md) — 5-minute contribution guide
- [API Docs](swagger.yaml) — OpenAPI specification
