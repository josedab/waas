"""WAAS SDK client implementation."""

from typing import Any, Optional
from uuid import UUID

import httpx

from waas_sdk.exceptions import (
    WAASAPIError,
    WAASAuthenticationError,
    WAASConnectionError,
    WAASNotFoundError,
    WAASRateLimitError,
    WAASTimeoutError,
    WAASValidationError,
)
from waas_sdk.models import (
    AnalyticsSummary,
    BatchSendResponse,
    CreateEndpointRequest,
    CreateTransformationRequest,
    DeliveryAttempt,
    PaginatedResponse,
    QuotaUsage,
    SendWebhookRequest,
    SendWebhookResponse,
    Tenant,
    TestTransformationRequest,
    TestTransformationResponse,
    TestWebhookRequest,
    TestWebhookResponse,
    TimeSeriesDataPoint,
    Transformation,
    UpdateEndpointRequest,
    WebhookEndpoint,
)

DEFAULT_BASE_URL = "https://api.waas-platform.com/api/v1"
DEFAULT_TIMEOUT = 30.0


class BaseClient:
    """Base HTTP client with common functionality."""

    def __init__(
        self,
        api_key: str,
        base_url: str = DEFAULT_BASE_URL,
        timeout: float = DEFAULT_TIMEOUT,
    ) -> None:
        self.api_key = api_key
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout

    def _get_headers(self) -> dict[str, str]:
        return {
            "X-API-Key": self.api_key,
            "Content-Type": "application/json",
            "User-Agent": "waas-sdk-python/1.0.0",
        }

    def _handle_error(self, response: httpx.Response) -> None:
        """Handle HTTP error responses."""
        if response.status_code == 401:
            raise WAASAuthenticationError()
        elif response.status_code == 404:
            raise WAASNotFoundError()
        elif response.status_code == 429:
            retry_after = response.headers.get("Retry-After")
            raise WAASRateLimitError(
                retry_after=int(retry_after) if retry_after else None
            )
        elif response.status_code == 400:
            try:
                data = response.json()
                raise WAASValidationError(
                    message=data.get("message", "Validation error"),
                    details=data.get("details"),
                )
            except ValueError:
                raise WAASValidationError(message=response.text)
        elif response.status_code >= 400:
            try:
                data = response.json()
                raise WAASAPIError(
                    message=data.get("message", "API error"),
                    status_code=response.status_code,
                    code=data.get("code"),
                    details=data.get("details"),
                )
            except ValueError:
                raise WAASAPIError(
                    message=response.text,
                    status_code=response.status_code,
                )


class EndpointsService:
    """Endpoints service for sync client."""

    def __init__(self, client: "WAASClient") -> None:
        self._client = client

    def list(
        self, page: int = 1, per_page: int = 20
    ) -> PaginatedResponse:
        """List all webhook endpoints."""
        response = self._client._request(
            "GET",
            "/webhooks/endpoints",
            params={"page": page, "per_page": per_page},
        )
        data = response.json()
        return PaginatedResponse(
            data=[WebhookEndpoint(**ep) for ep in data.get("data", [])],
            total=data.get("total", 0),
            page=data.get("page", page),
            per_page=data.get("per_page", per_page),
            total_pages=data.get("total_pages", 1),
        )

    def get(self, endpoint_id: UUID) -> WebhookEndpoint:
        """Get a webhook endpoint by ID."""
        response = self._client._request(
            "GET", f"/webhooks/endpoints/{endpoint_id}"
        )
        return WebhookEndpoint(**response.json())

    def create(
        self,
        url: str,
        custom_headers: Optional[dict[str, str]] = None,
        retry_config: Optional[dict[str, Any]] = None,
    ) -> WebhookEndpoint:
        """Create a new webhook endpoint."""
        request = CreateEndpointRequest(
            url=url,
            custom_headers=custom_headers or {},
            retry_config=retry_config,
        )
        response = self._client._request(
            "POST",
            "/webhooks/endpoints",
            json=request.model_dump(mode="json", exclude_none=True),
        )
        return WebhookEndpoint(**response.json())

    def update(
        self,
        endpoint_id: UUID,
        url: Optional[str] = None,
        is_active: Optional[bool] = None,
        custom_headers: Optional[dict[str, str]] = None,
        retry_config: Optional[dict[str, Any]] = None,
    ) -> WebhookEndpoint:
        """Update a webhook endpoint."""
        request = UpdateEndpointRequest(
            url=url,
            is_active=is_active,
            custom_headers=custom_headers,
            retry_config=retry_config,
        )
        response = self._client._request(
            "PATCH",
            f"/webhooks/endpoints/{endpoint_id}",
            json=request.model_dump(mode="json", exclude_none=True),
        )
        return WebhookEndpoint(**response.json())

    def delete(self, endpoint_id: UUID) -> None:
        """Delete a webhook endpoint."""
        self._client._request("DELETE", f"/webhooks/endpoints/{endpoint_id}")


