"""Tests for WAAS Python SDK."""

import pytest
from unittest.mock import Mock, patch, MagicMock
from waas_sdk import WAASClient, AsyncWAASClient
from waas_sdk.exceptions import (
    WAASError,
    WAASAPIError,
    WAASAuthenticationError,
    WAASNotFoundError,
    WAASRateLimitError,
)
from waas_sdk.models import (
    WebhookEndpoint,
    DeliveryAttempt,
    RetryConfiguration,
    DeliveryStatus,
)


class TestWAASClient:
    """Tests for synchronous WAAS client."""

    def test_client_initialization(self):
        """Test client initializes with API key."""
        client = WAASClient(api_key="test-key")
        assert client is not None
        assert client.endpoints is not None
        assert client.deliveries is not None
        assert client.webhooks is not None

    def test_client_requires_api_key(self):
        """Test client raises error without API key."""
        with pytest.raises(ValueError, match="API key is required"):
            WAASClient(api_key="")

    def test_client_custom_base_url(self):
        """Test client accepts custom base URL."""
        client = WAASClient(
            api_key="test-key",
            base_url="https://custom.api.com/v1"
        )
        assert client._base_url == "https://custom.api.com/v1"

    @patch('httpx.Client')
    def test_create_endpoint(self, mock_client_class):
        """Test creating a webhook endpoint."""
        mock_response = MagicMock()
        mock_response.status_code = 201
        mock_response.json.return_value = {
            "id": "123e4567-e89b-12d3-a456-426614174000",
            "tenant_id": "tenant-123",
            "url": "https://example.com/webhook",
            "is_active": True,
            "retry_config": {
                "max_attempts": 5,
                "initial_delay_ms": 1000,
                "max_delay_ms": 300000,
                "backoff_multiplier": 2
            },
            "created_at": "2024-01-01T00:00:00Z",
            "updated_at": "2024-01-01T00:00:00Z",
            "secret": "webhook-secret-123"
        }
        mock_response.raise_for_status = Mock()
        
        mock_client = MagicMock()
        mock_client.post.return_value = mock_response
        mock_client_class.return_value.__enter__ = Mock(return_value=mock_client)
        mock_client_class.return_value.__exit__ = Mock(return_value=False)

        client = WAASClient(api_key="test-key")
        client._client = mock_client

        endpoint = client.endpoints.create(url="https://example.com/webhook")
        
        assert endpoint.id == "123e4567-e89b-12d3-a456-426614174000"
        assert endpoint.url == "https://example.com/webhook"
        assert endpoint.is_active is True
        assert endpoint.secret == "webhook-secret-123"

    @patch('httpx.Client')
    def test_get_delivery(self, mock_client_class):
        """Test getting a delivery attempt."""
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "id": "delivery-123",
            "endpoint_id": "endpoint-456",
            "payload_hash": "abc123",
            "payload_size": 256,
            "status": "success",
            "http_status": 200,
            "response_body": '{"received": true}',
            "attempt_number": 1,
            "scheduled_at": "2024-01-01T00:00:00Z",
            "delivered_at": "2024-01-01T00:00:01Z",
            "created_at": "2024-01-01T00:00:00Z"
        }
        mock_response.raise_for_status = Mock()

        mock_client = MagicMock()
        mock_client.get.return_value = mock_response
        mock_client_class.return_value.__enter__ = Mock(return_value=mock_client)
        mock_client_class.return_value.__exit__ = Mock(return_value=False)

        client = WAASClient(api_key="test-key")
        client._client = mock_client

        delivery = client.deliveries.get("delivery-123")
        
        assert delivery.id == "delivery-123"
        assert delivery.status == DeliveryStatus.SUCCESS
        assert delivery.http_status == 200


class TestErrorHandling:
    """Tests for error handling."""

    def test_authentication_error(self):
        """Test authentication error is raised for 401."""
        error = WAASAuthenticationError()
        assert error.status_code == 401
        assert "authentication" in str(error).lower()

    def test_not_found_error(self):
        """Test not found error is raised for 404."""
        error = WAASNotFoundError("Endpoint not found")
        assert error.status_code == 404
        assert "Endpoint not found" in str(error)

    def test_rate_limit_error(self):
        """Test rate limit error includes retry_after."""
        error = WAASRateLimitError(retry_after=60)
        assert error.status_code == 429
        assert error.retry_after == 60


class TestModels:
    """Tests for data models."""

    def test_webhook_endpoint_from_dict(self):
        """Test WebhookEndpoint.from_dict() creates valid object."""
        data = {
            "id": "123e4567-e89b-12d3-a456-426614174000",
            "tenant_id": "tenant-123",
            "url": "https://example.com/webhook",
            "is_active": True,
            "retry_config": {
                "max_attempts": 5,
                "initial_delay_ms": 1000,
            },
            "created_at": "2024-01-01T00:00:00Z",
            "updated_at": "2024-01-01T00:00:00Z"
        }
        
        endpoint = WebhookEndpoint.model_validate(data)
        
        assert str(endpoint.id) == "123e4567-e89b-12d3-a456-426614174000"
        assert endpoint.url == "https://example.com/webhook"
        assert endpoint.is_active is True
        assert endpoint.retry_config.max_attempts == 5

    def test_retry_configuration_defaults(self):
        """Test RetryConfiguration has sensible defaults."""
        config = RetryConfiguration()
        
        assert config.max_attempts == 5
        assert config.initial_delay_ms == 1000
        assert config.max_delay_ms == 300000
        assert config.backoff_multiplier == 2

    def test_delivery_status_enum(self):
        """Test DeliveryStatus enum values."""
        assert DeliveryStatus.PENDING.value == "pending"
        assert DeliveryStatus.SUCCESS.value == "success"
        assert DeliveryStatus.FAILED.value == "failed"
        assert DeliveryStatus.RETRYING.value == "retrying"


class TestAsyncClient:
    """Tests for async client."""

    def test_async_client_initialization(self):
        """Test async client initializes correctly."""
        client = AsyncWAASClient(api_key="test-key")
        assert client is not None
        assert client.endpoints is not None

    def test_async_client_requires_api_key(self):
        """Test async client raises error without API key."""
        with pytest.raises(ValueError, match="API key is required"):
            AsyncWAASClient(api_key="")


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
