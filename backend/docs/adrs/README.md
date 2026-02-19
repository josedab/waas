# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the WaaS project.

ADRs document significant architectural decisions, their context, and consequences. Use the template below when adding new records.

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [0001](0001-architecture-overview.md) | Architecture Overview | Accepted |
| [0002](0002-goja-javascript-transform-engine.md) | goja JavaScript Transform Engine | Accepted |
| [0003](0003-redis-message-queuing.md) | Redis over RabbitMQ for Message Queuing | Accepted |
| [0004](0004-tiered-package-system.md) | Tiered Package System (Core/Standard/Enterprise) | Accepted |
| [0005](0005-multi-tenant-isolation.md) | Multi-Tenant Isolation Strategy | Accepted |
| [0006](0006-gin-http-framework.md) | Gin HTTP Framework over Chi | Accepted |
| [0007](0007-redis-sorted-sets-retry.md) | Redis Sorted Sets for Delayed Retry | Accepted |
| [0008](0008-monorepo-strategy.md) | Monorepo Strategy for SDKs and Platform | Accepted |

## Creating a New ADR

1. Copy the template below into a new file: `NNNN-short-title.md`
2. Fill in all sections
3. Add the entry to the index above
4. Submit a PR for team review

### Template

```markdown
# ADR-NNNN: Title

**Status:** Proposed | Accepted | Deprecated | Superseded by ADR-XXXX
**Date:** YYYY-MM-DD

## Context
What is the issue that we're seeing that motivates this decision?

## Decision
What is the change that we're proposing and/or doing?

## Consequences
What becomes easier or more difficult because of this change?
```
