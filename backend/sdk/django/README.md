# WaaS Django Integration

Drop-in webhook integration for Django applications.

## Installation

Since WaaS is self-hosted, install from the local SDK directory:

```bash
pip install /path/to/waas/backend/sdk/django
```

Or add to your `requirements.txt`:

```
waas-django @ file:///path/to/waas/backend/sdk/django
```

### Requirements

- Python ≥ 3.9
- Django ≥ 4.2

## Quick Start

1. Add `waas_django` to your `INSTALLED_APPS`:

```python
INSTALLED_APPS = [
    # ...
    "waas_django",
]
```

2. Configure your WaaS connection in `settings.py`:

```python
WAAS_API_URL = "http://localhost:8080"
WAAS_API_KEY = "your-api-key"
```

3. Send a webhook from any view or task:

```python
from waas_django import waas

waas.send(
    endpoint_id="your-endpoint-id",
    payload={"event": "order.created", "data": {"id": 123}},
)
```

## Documentation

For detailed API documentation, see the [API docs](../../docs/README.md).

## License

MIT — see [LICENSE](../../LICENSE) for details.
