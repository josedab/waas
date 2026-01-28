# ADR-0001: Architecture Overview

**Status:** Accepted
**Date:** 2025-06-01

## Context

WaaS (Webhook as a Service) needs a self-hosted, reliable webhook delivery platform. The system must handle multi-tenant webhook sending, retry logic with exponential back-off, real-time analytics, and be deployable via Docker Compose or Kubernetes.

## Decisions

### 1. Gin HTTP Framework

We chose [Gin](https://github.com/gin-gonic/gin) as the HTTP framework for the API service.

**Rationale:** Gin provides high performance, a mature middleware ecosystem (auth, CORS, rate limiting), and built-in Swagger integration. Its context-based request handling aligns well with the middleware chain pattern used for authentication and quota enforcement.

**Alternatives considered:** net/http (too low-level for middleware composition), Echo (smaller ecosystem), Chi (viable but Gin had broader team familiarity).

### 2. PostgreSQL for State, Redis for Queues

PostgreSQL stores all durable state (tenants, endpoints, deliveries, analytics). Redis serves as the queue backend for delivery jobs and as a cache for rate-limit counters.

**Rationale:** PostgreSQL provides ACID guarantees for billing and audit records. Redis sorted sets enable efficient delayed retry scheduling with `ZRANGEBYSCORE`. Separating state from queue concerns simplifies scaling each independently.

**Alternatives considered:** RabbitMQ (more operational overhead for self-hosted users), Kafka (overkill for single-node deployments).

### 3. goja JavaScript Transform Engine

Webhook payload transformations are executed in [goja](https://github.com/nicholasgasior/goja), a pure-Go JavaScript runtime.

**Rationale:** Allows users to write familiar JavaScript for payload transformations without requiring a separate runtime. goja runs in-process, avoids CGo, and can be sandboxed with execution timeouts. This keeps the deployment footprint minimal.

**Alternatives considered:** Lua (less familiar to most users), WASM (higher complexity), Jsonnet (limited expressiveness).

### 4. Tiered Package Structure

Packages under `pkg/` are organized into tiers:

| Tier | Examples | Description |
|------|----------|-------------|
| **Core** | `database`, `models`, `auth`, `queue`, `repository` | Essential for basic operation |
| **Standard** | `monitoring`, `transform`, `schema`, `replay` | Common features for production use |
| **Enterprise** | `billing`, `federation`, `blockchain`, `intelligence` | Advanced features for larger deployments |

**Rationale:** Tiered organization makes it clear which packages are required vs. optional. Contributors can work on enterprise features without risking core stability. Services in `internal/api/server.go` are initialized in phases matching these tiers.

### 5. Three-Service Architecture

The platform runs as three services: API (`:8080`), Delivery Engine (background), and Analytics (`:8082`).

**Rationale:** Separating the delivery engine allows it to scale independently based on queue depth. The analytics service has different performance characteristics (write-heavy aggregations, WebSocket connections) that benefit from isolation. All three share the same PostgreSQL and Redis instances for simplicity.

## Consequences

- New features must be classified into a package tier before implementation.
- Queue-related changes must consider both Redis sorted set semantics and PostgreSQL delivery records.
- JavaScript transforms must be tested with execution timeouts to prevent runaway scripts.
- Adding a new service type requires updates to Docker Compose, Kubernetes manifests, and the Makefile.
