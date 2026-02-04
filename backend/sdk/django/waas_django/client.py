"""WaaS API Client for Django"""

import httpx
from django.conf import settings


class WaaSClient:
    """HTTP client for the WaaS API."""

    def __init__(self, api_key: str | None = None, api_url: str | None = None):
        self.api_key = api_key or getattr(settings, 'WAAS_API_KEY', '')
        self.api_url = api_url or getattr(settings, 'WAAS_API_URL', 'http://localhost:8080')
        self._client = httpx.Client(
            base_url=self.api_url,
            headers={'X-API-Key': self.api_key, 'Content-Type': 'application/json'},
            timeout=30.0,
        )

    def send_webhook(self, event_type: str, payload: dict, endpoint_ids: list[str] | None = None) -> dict:
        """Send a webhook event through WaaS."""
        body = {'event_type': event_type, 'payload': payload}
        if endpoint_ids:
            body['endpoint_ids'] = endpoint_ids
        response = self._client.post('/api/v1/webhooks/send', json=body)
        response.raise_for_status()
        return response.json()

    def create_endpoint(self, url: str, event_types: list[str] | None = None, **kwargs) -> dict:
        """Register a webhook endpoint."""
        body = {'url': url, **kwargs}
        if event_types:
            body['event_types'] = event_types
        response = self._client.post('/api/v1/endpoints', json=body)
        response.raise_for_status()
        return response.json()

    def list_deliveries(self, limit: int = 20, offset: int = 0) -> dict:
        """List delivery attempts."""
        response = self._client.get('/api/v1/webhooks/deliveries', params={'limit': limit, 'offset': offset})
        response.raise_for_status()
        return response.json()

    def health(self) -> dict:
        """Check WaaS service health."""
        response = self._client.get('/health')
        response.raise_for_status()
        return response.json()

    def close(self):
        self._client.close()

    def __enter__(self):
        return self

    def __exit__(self, *args):
        self.close()
