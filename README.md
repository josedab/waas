# WaaS — Webhook as a Service

[![CI](https://github.com/josedab/waas/actions/workflows/test.yml/badge.svg)](https://github.com/josedab/waas/actions/workflows/test.yml)
![Go 1.24+](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Reliable webhook delivery infrastructure you can self-host. Send, receive, and manage webhooks with automatic retries, payload signing, real-time monitoring, and SDKs in six languages.

## Quick Start

```bash
git clone https://github.com/josedab/waas.git
cd waas/backend
make dev-setup    # starts PostgreSQL + Redis, runs migrations
make run-api      # API on http://localhost:8080
```

Verify it works:

```bash
# Health check
curl -s http://localhost:8080/health

# Create a tenant (returns your API key)
curl -s -X POST http://localhost:8080/api/v1/tenants \
  -H "Content-Type: application/json" \
  -d '{"name": "my-test-tenant", "email": "test@example.com"}'

# Browse interactive API docs
open http://localhost:8080/docs/
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

## SDKs

| Language | Path | Status |
|----------|------|--------|
| Go | [`sdk/go`](backend/sdk/go) | ✅ Examples included |
| Python | [`sdk/python`](backend/sdk/python) | ✅ Examples included |
| Node.js | [`sdk/nodejs`](backend/sdk/nodejs) | ✅ Examples included |
| Java | [`sdk/java`](backend/sdk/java) | Available |
| Ruby | [`sdk/ruby`](backend/sdk/ruby) | Available |
| PHP | [`sdk/php`](backend/sdk/php) | Available |

## Documentation

| Document | Description |
|----------|-------------|
| [Full README](backend/README.md) | Detailed setup, commands, environment variables |
| [Architecture](backend/ARCHITECTURE.md) | System design and package structure |
| [API Docs](backend/docs/README.md) | REST API reference with request/response examples |
| [Contributing](backend/CONTRIBUTING.md) | Development workflow and guidelines |
| [Deployment](backend/docs/deployment-guide.md) | Production Kubernetes deployment |

## Prerequisites

- **Go 1.24+** — [Download](https://go.dev/dl/)
- **Docker & Docker Compose** — [Install](https://docs.docker.com/get-docker/)

That's it. All other tools (database migrations, linters) are handled automatically or are optional.

## License

[MIT](https://opensource.org/licenses/MIT)