class DeliveriesService:
    """Deliveries service for sync client."""

    def __init__(self, client: "WAASClient") -> None:
        self._client = client

    def list(
        self,
        page: int = 1,
        per_page: int = 20,
        status: Optional[str] = None,
        endpoint_id: Optional[UUID] = None,
    ) -> PaginatedResponse:
        """List delivery attempts."""
        params: dict[str, Any] = {"page": page, "per_page": per_page}
        if status:
            params["status"] = status
        if endpoint_id:
            params["endpoint_id"] = str(endpoint_id)

        response = self._client._request(
            "GET", "/webhooks/deliveries", params=params
        )
        data = response.json()
        return PaginatedResponse(
            data=[DeliveryAttempt(**d) for d in data.get("data", [])],
            total=data.get("total", 0),
            page=data.get("page", page),
            per_page=data.get("per_page", per_page),
            total_pages=data.get("total_pages", 1),
        )

    def get(self, delivery_id: UUID) -> DeliveryAttempt:
        """Get a delivery attempt by ID."""
        response = self._client._request(
            "GET", f"/webhooks/deliveries/{delivery_id}"
        )
        return DeliveryAttempt(**response.json())

    def retry(self, delivery_id: UUID) -> SendWebhookResponse:
        """Retry a failed delivery."""
        response = self._client._request(
            "POST", f"/webhooks/deliveries/{delivery_id}/retry"
        )
        return SendWebhookResponse(**response.json())


class WebhooksService:
    """Webhooks service for sync client."""

    def __init__(self, client: "WAASClient") -> None:
        self._client = client

    def send(
        self,
        payload: Any,
        endpoint_id: Optional[UUID] = None,
        headers: Optional[dict[str, str]] = None,
    ) -> SendWebhookResponse:
        """Send a webhook."""
        request = SendWebhookRequest(
            endpoint_id=endpoint_id,
            payload=payload,
            headers=headers or {},
        )
        response = self._client._request(
            "POST",
            "/webhooks/send",
            json=request.model_dump(mode="json", exclude_none=True),
        )
        return SendWebhookResponse(**response.json())

    def send_batch(
        self,
        payload: Any,
        endpoint_ids: Optional[list[UUID]] = None,
        headers: Optional[dict[str, str]] = None,
    ) -> BatchSendResponse:
        """Send webhooks to multiple endpoints."""
        data: dict[str, Any] = {"payload": payload}
        if endpoint_ids:
            data["endpoint_ids"] = [str(eid) for eid in endpoint_ids]
        if headers:
            data["headers"] = headers

        response = self._client._request("POST", "/webhooks/send/batch", json=data)
        return BatchSendResponse(**response.json())


