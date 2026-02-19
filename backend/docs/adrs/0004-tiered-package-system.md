# ADR-0004: Tiered Package System (Core / Standard / Enterprise)

**Status:** Accepted
**Date:** 2025-06-01

## Context

WaaS has grown to 90+ packages spanning basic webhook delivery through advanced features like AI classification, blockchain audit trails, and multi-region federation. Contributors and operators need a clear way to understand which packages are essential, which add production value, and which are advanced/optional. The build and initialization system needs to reflect these boundaries.

## Decision

We organize all `pkg/` packages into three tiers: **Core**, **Standard**, and **Enterprise**.

| Tier | Purpose | Examples |
|------|---------|----------|
| **Core** | Essential for basic webhook operation | `database`, `models`, `auth`, `queue`, `repository`, `signatures`, `errors` |
| **Standard** | Common production features | `monitoring`, `transform`, `schema`, `replay`, `otel`, `analytics` |
| **Enterprise** | Advanced features for large deployments | `billing`, `federation`, `blockchain`, `intelligence`, `multicloud` |

## Rationale

- **Clarity for contributors:** New contributors can focus on Core packages without risk of breaking advanced features. Enterprise package maintainers can iterate independently.
- **Phased initialization:** Services in `internal/api/server.go` initialize in tier order. If an Enterprise package fails to initialize (e.g., missing Stripe key), Core functionality remains available.
- **Deployment flexibility:** Operators running small deployments can ignore Enterprise packages entirely. The tier system guides which environment variables and infrastructure are truly required.
- **Feature matrix communication:** The tier maps directly to the feature matrix in documentation, making it easy for users to understand what they get at each level.

## Alternatives Considered

| Alternative | Why Not |
|-------------|---------|
| **Flat package structure** | No visibility into what's essential vs. optional at 90+ packages. |
| **Separate repositories** | Complicates development workflow, versioning, and cross-package refactoring. |
| **Feature flags only** | Runtime flags don't communicate architectural intent to contributors. |
| **Plugin system** | Higher complexity; premature for current project maturity level. |

## Consequences

- Every new package must be assigned a tier before merge.
- Core packages have the highest stability bar—breaking changes require an ADR.
- Enterprise packages may have external dependencies (Stripe, cloud SDKs) that Core packages must never import.
- The FEATURE_MATRIX.md document must be updated when packages change tiers.
