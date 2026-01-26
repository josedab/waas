# WaaS Packages — Start Here

> **New to the codebase?** You only need to know **7 packages** to work on core webhook functionality. Everything else is optional enterprise features.

## 🟢 Core Packages (start here)

These packages handle the core webhook lifecycle: ingest → queue → deliver → track.

| Package | What It Does | Key Files |
|---------|-------------|-----------|
| `internal/api` | HTTP server, routing, middleware | `server.go`, `handlers/` |
| `pkg/models` | Data models (Tenant, Endpoint, Delivery, Event) | `*.go` |
| `pkg/repository` | Database layer (PostgreSQL via sqlx) | `*_repository.go` |
| `pkg/queue` | Message queue (Redis-backed job processing) | `consumer.go`, `producer.go` |
| `pkg/auth` | API key auth, JWT, tenant middleware | `middleware.go`, `api_key.go` |
| `pkg/errors` | Structured error types and helpers | `types.go`, `helpers.go`, `definitions.go` |
| `pkg/signatures` | HMAC-SHA256 webhook signing | `signatures.go` |

### Also useful for day-to-day work

| Package | What It Does |
|---------|-------------|
| `pkg/database` | PostgreSQL connection management |
| `pkg/delivery` | Delivery engine (retry logic, backoff) |
| `pkg/utils` | Shared utilities (crypto, URL validation) |

## 🟡 Enterprise Packages (80+ — safe to ignore)

The remaining ~80 packages in `pkg/` provide enterprise capabilities: AI, billing, multi-cloud, observability, etc. They are **additive** — removing or ignoring them does not break core webhook delivery.

See [ARCHITECTURE.md](../ARCHITECTURE.md) for the full package list with status badges (🟢 Stable / 🟡 Beta / 🔵 Alpha).

## Common Tasks

| I want to... | Where to look |
|--------------|--------------|
| Add an API endpoint | `internal/api/handlers/` → `internal/api/server.go` (routes) |
| Add a data model | `pkg/models/` |
| Write a database query | `pkg/repository/` |
| Return a structured error | `pkg/errors/helpers.go` → use `AbortWithNotFound`, `HandleBindError`, etc. |
| Add a new feature package | Follow the pattern in any `pkg/<feature>/` with `models.go`, `repository.go`, `service.go`, `handlers.go` |
| Run tests | `make test` (core) or `make test-all` (everything) |
