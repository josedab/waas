#!/bin/bash
# seed_data.sh - Seeds the quickstart database with sample data
# This runs automatically during quickstart mode to provide a working demo

set -e

DB_URL="${DATABASE_URL:-postgres://postgres@localhost:5432/webhook_platform?sslmode=disable}"

echo "Seeding sample data..."

psql "$DB_URL" <<'SQL'
-- Create a demo tenant
INSERT INTO tenants (id, name, email, api_key_hash, subscription_tier, is_active, rate_limit, monthly_quota, created_at, updated_at)
VALUES (
  'demo-tenant-001',
  'Quickstart Demo',
  'demo@waas-quickstart.local',
  'quickstart_demo_key_hash',
  'starter',
  true,
  100,
  10000,
  NOW(),
  NOW()
) ON CONFLICT (id) DO NOTHING;

-- Create sample webhook endpoints
INSERT INTO webhook_endpoints (id, tenant_id, url, secret_hash, is_active, retry_max_attempts, retry_initial_delay_ms, retry_max_delay_ms, created_at, updated_at)
VALUES
  ('ep-httpbin-001', 'demo-tenant-001', 'https://httpbin.org/post', 'demo_secret_hash', true, 3, 1000, 30000, NOW(), NOW()),
  ('ep-webhook-002', 'demo-tenant-001', 'https://webhook.site/demo', 'demo_secret_hash_2', true, 3, 1000, 30000, NOW(), NOW()),
  ('ep-inactive-003', 'demo-tenant-001', 'https://example.com/webhook', 'demo_secret_hash_3', false, 3, 1000, 30000, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

SQL

echo "Sample data seeded successfully!"
echo ""
echo "  Demo Tenant ID: demo-tenant-001"
echo "  Active Endpoints: 2"
echo "  Inactive Endpoints: 1"
echo ""
