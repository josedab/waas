# WAAS Python SDK

Official Python SDK for the WAAS (Webhook-as-a-Service) Platform.

## Installation

```bash
pip install waas-sdk
```

## Quick Start

```python
from waas_sdk import WAASClient

# Initialize client
client = WAASClient(api_key="your-api-key")

# Create a webhook endpoint
endpoint = client.endpoints.create(
    url="https://your-server.com/webhook",
    retry_config={
        "max_attempts": 5,
        "initial_delay_ms": 1000
    }
)
print(f"Created endpoint: {endpoint.id}")

# Send a webhook
delivery = client.webhooks.send(
    endpoint_id=endpoint.id,
    payload={"event": "user.created", "data": {"id": 123}}
)
print(f"Delivery scheduled: {delivery.delivery_id}")

# Check delivery status
status = client.deliveries.get(delivery.delivery_id)
print(f"Status: {status.status}")
```

## Async Support

```python
import asyncio
from waas_sdk import AsyncWAASClient

async def main():
    async with AsyncWAASClient(api_key="your-api-key") as client:
        # All methods are async
        endpoints = await client.endpoints.list()
        for ep in endpoints.data:
            print(f"Endpoint: {ep.url}")

asyncio.run(main())
```

## Features

- Full API coverage for endpoints, deliveries, analytics, and testing
- Both sync and async clients
- Type hints with Pydantic models
- Automatic retries with exponential backoff
- Comprehensive error handling

## Documentation

For full documentation, visit [docs.waas-platform.com/sdks/python](https://docs.waas-platform.com/sdks/python)

## License

MIT License
