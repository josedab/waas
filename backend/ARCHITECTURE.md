# Architecture

This document describes the system architecture of the WaaS (Webhook-as-a-Service) platform.

## Overview

WaaS is a multi-service Go platform that enables tenants to reliably send, receive, and manage webhooks. It uses PostgreSQL for persistence, Redis for queuing and caching, and exposes a REST API with Swagger documentation.

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

## Services

### API Service (`cmd/api-service` / `internal/api`)

The HTTP gateway for all client interactions. Built on the Gin framework.

**Responsibilities:**
- Tenant registration and API key authentication
- CRUD operations for webhook endpoints
- Accepting webhook send requests and enqueuing them
- Serving Swagger documentation at `/docs/`
- Rate limiting and quota enforcement

### Delivery Engine (`cmd/delivery-engine` / `internal/delivery`)

Background service that processes queued webhook deliveries.

**Responsibilities:**
- Consuming delivery jobs from the Redis queue
- Making HTTP requests to downstream webhook URLs
- Retry logic with exponential backoff
- Health monitoring of downstream endpoints
- Recording delivery attempt results

### Analytics Service (`cmd/analytics-service` / `internal/analytics`)

Provides metrics, aggregations, and real-time monitoring.

**Responsibilities:**
- Metrics collection and time-series aggregation
- Dashboard data APIs
- Real-time updates via WebSocket
- Prometheus metrics export at `/metrics`

## Data Flow

1. **Client** sends `POST /webhooks/send` with payload and target endpoint ID
2. **API Service** validates the request, authenticates the tenant, checks quotas
3. **API Service** enqueues the delivery job into **Redis**
4. **Delivery Engine** picks up the job, makes the HTTP request to the endpoint URL
5. **Delivery Engine** records the attempt (status, latency, response) in **PostgreSQL**
6. On failure, the engine schedules a retry with backoff
7. **Analytics Service** aggregates delivery data for dashboards

## What Actually Runs (Minimal Core)

If you stripped the platform to its essentials, the entire webhook lifecycle depends on only these packages:

```
internal/api          ← HTTP server, routes, middleware
internal/delivery     ← Delivery engine, retry logic
pkg/models            ← Data types (Tenant, Endpoint, Delivery, Event)
pkg/repository        ← PostgreSQL queries (sqlx)
pkg/queue             ← Redis job queue
pkg/auth              ← API key validation, tenant middleware
pkg/errors            ← Structured error responses
pkg/signatures        ← HMAC webhook signing
pkg/database          ← DB connection pool
pkg/utils             ← Shared helpers
```

**Everything else (~80 packages) is additive.** Enterprise packages provide advanced features (AI, billing, multi-cloud, observability) but can be removed without affecting core webhook delivery.

> **Rule of thumb:** If your change only involves sending or receiving webhooks, you should only need the 10 packages listed above.

## Package Structure

### Core (`internal/`)

Private service implementations. Not importable by external code.

| Package | Purpose |
|---------|---------|
| `internal/api` | API server setup, route registration, middleware wiring |
| `internal/delivery` | Delivery engine, HTTP client, retry logic, health monitor |
| `internal/analytics` | Analytics aggregation, WebSocket server |

### Shared Libraries (`pkg/`)

Reusable packages imported by services. Organized into three tiers:

> **New here?** Start with `internal/api`, `pkg/models`, `pkg/repository`, and `pkg/queue` — these are the core of the platform. Everything else builds on top.

#### 🟢 Core (required — the webhook platform essentials)

| Package | Purpose |
|---------|---------|
| `database` | PostgreSQL connection pool management |
| `repository` | Data access layer (tenant, endpoint, delivery, analytics repos) |
| `models` | Shared data structures |
| `queue` | Redis-based job queue (producer/consumer) |
| `auth` | API key auth, rate limiting, quota enforcement, subscriptions |
| `signatures` | Webhook payload signing (HMAC) |
| `errors` | Structured error types with categories, hints, docs |

#### 🔵 Standard (commonly used features)

| Package | Purpose |
|---------|---------|
| `transform` | Payload transformation engine (JavaScript via goja) |
| `schema` | JSON Schema validation for payloads |
| `idempotency` | Idempotency key management |
| `metrics` | Prometheus metrics middleware |
| `monitoring` | Alerting, health checks, tracing, system monitoring |
| `security` | Auth middleware, tenant isolation, encryption |
| `replay` | Event replay functionality |
| `autoretry` | Automated retry policies |
| `remediation` | Auto-remediation on delivery failures |
| `mocking` | Mock webhook endpoints for testing |

#### 🟡 Enterprise (advanced / optional features)

These packages provide enterprise-grade capabilities. They are not required for the core webhook workflow. Most are loaded unconditionally in `server.go` but function independently — you can safely ignore them when working on core webhook features.

> **Tip:** If you're fixing a bug or adding a feature, you almost certainly only need packages from the 🟢 Core and 🔵 Standard tiers. Enterprise packages are additive and won't affect core webhook delivery.

**Status key:** 🟢 Stable — 🟡 Beta — 🔵 Alpha — ⚪ Planned

**Observability & Analytics:**

| Package | Purpose | Status |
|---------|---------|--------|
| `otel` | OpenTelemetry integration | 🟡 Beta |
| `observability` | Observability utilities | 🟡 Beta |
| `anomaly` | Anomaly detection on delivery patterns | 🔵 Alpha |
| `tracing` | Distributed tracing | 🟡 Beta |
| `analyticsembed` | Embeddable analytics SDK | 🔵 Alpha |

**Advanced Features:**

