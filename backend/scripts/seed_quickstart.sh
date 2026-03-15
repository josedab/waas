#!/bin/bash
# seed_data.sh - Seeds the quickstart database with sample data
# This runs automatically during quickstart mode to provide a working demo

set -e

DB_URL="${DATABASE_URL:-postgres://postgres@localhost:5432/webhook_platform?sslmode=disable}"

echo "Seeding sample data..."

psql "$DB_URL" <<'SQL'
-- Create a demo tenant
INSERT INTO tenants (id, name, api_key_hash, subscription_tier, rate_limit_per_minute, monthly_quota, created_at, updated_at)
VALUES (
  uuid_generate_v4(),
  'Quickstart Demo',
  '$2a$10$quickstart_placeholder_hash_not_for_production',
  'free',
  100,
  10000,
  NOW(),
  NOW()
) ON CONFLICT DO NOTHING;

-- Create sample webhook endpoints (using the first tenant)
INSERT INTO webhook_endpoints (id, tenant_id, url, secret_hash, is_active, retry_config, custom_headers, created_at, updated_at)
SELECT
  uuid_generate_v4(),
  t.id,
  ep.url,
  ep.secret_hash,
  ep.is_active,
  '{"max_attempts":3,"initial_delay_ms":1000,"max_delay_ms":30000,"backoff_multiplier":2}'::jsonb,
  '{}'::jsonb,
  NOW(),
  NOW()
FROM tenants t
CROSS JOIN (VALUES
  ('https://httpbin.org/post', 'demo_secret_hash_1', true),
  ('https://httpbin.org/anything', 'demo_secret_hash_2', true),
  ('https://example.com/webhook', 'demo_secret_hash_3', false)
) AS ep(url, secret_hash, is_active)
WHERE t.name = 'Quickstart Demo'
ON CONFLICT DO NOTHING;

SQL

echo "Sample data seeded successfully!"
echo ""
echo "  Demo Tenant ID: demo-tenant-001"
echo "  Active Endpoints: 2"
echo "  Inactive Endpoints: 1"
echo ""