class AnalyticsService:
    """Analytics service for sync client."""

    def __init__(self, client: "WAASClient") -> None:
        self._client = client

    def get_summary(
        self,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None,
    ) -> AnalyticsSummary:
        """Get analytics summary."""
        params: dict[str, str] = {}
        if start_date:
            params["start_date"] = start_date
        if end_date:
            params["end_date"] = end_date

        response = self._client._request(
            "GET", "/analytics/summary", params=params
        )
        return AnalyticsSummary(**response.json())

    def get_quota_usage(self) -> QuotaUsage:
        """Get current quota usage."""
        response = self._client._request("GET", "/analytics/quota")
        return QuotaUsage(**response.json())

    def get_delivery_timeseries(
        self,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None,
        interval: str = "day",
    ) -> list[TimeSeriesDataPoint]:
        """Get delivery time series data."""
        params: dict[str, str] = {"interval": interval}
        if start_date:
            params["start_date"] = start_date
        if end_date:
            params["end_date"] = end_date

        response = self._client._request(
            "GET", "/analytics/deliveries/timeseries", params=params
        )
        return [TimeSeriesDataPoint(**dp) for dp in response.json()]


class TransformationsService:
    """Transformations service for sync client."""

    def __init__(self, client: "WAASClient") -> None:
        self._client = client

    def list(self, page: int = 1, per_page: int = 20) -> list[Transformation]:
        """List transformations."""
        response = self._client._request(
            "GET",
            "/transformations",
            params={"page": page, "per_page": per_page},
        )
        return [Transformation(**t) for t in response.json()]

    def get(self, transformation_id: UUID) -> Transformation:
        """Get a transformation by ID."""
        response = self._client._request(
            "GET", f"/transformations/{transformation_id}"
        )
        return Transformation(**response.json())

    def create(
        self,
        name: str,
        script: str,
        description: Optional[str] = None,
        enabled: bool = True,
    ) -> Transformation:
        """Create a new transformation."""
        request = CreateTransformationRequest(
            name=name,
            script=script,
            description=description,
            enabled=enabled,
        )
        response = self._client._request(
            "POST",
            "/transformations",
            json=request.model_dump(mode="json", exclude_none=True),
        )
        return Transformation(**response.json())

    def delete(self, transformation_id: UUID) -> None:
        """Delete a transformation."""
        self._client._request("DELETE", f"/transformations/{transformation_id}")

    def test(
        self, script: str, input_payload: Any
    ) -> TestTransformationResponse:
        """Test a transformation script."""
        request = TestTransformationRequest(
            script=script, input_payload=input_payload
        )
        response = self._client._request(
            "POST",
            "/transformations/test",
            json=request.model_dump(mode="json"),
        )
        return TestTransformationResponse(**response.json())


class TestingService:
    """Testing service for sync client."""

    def __init__(self, client: "WAASClient") -> None:
        self._client = client

    def test_webhook(
        self,
        url: str,
        payload: Any,
        headers: Optional[dict[str, str]] = None,
        method: str = "POST",
        timeout: int = 30,
    ) -> TestWebhookResponse:
        """Test a webhook delivery."""
        request = TestWebhookRequest(
            url=url,
            payload=payload,
            headers=headers or {},
            method=method,
            timeout=timeout,
        )
        response = self._client._request(
            "POST",
            "/testing/webhook",
            json=request.model_dump(mode="json"),
        )
        return TestWebhookResponse(**response.json())


class TenantService:
    """Tenant service for sync client."""

    def __init__(self, client: "WAASClient") -> None:
        self._client = client

    def get_current(self) -> Tenant:
        """Get current tenant information."""
        response = self._client._request("GET", "/tenants/me")
        return Tenant(**response.json())


