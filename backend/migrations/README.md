# Database Migrations

83 migrations organized by feature tier. For most development, you only need the first 5.

## Quick Commands

```bash
make migrate-up             # Run ALL migrations (83 total)
make migrate-core           # Run only core migrations (1-5) — fastest setup
make migrate-status         # Show current migration version
make migrate-rollback-last  # Undo the last migration
make migrate-reset          # ⚠️ Drop everything and re-migrate (interactive)
```

## Migration Tiers

### Core (Migrations 1-5) — Required

These create the essential tables for webhook delivery. Use `make migrate-core` for the fastest dev setup.

| # | Migration | Table(s) |
|---|-----------|----------|
| 1 | `000001_create_tenants_table` | `tenants`, `api_keys` |
| 2 | `000002_create_webhook_endpoints_table` | `webhook_endpoints` |
| 3 | `000003_create_delivery_attempts_table` | `delivery_attempts` |
| 4 | `000004_create_analytics_tables` | `analytics_events`, `analytics_daily` |
| 5 | `000005_create_quota_tables` | `tenant_quotas`, `usage_records` |

### Standard (Migrations 6-13) — Common features

These add commonly used features beyond basic webhook delivery.

| # | Migration | Feature |
|---|-----------|---------|
| 6 | `000006_create_secrets_table` | Secret management |
| 7 | `000007_create_audit_logs_table` | Audit logging |
| 8 | `000008_create_transformations_table` | Payload transformations |
| 9 | `000009_create_idempotency_keys_table` | Idempotency |
| 10 | `000010_create_schemas_table` | Schema validation |
| 11 | `000011_create_replay_tables` | Event replay |
| 12 | `000012_create_gateway_tables` | API gateway |
| 13 | `000013_create_connectors_tables` | Third-party connectors |

### Enterprise (Migrations 14-83) — Advanced features

These support enterprise capabilities: AI, billing, multi-cloud, observability, compliance, etc.

Most developers can skip these unless working on specific enterprise features. They are safe to run — they only create new tables and don't modify core tables.

<details>
<summary>Full enterprise migration list (click to expand)</summary>

| Range | Feature Area |
|-------|-------------|
| 14-17 | Anomaly detection, multi-region, billing, AI debugging |
| 18-23 | Flows, meta-events, geo-routing, embed, mocking, costing |
| 24-28 | OpenTelemetry, protocols, cloud, event catalog, playground |
| 29-33 | Auto-retry, event sourcing, zero-trust, multicloud, contracts |
| 34-38 | Marketplace, AI composer, collaboration, replay, SDK generator |
| 39-43 | GraphQL, compliance, self-healing, bidirectional sync, federation |
| 44-48 | Edge functions, observability, smart limiting, chaos, CDC |
| 49-53 | Workflow, signatures, push bridge, billing v2, versioning |
| 54-58 | Federation, push credentials, streaming, remediation, blockchain |
| 59-63 | Prediction, monetization, federation clusters, GraphQL subs, SLA |
| 64-68 | mTLS, contracts v2, marketplace v2, event mesh, debugger |
| 69-73 | Cloud v2, Terraform, portal, tracing, canary |
| 74-78 | Auto-remediation, schema registry, sandbox, protocol gateway, analytics embed |
| 79-83 | Cost engine, GitOps, live migration, inbound, fanout |

</details>

## Creating New Migrations

```bash
# Create a new migration pair (up + down)
migrate create -ext sql -dir migrations -seq <descriptive_name>
```

This creates two files:
- `migrations/000084_<name>.up.sql` — Apply the migration
- `migrations/000084_<name>.down.sql` — Revert the migration

### Guidelines

- Every `up.sql` must have a corresponding `down.sql` that cleanly reverts it
- Use `IF NOT EXISTS` for table creation
- New enterprise features should not modify existing core tables (1-5)
- Add appropriate indexes for any columns used in WHERE clauses
- Include `tenant_id` column in all tenant-scoped tables
