#!/usr/bin/env bash
set -euo pipefail

# Seed development data via the running API.
# Usage: ./scripts/seed-dev-data.sh [BASE_URL]

BASE_URL="${1:-http://localhost:8080}"
API_V1="${BASE_URL}/api/v1"

echo "Seeding demo data against ${BASE_URL}..."
echo ""

# --- Health check -----------------------------------------------------------
if ! curl -sf "${BASE_URL}/health" >/dev/null 2>&1; then
  echo "❌ API is not reachable at ${BASE_URL}. Start it with: make run-api"
  exit 1
fi

# --- Helper ------------------------------------------------------------------
create_tenant() {
  local name="$1" email="$2"
  RESP=$(curl -sf -X POST "${API_V1}/tenants" \
    -H "Content-Type: application/json" \
    -d "{\"name\": \"${name}\", \"email\": \"${email}\"}" 2>/dev/null) || {
    echo "  ⚠️  Could not create tenant '${name}' (may already exist)"
    return 1
  }
  API_KEY=$(echo "${RESP}" | grep -o '"api_key":"[^"]*"' | head -1 | cut -d'"' -f4)
  TENANT_ID=$(echo "${RESP}" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  echo "  ✅ Tenant '${name}' created (id=${TENANT_ID})"
}

create_endpoint() {
  local key="$1" url="$2" events="$3"
  ENDPOINT_RESP=$(curl -sf -X POST "${API_V1}/webhooks/endpoints" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${key}" \
    -d "{\"url\": \"${url}\", \"event_types\": ${events}}" 2>/dev/null) || {
    echo "  ⚠️  Could not create endpoint for ${url}"
    return 1
  }
  ENDPOINT_ID=$(echo "${ENDPOINT_RESP}" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  echo "  ✅ Endpoint created: ${url} (id=${ENDPOINT_ID})"
}

send_webhook() {
  local key="$1" endpoint_id="$2" payload="$3"
  curl -sf -X POST "${API_V1}/webhooks/send" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${key}" \
    -d "{\"endpoint_id\": \"${endpoint_id}\", \"payload\": ${payload}}" >/dev/null 2>&1 || {
    echo "  ⚠️  Could not send webhook to endpoint ${endpoint_id}"
    return 1
  }
  echo "  ✅ Webhook sent to endpoint ${endpoint_id}"
}

# --- Tenant 1: demo-shop ----------------------------------------------------
echo "Creating tenant: demo-shop..."
if create_tenant "demo-shop" "shop@example.com"; then
  SHOP_KEY="${API_KEY}"
  SHOP_ID="${TENANT_ID}"

  echo "Creating endpoints for demo-shop..."
  create_endpoint "${SHOP_KEY}" "https://httpbin.org/post" '["order.created","order.updated"]'
  SHOP_ENDPOINT1="${ENDPOINT_ID:-}"
  create_endpoint "${SHOP_KEY}" "https://httpbin.org/anything" '["payment.completed"]'
  SHOP_ENDPOINT2="${ENDPOINT_ID:-}"

  if [ -n "${SHOP_ENDPOINT1}" ]; then
    echo "Sending sample webhooks..."
    send_webhook "${SHOP_KEY}" "${SHOP_ENDPOINT1}" '{"event":"order.created","order_id":"ORD-001","total":99.99}'
    send_webhook "${SHOP_KEY}" "${SHOP_ENDPOINT1}" '{"event":"order.updated","order_id":"ORD-001","status":"shipped"}'
  fi
  if [ -n "${SHOP_ENDPOINT2}" ]; then
    send_webhook "${SHOP_KEY}" "${SHOP_ENDPOINT2}" '{"event":"payment.completed","amount":99.99,"currency":"USD"}'
  fi
fi

echo ""

# --- Tenant 2: demo-saas ----------------------------------------------------
echo "Creating tenant: demo-saas..."
if create_tenant "demo-saas" "saas@example.com"; then
  SAAS_KEY="${API_KEY}"

  echo "Creating endpoints for demo-saas..."
  create_endpoint "${SAAS_KEY}" "https://httpbin.org/post" '["user.signup","user.deleted"]'
  SAAS_ENDPOINT="${ENDPOINT_ID:-}"

  if [ -n "${SAAS_ENDPOINT}" ]; then
    echo "Sending sample webhooks..."
    send_webhook "${SAAS_KEY}" "${SAAS_ENDPOINT}" '{"event":"user.signup","user_id":"USR-100","email":"alice@example.com"}'
    send_webhook "${SAAS_KEY}" "${SAAS_ENDPOINT}" '{"event":"user.signup","user_id":"USR-101","email":"bob@example.com"}'
  fi
fi

echo ""

# --- Tenant 3: demo-ci (no endpoints — shows empty state) -------------------
echo "Creating tenant: demo-ci (empty, no endpoints)..."
create_tenant "demo-ci" "ci@example.com" || true

echo ""
echo "Seed complete! 🌱"
echo "  Browse the API:   ${BASE_URL}/docs/"
echo "  Dashboard:        make run-dashboard"
