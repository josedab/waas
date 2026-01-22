"""Integration tests for WAAS Python SDK.

These tests require a running WAAS server. Set WAAS_API_KEY and WAAS_BASE_URL
environment variables to run these tests.

Run with: pytest tests/test_integration.py -v -m integration
"""

import os
import pytest
from waas_sdk import WAASClient, AsyncWAASClient
from waas_sdk.exceptions import WAASNotFoundError


# Skip all tests in this file if integration tests are not enabled
pytestmark = pytest.mark.integration


@pytest.fixture
def api_key():
    """Get API key from environment."""
    key = os.environ.get("WAAS_API_KEY")
    if not key:
        pytest.skip("WAAS_API_KEY not set")
    return key


@pytest.fixture
def base_url():
    """Get base URL from environment."""
    return os.environ.get("WAAS_BASE_URL", "http://localhost:8080/api/v1")


@pytest.fixture
def client(api_key, base_url):
    """Create a sync client for testing."""
    return WAASClient(api_key=api_key, base_url=base_url)


@pytest.fixture
def async_client(api_key, base_url):
    """Create an async client for testing."""
    return AsyncWAASClient(api_key=api_key, base_url=base_url)


class TestEndpointsIntegration:
    """Integration tests for endpoints service."""

    def test_create_and_get_endpoint(self, client):
        """Test creating and retrieving an endpoint."""
        # Create endpoint
        endpoint = client.endpoints.create(
            url="https://httpbin.org/post",
            retry_config={
                "max_attempts": 3,
                "initial_delay_ms": 500
            }
        )
        
        assert endpoint.id is not None
        assert endpoint.url == "https://httpbin.org/post"
        assert endpoint.is_active is True
        assert endpoint.secret is not None  # Secret returned on creation
        
        # Get endpoint
        fetched = client.endpoints.get(str(endpoint.id))
        assert fetched.id == endpoint.id
        assert fetched.url == endpoint.url
        
        # Clean up
        client.endpoints.delete(str(endpoint.id))
        
        # Verify deleted
        with pytest.raises(WAASNotFoundError):
            client.endpoints.get(str(endpoint.id))

    def test_list_endpoints(self, client):
        """Test listing endpoints."""
        # Create a few endpoints
        created = []
        for i in range(3):
            endpoint = client.endpoints.create(
                url=f"https://httpbin.org/post?test={i}"
            )
            created.append(endpoint)
        
        try:
            # List endpoints
            result = client.endpoints.list(per_page=10)
            assert "endpoints" in result
            assert len(result["endpoints"]) >= 3
        finally:
            # Clean up
            for endpoint in created:
                client.endpoints.delete(str(endpoint.id))

    def test_update_endpoint(self, client):
        """Test updating an endpoint."""
        # Create endpoint
        endpoint = client.endpoints.create(url="https://httpbin.org/post")
        
        try:
            # Update URL
            updated = client.endpoints.update(
                str(endpoint.id),
                url="https://httpbin.org/anything",
                is_active=False
            )
            
            assert updated.url == "https://httpbin.org/anything"
            assert updated.is_active is False
        finally:
            client.endpoints.delete(str(endpoint.id))


class TestWebhooksIntegration:
    """Integration tests for webhooks service."""

    def test_send_webhook(self, client):
        """Test sending a webhook."""
        # Create endpoint first
        endpoint = client.endpoints.create(url="https://httpbin.org/post")
        
        try:
            # Send webhook
            result = client.webhooks.send(
                endpoint_id=str(endpoint.id),
                payload={"event": "test.event", "data": {"message": "Hello!"}}
            )
            
            assert result.delivery_id is not None
            assert result.status == "pending"
            
            # Verify delivery was created
            delivery = client.deliveries.get(str(result.delivery_id))
            assert delivery.endpoint_id == endpoint.id
        finally:
            client.endpoints.delete(str(endpoint.id))


class TestDeliveriesIntegration:
    """Integration tests for deliveries service."""

    def test_list_deliveries(self, client):
        """Test listing deliveries."""
        result = client.deliveries.list(per_page=5)
        
        assert "deliveries" in result
        assert "total" in result
        assert isinstance(result["deliveries"], list)

    def test_retry_delivery(self, client):
        """Test retrying a delivery."""
        # Create endpoint and send webhook
        endpoint = client.endpoints.create(url="https://httpbin.org/post")
        
        try:
            send_result = client.webhooks.send(
                endpoint_id=str(endpoint.id),
                payload={"test": True}
            )
            
            # Wait a bit for processing
            import time
            time.sleep(2)
            
            # Retry the delivery
            retry_result = client.deliveries.retry(str(send_result.delivery_id))
            assert retry_result.delivery_id is not None
        finally:
            client.endpoints.delete(str(endpoint.id))


@pytest.mark.asyncio
class TestAsyncIntegration:
    """Async integration tests."""

    async def test_async_create_endpoint(self, async_client):
        """Test creating endpoint with async client."""
        async with async_client:
            endpoint = await async_client.endpoints.create(
                url="https://httpbin.org/post"
            )
            
            assert endpoint.id is not None
            assert endpoint.url == "https://httpbin.org/post"
            
            # Clean up
            await async_client.endpoints.delete(str(endpoint.id))

    async def test_async_send_webhook(self, async_client):
        """Test sending webhook with async client."""
        async with async_client:
            # Create endpoint
            endpoint = await async_client.endpoints.create(
                url="https://httpbin.org/post"
            )
            
            try:
                # Send webhook
                result = await async_client.webhooks.send(
                    endpoint_id=str(endpoint.id),
                    payload={"async_test": True}
                )
                
                assert result.delivery_id is not None
            finally:
                await async_client.endpoints.delete(str(endpoint.id))