class WAASClient:
    """Synchronous WAAS API client."""

    def __init__(
        self,
        api_key: str,
        base_url: str = DEFAULT_BASE_URL,
        timeout: float = DEFAULT_TIMEOUT,
    ) -> None:
        self._base = BaseClient(api_key, base_url, timeout)
        self._http = httpx.Client(
            base_url=self._base.base_url,
            headers=self._base._get_headers(),
            timeout=timeout,
        )

        # Initialize services
        self.endpoints = EndpointsService(self)
        self.deliveries = DeliveriesService(self)
        self.webhooks = WebhooksService(self)
        self.analytics = AnalyticsService(self)
        self.transformations = TransformationsService(self)
        self.testing = TestingService(self)
        self.tenant = TenantService(self)

    def _request(
        self,
        method: str,
        path: str,
        params: Optional[dict[str, Any]] = None,
        json: Optional[dict[str, Any]] = None,
    ) -> httpx.Response:
        """Make an HTTP request."""
        try:
            response = self._http.request(
                method, path, params=params, json=json
            )
            if response.status_code >= 400:
                self._base._handle_error(response)
            return response
        except httpx.ConnectError as e:
            raise WAASConnectionError(f"Connection failed: {e}")
        except httpx.TimeoutException as e:
            raise WAASTimeoutError(f"Request timeout: {e}")

    def close(self) -> None:
        """Close the HTTP client."""
        self._http.close()

    def __enter__(self) -> "WAASClient":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()


class AsyncWAASClient:
    """Asynchronous WAAS API client."""

    def __init__(
        self,
        api_key: str,
        base_url: str = DEFAULT_BASE_URL,
        timeout: float = DEFAULT_TIMEOUT,
    ) -> None:
        self._base = BaseClient(api_key, base_url, timeout)
        self._http = httpx.AsyncClient(
            base_url=self._base.base_url,
            headers=self._base._get_headers(),
            timeout=timeout,
        )

        # Initialize async services
        self.endpoints = AsyncEndpointsService(self)
        self.deliveries = AsyncDeliveriesService(self)
        self.webhooks = AsyncWebhooksService(self)
        self.analytics = AsyncAnalyticsService(self)
        self.transformations = AsyncTransformationsService(self)
        self.testing = AsyncTestingService(self)
        self.tenant = AsyncTenantService(self)

    async def _request(
        self,
        method: str,
        path: str,
        params: Optional[dict[str, Any]] = None,
        json: Optional[dict[str, Any]] = None,
    ) -> httpx.Response:
        """Make an async HTTP request."""
        try:
            response = await self._http.request(
                method, path, params=params, json=json
            )
            if response.status_code >= 400:
                self._base._handle_error(response)
            return response
        except httpx.ConnectError as e:
            raise WAASConnectionError(f"Connection failed: {e}")
        except httpx.TimeoutException as e:
            raise WAASTimeoutError(f"Request timeout: {e}")

    async def close(self) -> None:
        """Close the HTTP client."""
        await self._http.aclose()

    async def __aenter__(self) -> "AsyncWAASClient":
        return self

    async def __aexit__(self, *args: Any) -> None:
        await self.close()


# Async service implementations
class AsyncEndpointsService:
    def __init__(self, client: AsyncWAASClient) -> None:
        self._client = client

    async def list(self, page: int = 1, per_page: int = 20) -> PaginatedResponse:
        response = await self._client._request(
            "GET", "/webhooks/endpoints", params={"page": page, "per_page": per_page}
        )
        data = response.json()
        return PaginatedResponse(
            data=[WebhookEndpoint(**ep) for ep in data.get("data", [])],
            total=data.get("total", 0),
            page=data.get("page", page),
            per_page=data.get("per_page", per_page),
            total_pages=data.get("total_pages", 1),
        )

    async def get(self, endpoint_id: UUID) -> WebhookEndpoint:
        response = await self._client._request("GET", f"/webhooks/endpoints/{endpoint_id}")
        return WebhookEndpoint(**response.json())

    async def create(
        self,
        url: str,
        custom_headers: Optional[dict[str, str]] = None,
        retry_config: Optional[dict[str, Any]] = None,
    ) -> WebhookEndpoint:
        request = CreateEndpointRequest(
            url=url, custom_headers=custom_headers or {}, retry_config=retry_config
        )
        response = await self._client._request(
            "POST", "/webhooks/endpoints", json=request.model_dump(mode="json", exclude_none=True)
        )
        return WebhookEndpoint(**response.json())

    async def delete(self, endpoint_id: UUID) -> None:
        await self._client._request("DELETE", f"/webhooks/endpoints/{endpoint_id}")


