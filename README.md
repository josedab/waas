# WaaS — Webhook as a Service

[![CI](https://github.com/josedab/waas/actions/workflows/ci.yml/badge.svg)](https://github.com/josedab/waas/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/josedab/waas/branch/main/graph/badge.svg)](https://codecov.io/gh/josedab/waas)
![Go 1.24+](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Reliable webhook delivery infrastructure you can self-host. Send, receive, and manage webhooks with automatic retries, payload signing, real-time monitoring, and SDKs in six languages.

## Quick Start

```bash
git clone https://github.com/josedab/waas.git
cd waas/backend
make dev-setup    # starts PostgreSQL + Redis, runs core migrations (5 tables)
make run-api      # API on http://localhost:8080
make seed         # (optional) seed sample tenants, endpoints, and deliveries
```

> **Tip:** `make dev-setup` runs core migrations only (5 tables). For all 83 migrations (enterprise features), use `make dev-setup-full`.

Verify it works:

```bash
# Health check
curl -s http://localhost:8080/health
```

Expected response:

```json
{
  "status": "healthy",
  "timestamp": "2025-01-01T00:00:00Z",
  "version": "1.0.0",
  "components": { "database": { "status": "healthy" }, "redis": { "status": "healthy" }, "system": { "status": "healthy" } },
  "uptime": "5s"
}
```

```bash
# Create a tenant (returns your API key)
curl -s -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name": "my-test-tenant", "subscription_tier": "free"}'
```

Expected response:

```json
{
  "tenant": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "my-test-tenant",
    "subscription_tier": "free",
    "rate_limit_per_minute": 10,
    "monthly_quota": 1000,
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  },
  "api_key": "wh_..."
}
```

```bash
# Browse interactive API docs
open http://localhost:8080/docs/
```

## Key Commands

```bash
make help              # Show all available commands
make run-all           # Run API + delivery engine + analytics
make run-dashboard     # Run React dashboard (http://localhost:5173)
make test              # Run core tests with coverage
make test-integration  # Run integration tests in Docker
make smoke-test        # Quick smoke test against running API
make lint              # Run golangci-lint
make validate-setup    # Check prerequisites
```

## Architecture

```
                    ┌──────────────┐
                    │   Clients    │
                    │  (SDKs/API)  │
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │  API Service │  :8080
                    │   (Gin)      │
                    └──┬───────┬───┘
                       │       │
              ┌────────▼──┐ ┌──▼──────────┐
              │ PostgreSQL │ │    Redis     │
              │  (State)   │ │  (Queue)    │
              └────────┬──┘ └──┬──────────┘
                       │       │
                    ┌──▼───────▼───┐
                    │   Delivery   │  (background)
                    │    Engine    │
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │  Downstream  │
                    │  Webhooks    │
                    └──────────────┘

              ┌──────────────────┐
              │   Analytics      │  :8082
              │   Service        │
              └──────────────────┘
```

**Three services:**

| Service | Port | Purpose |
|---------|------|---------|
| **API Service** | 8080 | REST API, tenant management, webhook sending, Swagger docs |
| **Delivery Engine** | — | Background job processor, retries with exponential backoff |
| **Analytics Service** | 8082 | Metrics aggregation, dashboards, WebSocket real-time updates |

### Deployment

Production deployment is supported via [Helm chart](deploy/helm/waas/) and [Terraform modules](deploy/terraform/) for AWS (EKS + RDS + ElastiCache) and GCP (GKE + Cloud SQL + Memorystore). See the [Infrastructure guide](deploy/) for details.

## SDKs

| Language | Path | Status |
|----------|------|--------|
| Go | [`sdk/go`](backend/sdk/go) | ✅ Examples included |
| Python | [`sdk/python`](backend/sdk/python) | ✅ Examples included |
| Node.js | [`sdk/nodejs`](backend/sdk/nodejs) | ✅ Examples included |
| Java | [`sdk/java`](backend/sdk/java) | Available |
| Ruby | [`sdk/ruby`](backend/sdk/ruby) | Available |
| PHP | [`sdk/php`](backend/sdk/php) | Available |

### Framework Integrations

| Framework | Path | Status |
|-----------|------|--------|
| Django | [`sdk/django`](backend/sdk/django) | Available |
| Express | [`sdk/express`](backend/sdk/express) | Available |
| Laravel | [`sdk/laravel`](backend/sdk/laravel) | Available |
| Next.js | [`sdk/nextjs`](backend/sdk/nextjs) | Available |
| Rails | [`sdk/rails`](backend/sdk/rails) | Available |
| Spring Boot | [`sdk/spring-boot`](backend/sdk/spring-boot) | Available |

> **Note:** SDKs are bundled in the monorepo. See each SDK's README for local installation instructions.

## Documentation

| Document | What You'll Find |
|----------|-----------------|
| [**Setup & Commands**](backend/README.md) | Environment variables, all `make` targets, prerequisites, troubleshooting |
| [**Architecture**](backend/ARCHITECTURE.md) | System design, package tiers (core/standard/enterprise), FAQ for contributors |
| [**API Reference**](backend/docs/README.md) | REST endpoints, request/response examples, authentication |
| [**Contributing**](backend/CONTRIBUTING.md) | Development workflow, commit conventions, PR guidelines |
| [**Deployment**](backend/docs/deployment-guide.md) | Production Kubernetes deployment with Helm and Terraform |
| [**Feature Matrix**](backend/docs/FEATURE_MATRIX.md) | Platform capabilities and maturity status |
| [**Troubleshooting**](backend/docs/TROUBLESHOOTING.md) | Common issues and solutions for local development |
| [**Observability**](backend/docs/observability-guide.md) | Tracing, metrics, and Grafana dashboards setup |
| [**CLI Tool**](backend/cmd/waas-cli/README.md) | Terminal-based webhook management, testing, and GitOps workflows |
| [**Dashboard**](backend/web/dashboard/README.md) | React frontend development |
| [**Infrastructure**](deploy/) | Helm chart, Terraform modules for AWS & GCP |

## Prerequisites

- **Go 1.24+** — [Download](https://go.dev/dl/)
- **Docker & Docker Compose** — [Install](https://docs.docker.com/get-docker/)
- **Node.js 18+** *(optional, for dashboard)* — [Download](https://nodejs.org/)

> **IDE support:** Copy `go.work.example` to `go.work` at the repository root so that `gopls` (VS Code, GoLand) can resolve imports across the monorepo:
> ```bash
> cp go.work.example go.work
> ```

That's it. All other tools (database migrations, linters) are handled automatically or are optional.

## License

[MIT](https://opensource.org/licenses/MIT)
