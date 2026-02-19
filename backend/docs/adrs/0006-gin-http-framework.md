# ADR-0006: Gin HTTP Framework over Chi

**Status:** Accepted
**Date:** 2025-06-01

## Context

The WaaS API service needs an HTTP framework that supports middleware composition, route grouping, context propagation, and integrates well with Swagger documentation generation. The framework must handle high-throughput webhook ingestion with minimal overhead.

## Decision

We chose [Gin](https://github.com/gin-gonic/gin) as the HTTP framework.

## Rationale

- **Performance:** Gin uses httprouter under the hood, providing one of the fastest Go HTTP routers with zero-allocation path matching.
- **Middleware ecosystem:** Built-in support for CORS, recovery, logging, and a large ecosystem of third-party middleware (rate limiting, JWT auth, request ID).
- **Context-based request handling:** `gin.Context` carries request-scoped values (tenant ID, API key, request ID) through the middleware chain, aligning with the authentication and quota enforcement pattern.
- **Swagger integration:** `gin-swagger` provides automatic Swagger UI from annotations, reducing documentation maintenance.
- **Team familiarity:** The team had prior experience with Gin, reducing onboarding time.

## Alternatives Considered

| Alternative | Why Not |
|-------------|---------|
| **net/http + gorilla/mux** | Too low-level; requires manual middleware composition and lacks built-in context helpers. |
| **Chi** | Viable alternative with idiomatic `net/http` compatibility. Smaller middleware ecosystem and less Swagger tooling at the time of decision. Would be the second choice. |
| **Echo** | Similar feature set to Gin but smaller community and fewer battle-tested middleware options. |
| **Fiber** | Built on fasthttp, which breaks `net/http` compatibility. Makes integrating standard Go middleware difficult. |

## Consequences

- All HTTP handlers use `gin.Context` for request/response handling.
- Middleware follows the `gin.HandlerFunc` signature.
- Migrating to Chi or standard `net/http` would require rewriting all handler signatures and middleware.
- Route registration is centralized in `internal/api/server.go`.
