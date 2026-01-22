"""
Basic usage example for the WAAS Python SDK.

Prerequisites:
  1. pip install waas-sdk   (or: pip install -e . from sdk/python/)
  2. A running WaaS API:    cd backend && make dev-setup && make run-api

This script walks through the core webhook workflow:
  - Create a tenant → get an API key
  - Create a webhook endpoint
  - Send a webhook
  - Check delivery status
"""

import httpx

from waas_sdk import WAASClient

BASE_URL = "http://localhost:8080/api/v1"


def create_tenant() -> str:
    """Register a tenant and return the API key."""
    resp = httpx.post(
        f"{BASE_URL}/tenants",
        json={"name": "python-sdk-demo", "email": "demo@example.com"},
    )
    resp.raise_for_status()
    data = resp.json()
    print(f"✅ Tenant created: {data['name']} (id: {data['id']})")
    return data["api_key"]


def main() -> None:
    # Step 1: Create a tenant to get an API key
    print("1. Creating tenant...")
    api_key = create_tenant()

    # Step 2: Initialize the SDK client
    client = WAASClient(api_key=api_key, base_url=BASE_URL)

    # Step 3: Create a webhook endpoint
    print("\n2. Creating webhook endpoint...")
    endpoint = client.endpoints.create(
        url="https://httpbin.org/post",
        custom_headers={"X-Source": "waas-python-demo"},
        retry_config={"max_attempts": 3, "initial_delay_ms": 1000},
    )
    print(f"✅ Endpoint created: {endpoint.id} → {endpoint.url}")

    # Step 4: Send a webhook
    print("\n3. Sending webhook...")
    delivery = client.webhooks.send(
        endpoint_id=endpoint.id,
        payload={
            "event": "user.created",
            "data": {"user_id": "42", "email": "alice@example.com"},
        },
        headers={"X-Event-Type": "user.created"},
    )
    print(f"✅ Webhook queued: delivery_id={delivery.delivery_id}")

    # Step 5: List endpoints
    print("\n4. Listing endpoints...")
    result = client.endpoints.list()
    for ep in result.data:
        print(f"   - {ep.id}  active={ep.is_active}  url={ep.url}")

    print("\n🎉 Done! Check http://localhost:8080/docs/ for the full API reference.")


if __name__ == "__main__":
    main()
