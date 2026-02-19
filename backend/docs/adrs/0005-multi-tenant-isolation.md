# ADR-0005: Multi-Tenant Isolation Strategy

**Status:** Accepted
**Date:** 2025-06-01

## Context

WaaS is a multi-tenant platform where multiple organizations share the same database, Redis instance, and application services. Tenants must not be able to access each other's data, exhaust shared resources, or interfere with each other's webhook delivery. The isolation model must balance security with operational simplicity for self-hosted deployments.

## Decision

We implement **logical tenant isolation** at the application layer using API key–scoped middleware, row-level tenant filtering in the database, and per-tenant quota enforcement.

### Isolation Mechanisms

1. **Authentication middleware** (`pkg/auth`): Every request is scoped to a tenant via API key validation. The tenant ID is injected into the Gin context and propagated to all downstream operations.

2. **Row-level filtering** (`pkg/repository`): All database queries include a `tenant_id` WHERE clause. Repository methods accept tenant ID as a required parameter—there are no unscoped queries for tenant data.

3. **Quota enforcement** (`pkg/auth/quota_middleware.go`): Per-tenant rate limits and usage quotas prevent any single tenant from monopolizing shared resources.

4. **Queue isolation** (`pkg/queue`): Delivery jobs are tagged with tenant ID. Queue consumers respect tenant context for logging, metrics, and error attribution.

## Rationale

- **Operational simplicity:** A single PostgreSQL database and Redis instance is far simpler to operate than per-tenant databases, especially for self-hosted users.
- **Cost efficiency:** Shared infrastructure keeps hosting costs low for small deployments with few tenants.
- **Sufficient security:** Application-level isolation with row-level filtering provides strong data separation. For regulated environments, the database can be configured with PostgreSQL Row-Level Security (RLS) as an additional layer.
- **Consistent patterns:** Every package follows the same tenant-scoping pattern via context propagation, making the codebase predictable and auditable.

## Alternatives Considered

| Alternative | Why Not |
|-------------|---------|
| **Database-per-tenant** | High operational overhead; impractical for self-hosted users with many tenants. |
| **Schema-per-tenant** | Migration complexity scales linearly with tenant count; connection pooling becomes difficult. |
| **PostgreSQL RLS only** | RLS alone doesn't cover Redis, queue isolation, or application-level quota enforcement. |
| **Separate deployments** | Defeats the purpose of a multi-tenant SaaS platform; massive resource waste. |

## Consequences

- All repository methods must accept and filter by tenant ID—unscoped queries are a security bug.
- Integration tests must verify cross-tenant data isolation.
- Future compliance requirements (GDPR data residency) may require hybrid approaches where specific tenants get dedicated resources.
- Monitoring and alerting should be tenant-aware to detect noisy-neighbor issues.