class AsyncDeliveriesService:
    def __init__(self, client: AsyncWAASClient) -> None:
        self._client = client

    async def list(
        self, page: int = 1, per_page: int = 20, status: Optional[str] = None
    ) -> PaginatedResponse:
        params: dict[str, Any] = {"page": page, "per_page": per_page}
        if status:
            params["status"] = status
        response = await self._client._request("GET", "/webhooks/deliveries", params=params)
        data = response.json()
        return PaginatedResponse(
            data=[DeliveryAttempt(**d) for d in data.get("data", [])],
            total=data.get("total", 0),
            page=data.get("page", page),
            per_page=data.get("per_page", per_page),
            total_pages=data.get("total_pages", 1),
        )

    async def get(self, delivery_id: UUID) -> DeliveryAttempt:
        response = await self._client._request("GET", f"/webhooks/deliveries/{delivery_id}")
        return DeliveryAttempt(**response.json())


class AsyncWebhooksService:
    def __init__(self, client: AsyncWAASClient) -> None:
        self._client = client

    async def send(
        self,
        payload: Any,
        endpoint_id: Optional[UUID] = None,
        headers: Optional[dict[str, str]] = None,
    ) -> SendWebhookResponse:
        request = SendWebhookRequest(endpoint_id=endpoint_id, payload=payload, headers=headers or {})
        response = await self._client._request(
            "POST", "/webhooks/send", json=request.model_dump(mode="json", exclude_none=True)
        )
        return SendWebhookResponse(**response.json())


class AsyncAnalyticsService:
    def __init__(self, client: AsyncWAASClient) -> None:
        self._client = client

    async def get_summary(
        self, start_date: Optional[str] = None, end_date: Optional[str] = None
    ) -> AnalyticsSummary:
        params: dict[str, str] = {}
        if start_date:
            params["start_date"] = start_date
        if end_date:
            params["end_date"] = end_date
        response = await self._client._request("GET", "/analytics/summary", params=params)
        return AnalyticsSummary(**response.json())


class AsyncTransformationsService:
    def __init__(self, client: AsyncWAASClient) -> None:
        self._client = client

    async def list(self, page: int = 1, per_page: int = 20) -> list[Transformation]:
        response = await self._client._request(
            "GET", "/transformations", params={"page": page, "per_page": per_page}
        )
        return [Transformation(**t) for t in response.json()]

    async def test(self, script: str, input_payload: Any) -> TestTransformationResponse:
        request = TestTransformationRequest(script=script, input_payload=input_payload)
        response = await self._client._request(
            "POST", "/transformations/test", json=request.model_dump(mode="json")
        )
        return TestTransformationResponse(**response.json())


class AsyncTestingService:
    def __init__(self, client: AsyncWAASClient) -> None:
        self._client = client

    async def test_webhook(
        self,
        url: str,
        payload: Any,
        headers: Optional[dict[str, str]] = None,
    ) -> TestWebhookResponse:
        request = TestWebhookRequest(url=url, payload=payload, headers=headers or {})
        response = await self._client._request(
            "POST", "/testing/webhook", json=request.model_dump(mode="json")
        )
        return TestWebhookResponse(**response.json())


class AsyncTenantService:
    def __init__(self, client: AsyncWAASClient) -> None:
        self._client = client

    async def get_current(self) -> Tenant:
        response = await self._client._request("GET", "/tenants/me")
        return Tenant(**response.json())
