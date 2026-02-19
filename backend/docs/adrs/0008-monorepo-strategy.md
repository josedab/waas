# ADR-0008: Monorepo Strategy for SDKs and Platform

**Status:** Accepted
**Date:** 2025-06-01

## Context

WaaS produces multiple deliverables: the core platform (API service, delivery engine, analytics), client SDKs (Go, TypeScript, Python), CLI tools, web dashboard, Terraform provider, and documentation. These components share models, API contracts, and versioning. We needed to decide whether to maintain them in a single repository or split across multiple repositories.

## Decision

We use a **monorepo** structure where all platform components live in a single Git repository under organized directories.

### Repository Layout

```
waas/
├── backend/           # Go platform (API, delivery, analytics)
│   ├── cmd/           # Service entry points
│   ├── internal/      # Private application logic
│   ├── pkg/           # Public packages (tiered)
│   ├── sdk/           # Generated SDK scaffolds
│   └── web/           # Dashboard frontend
├── deploy/            # Helm charts, Terraform modules
└── docs/              # Architecture docs, ADRs
```

## Rationale

- **Atomic changes:** API contract changes, SDK updates, and documentation can be landed in a single pull request, eliminating cross-repo synchronization.
- **Shared CI:** One CI pipeline validates that SDK changes are compatible with API changes, catching integration issues early.
- **Simplified dependency management:** Internal packages reference each other via Go module paths without version pinning or `replace` directives.
- **Contributor experience:** New contributors clone one repository and can navigate the full system. No need to discover and set up multiple repos.
- **Refactoring safety:** Renaming a model field updates all references (handlers, SDK generators, tests) in one commit.

## Alternatives Considered

| Alternative | Why Not |
|-------------|---------|
| **Polyrepo (one repo per component)** | Cross-repo PRs for API changes; version matrix complexity; harder to maintain consistency. |
| **Polyrepo with shared proto/schema repo** | Reduces some sync issues but adds a third repo to coordinate; still requires multi-repo PRs for breaking changes. |
| **Git submodules** | Fragile tooling; confusing for contributors; adds checkout complexity. |

## Consequences

- CI must be configured to run only affected tests (e.g., skip SDK tests when only backend changes).
- Release tagging may need component prefixes (e.g., `api/v1.2.0`, `sdk-go/v0.3.0`) as the project matures.
- Repository size will grow; Git LFS or shallow clones may be needed for large assets.
- All contributors need Go tooling installed even if they only work on the dashboard (mitigated by devcontainer configuration).
