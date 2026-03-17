# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Dead letter queue (DLQ) package with filtering and replay (`pkg/dlq`)
- OpenAPI-to-webhook config generator (`pkg/openapigen`)
- Event catalog versioning, codegen, and changelog (`pkg/catalog`)
- Inbound webhooks with DLQ, health monitoring, content routing, and transforms (`pkg/inbound`)
- Fan-out routing rules with versioning and condition evaluation (`pkg/fanout`)
- WAF endpoint security scanning and threshold management (`pkg/waf`)
- Live migration with multi-provider import and compatibility layers (`pkg/livemigration`)
- Cloud-managed tenant isolation, SLA, and lifecycle management (`pkg/cloudmanaged`)
- Playground shared scenarios, test suites, and diff viewer
- CLI: generate, import/export, and migrate commands with CSV output
- Goreleaser configuration for CLI releases
- Enterprise packages: intelligence, flow builder, time-travel, plugin marketplace, white-label, callback, collaborative debug, docgen
- Observability stack with Prometheus and Grafana (`deploy/`)
- Dependabot configuration for Go modules and GitHub Actions
- Security scanning CI workflow
- Issue and pull request templates
- Helm chart, Terraform modules, and deployment guide
- Request ID (`X-Request-ID`) and API version (`X-API-Version`) response headers on all endpoints
- Structured request logging middleware with method, path, status, latency, and request ID
- Event type registry with well-known constants and `ValidateEventType()` for catalog integration
- Event type field on webhook send requests with optional validation
- Delivery attempt retention API (`DeleteOlderThan`) for table growth management
- Shared pagination helpers (`httputil.ParsePagination`, `RespondWithList`) for consistent list responses
- Queue message versioning for safe rolling deployments
- `BindJSON`, `BadRequest`, `NotFound`, `Forbidden`, `Conflict`, `ValidationError` error response helpers
- Database connection pool configurability via `DB_MAX_CONNS`, `DB_MIN_CONNS`, `DB_CONN_MAX_LIFETIME`, `DB_CONN_MAX_IDLE_TIME`

### Changed
- Delivery engine now routes permanently failed deliveries to DLQ
- Services constructors return errors instead of panicking
- Server wires next-gen v7 packages into API routes
- Makefile: added `test-coverage`, `validate-env`, `migrate-reset`, and `audit-deps` targets
- Pre-commit hook detects secrets and env files
- `UpdateWebhookEndpoint` route changed from PUT to PATCH (partial update semantics)
- Webhook endpoint list, tenant list, and delivery history use shared pagination helpers
- Queue consumer uses exponential backoff (100ms→30s cap) instead of fixed 100ms sleep on errors
- Delivery engine `/ready` returns 503 during graceful shutdown drain
- Health check `checkSystem()` now reports runtime memory and goroutine stats
- Readiness probe checks both database and Redis (previously only database)
- Smoke test expanded from 3 to 7 checks (endpoint creation, auth rejection, response headers)
- CI now tests `./cmd/...` packages in addition to `./pkg/...` and `./internal/...`
- Batch send reports skipped endpoint IDs instead of silently dropping them
- Webhook queue publish has 5-second context timeout to prevent hanging

### Fixed
- SSRF protection in delivery engine with extracted config constants
- WebSocket origin validation and extracted timing constants
- Database connection now requires `DATABASE_URL` (no silent fallback)
- Queue retry delay bit-shift overflow on high attempt counts
- Eliminated 51+ instances of `err.Error()` leaked to API clients across 16 handler files
- Go SDK error parsing now handles flat `{code, message}` response format (was only parsing nested format)
- Model validation now accepts `free` and `pro` subscription tiers (was rejecting valid tiers)
- Retry config validation ensures `initial_delay_ms ≤ max_delay_ms` on endpoint updates
- Removed unused `uuid` import from Go SDK client
- Response body truncation (64KB) now logs a warning when truncation occurs

### Documentation
- Architecture docs updated with package status badges and new packages
- Error catalog, feature matrix, testing guide, and troubleshooting guide
- Contributing quickstart, SDK development guide, and migrations guide
- Deployment guide with dev-to-production promotion path and env reference
- `migrate-core` quickstart tip in root README

## [0.1.0] - 2025-06-01

### Added
- Core webhook delivery engine with retry/backoff mechanisms
- REST API with tenant management and authentication
- PostgreSQL storage with Redis queue integration
- Real-time analytics with WebSocket support
- SDK support for Go, Python, Node.js, Java, Ruby, and PHP
- Swagger/OpenAPI documentation
- Docker Compose development setup
- Kubernetes deployment manifests

[Unreleased]: https://github.com/josedab/waas/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/josedab/waas/releases/tag/v0.1.0
