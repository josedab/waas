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

### Changed
- Delivery engine now routes permanently failed deliveries to DLQ
- Services constructors return errors instead of panicking
- Server wires next-gen v7 packages into API routes
- Makefile: added `test-coverage`, `validate-env`, `migrate-reset`, and `audit-deps` targets
- Pre-commit hook detects secrets and env files

### Fixed
- SSRF protection in delivery engine with extracted config constants
- WebSocket origin validation and extracted timing constants
- Database connection now requires `DATABASE_URL` (no silent fallback)
- Queue retry delay bit-shift overflow on high attempt counts

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
