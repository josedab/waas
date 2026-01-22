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

These packages provide enterprise-grade capabilities. They are not required for the core webhook workflow.

**Observability & Analytics:**
- `otel` - OpenTelemetry integration
- `observability` - Observability utilities
- `anomaly` - Anomaly detection on delivery patterns

**Advanced Features:**
- `ai` - AI/ML classification and pattern analysis
- `prediction` - Predictive analytics for delivery success
- `flow` - Workflow/event flow orchestration
- `connectors` - Third-party service connectors
- `streaming` - Event streaming
- `eventsourcing` - Event sourcing patterns
- `cdc` - Change data capture
- `graphql` / `graphqlsub` - GraphQL API and subscriptions

**Infrastructure:**
- `multicloud` / `cloud` - Multi-cloud provider support
- `multiregion` / `georouting` - Multi-region routing
- `federation` - Cross-region federation
- `edge` - Edge computing support
- `gateway` - API gateway patterns
- `protocols` - Multi-protocol support (HTTP, gRPC, MQTT)

**Business:**
- `billing` / `monetization` / `costing` - Billing and usage metering
- `marketplace` / `catalog` - Extension marketplace
- `compliancecenter` / `contracts` - Compliance (GDPR, HIPAA, SOC2)
- `blockchain` - Blockchain audit trail

**Developer Tools:**
- `playground` - Interactive webhook playground
- `embed` - Embeddable webhook widgets
- `metaevents` - Meta-event notifications
- `workflow` - Workflow automation
- `versioning` - API versioning
- `smartlimit` - Intelligent rate limiting
- `pushbridge` - Push notification bridge
- `zerotrust` - Zero-trust security model
- `chaos` - Chaos engineering utilities

## Database

PostgreSQL 15 with migrations managed by golang-migrate.

**Core tables:** `tenants`, `webhook_endpoints`, `delivery_attempts`
**Supporting tables:** `analytics_*`, `quotas`, `secrets`, `audit_logs`, `transformations`, `idempotency_keys`, `schemas`

Migrations live in `migrations/` and are applied via `make migrate-up`.

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