| Package | Purpose | Status |
|---------|---------|--------|
| `ai` | AI/ML classification and pattern analysis | 🔵 Alpha |
| `prediction` | Predictive analytics for delivery success | 🔵 Alpha |
| `flow` | Workflow/event flow orchestration | 🟡 Beta |
| `connectors` | Third-party service connectors | 🔵 Alpha |
| `streaming` | Event streaming | 🟡 Beta |
| `eventsourcing` | Event sourcing patterns | 🔵 Alpha |
| `cdc` | Change data capture | 🟡 Beta |
| `graphql` / `graphqlsub` | GraphQL API and subscriptions | 🔵 Alpha |
| `intelligence` | AI-powered failure prediction & anomaly detection | 🔵 Alpha |
| `flowbuilder` | Visual DAG workflow builder with execution engine | 🔵 Alpha |
| `timetravel` | Event replay & point-in-time recovery | 🔵 Alpha |
| `callback` | Bi-directional webhooks with correlation tracking | 🔵 Alpha |
| `collabdebug` | Multiplayer debugging sessions | 🔵 Alpha |

**Infrastructure:**

| Package | Purpose | Status |
|---------|---------|--------|
| `multicloud` / `cloud` | Multi-cloud provider support | 🟡 Beta |
| `multiregion` / `georouting` | Multi-region routing | 🟡 Beta |
| `federation` | Cross-region federation | 🔵 Alpha |
| `edge` | Edge computing support | 🔵 Alpha |
| `gateway` | API gateway patterns | 🔵 Alpha |
| `protocols` / `protocolgw` | Multi-protocol support (HTTP, gRPC, MQTT) | 🔵 Alpha |
| `cloudmanaged` | Managed cloud offering (signup, billing, tiers) | 🔵 Alpha |
| `whitelabel` | Custom domains, branding, sub-tenant management | 🔵 Alpha |

**Business:**

| Package | Purpose | Status |
|---------|---------|--------|
| `billing` / `monetization` / `costing` | Billing and usage metering | 🟡 Beta |
| `marketplace` / `catalog` | Extension marketplace | 🔵 Alpha |
| `compliancecenter` / `contracts` | Compliance (GDPR, HIPAA, SOC2) | 🔵 Alpha |
| `blockchain` | Blockchain audit trail | 🔵 Alpha |
| `pluginmarket` | Plugin SDK, marketplace, lifecycle, reviews | 🔵 Alpha |
| `costengine` | Cost attribution engine | 🔵 Alpha |

**Security & Operations:**

| Package | Purpose | Status |
|---------|---------|--------|
| `waf` | Payload scanning, WAF rules, threat detection | 🔵 Alpha |
| `zerotrust` | Zero-trust security model | 🔵 Alpha |
| `mtls` | mTLS certificate management | 🔵 Alpha |
| `canary` | Canary deployments | 🔵 Alpha |
| `chaos` | Chaos engineering utilities | 🔵 Alpha |
| `gitops` | GitOps configuration management | 🔵 Alpha |

**Developer Tools:**

| Package | Purpose | Status |
|---------|---------|--------|
| `playground` | Interactive webhook playground | 🟡 Beta |
| `embed` | Embeddable webhook widgets | 🟡 Beta |
| `metaevents` | Meta-event notifications | 🟡 Beta |
| `workflow` | Workflow automation | 🟡 Beta |
| `versioning` | API versioning | 🟡 Beta |
| `smartlimit` | Intelligent rate limiting | 🟡 Beta |
| `pushbridge` | Push notification bridge | 🔵 Alpha |
| `debugger` | Time-travel debugging | 🔵 Alpha |
| `docgen` | Auto-generate webhook docs, code samples, widgets | 🔵 Alpha |
| `sandbox` | Webhook replay sandbox | 🔵 Alpha |

## Database

PostgreSQL 15 with migrations managed by golang-migrate.

**Core tables:** `tenants`, `webhook_endpoints`, `delivery_attempts`
**Supporting tables:** `analytics_*`, `quotas`, `secrets`, `audit_logs`, `transformations`, `idempotency_keys`, `schemas`

Migrations live in `migrations/` and are applied via `make migrate-up`.

## Common Questions for New Contributors

**Q: There are 90+ packages — which ones do I need to understand?**
Start with `internal/api`, `pkg/models`, `pkg/repository`, and `pkg/queue`. These handle 90% of the core webhook workflow. Enterprise packages (🟡 tier) are additive features that don't affect core delivery. See the status badges (🟢 Stable / 🟡 Beta / 🔵 Alpha) in the tables above.

**Q: What's the difference between `pkg/billing`, `pkg/monetization`, and `pkg/costing`?**
- `billing` — Stripe integration and subscription management
- `monetization` — Usage metering and pricing tier enforcement
- `costing` — Internal cost tracking and resource attribution

**Q: Do I need to wire my new package into `server.go`?**
Yes. All services are initialized in `internal/api/server.go`'s `NewServer()` function and routes are registered in `setupRoutes()`. Follow the existing pattern: add an import, a struct field, service initialization, and route registration.

**Q: How do I run just the tests for my package?**
```bash
make -f Makefile.test test-pkg PKG=./pkg/your-package
```

## Technology Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.24 |
| HTTP Framework | Gin |
| Database | PostgreSQL 15 (pgx driver) |
| Queue / Cache | Redis 7 |
| API Docs | Swagger/OpenAPI (swaggo) |
| Metrics | Prometheus |
| Tracing | OpenTelemetry |
| Container | Docker, Kubernetes |
| Testing | testify, Go testing, k6 |
| Frontend | React, TypeScript, Vite, Tailwind CSS |
