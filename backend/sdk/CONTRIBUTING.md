# WAAS SDK Development Guide

This document provides guidelines for developing and maintaining WAAS SDKs.

## SDK Architecture

All WAAS SDKs follow a consistent architecture:

```
sdk/{language}/
├── README.md           # Quick start and installation
├── {package-file}      # Language-specific package manifest
├── src/
│   ├── client.{ext}    # Main client class
│   ├── models/         # API models (endpoints, deliveries, etc.)
│   ├── services/       # Service classes (endpoints, deliveries, webhooks)
│   └── exceptions/     # Error types
└── tests/              # Unit and integration tests
```

## Design Principles

### 1. Consistency Across Languages

All SDKs should provide the same functionality and follow similar patterns:

```python
# Python
client = WAASClient(api_key="...")
endpoint = client.endpoints.create(url="...")
```

```javascript
// Node.js
const client = new WAASClient({ apiKey: "..." });
const endpoint = await client.endpoints.create({ url: "..." });
```

```java
// Java
WAASClient client = WAASClient.builder().apiKey("...").build();
WebhookEndpoint endpoint = client.endpoints().create(CreateEndpointRequest.builder().url("...").build());
```

### 2. Idiomatic Code

Each SDK should follow the conventions of its language:
- **Python**: snake_case, type hints, async/await support
- **Node.js**: camelCase, Promises, TypeScript types
- **Java**: Builder pattern, checked exceptions, Optional
- **Ruby**: snake_case, blocks, duck typing
- **PHP**: PSR-4 autoloading, typed properties

### 3. Error Handling

All SDKs implement a consistent exception hierarchy:

```
WAASException (base)
├── WAASAPIException (API errors)
│   ├── AuthenticationException (401)
│   ├── NotFoundException (404)
│   ├── ValidationException (422)
│   └── RateLimitException (429)
└── WAASNetworkException (connection errors)
```

### 4. Configuration

Support multiple configuration methods:

```python
# Direct initialization
client = WAASClient(api_key="...", base_url="...")

# Environment variables
client = WAASClient()  # Reads WAAS_API_KEY

# Configuration object
config = WAASConfig(api_key="...", timeout=60)
client = WAASClient(config)
```

## API Coverage

### Required Services

All SDKs must implement these services:

| Service | Methods |
|---------|---------|
| **Endpoints** | create, get, list, update, delete, rotateSecret |
| **Deliveries** | get, list, retry |
| **Webhooks** | send |
| **Analytics** | getMetrics, getUsage |
| **Transformations** | create, get, list, update, delete, test |

### Models

Required model classes:

- `WebhookEndpoint`
- `DeliveryAttempt`
- `RetryConfiguration`
- `SendWebhookRequest/Response`
- `AnalyticsMetrics`
- `Transformation`

## Testing Guidelines

### Unit Tests

Cover all service methods and models:

```python
def test_create_endpoint():
    client = WAASClient(api_key="test")
    with mock_server() as server:
        server.expect_post("/webhooks/endpoints", return_json={...})
        endpoint = client.endpoints.create(url="https://example.com")
        assert endpoint.id == "..."
```

### Integration Tests

Optional but recommended, run against a test server:

```python
@pytest.mark.integration
def test_full_workflow():
    client = WAASClient(api_key=os.environ["WAAS_API_KEY"])
    endpoint = client.endpoints.create(url="https://httpbin.org/post")
    delivery = client.webhooks.send(endpoint_id=endpoint.id, payload={"test": True})
    assert delivery.status == "pending"
```

## Release Process

1. Update version in package manifest
2. Update CHANGELOG.md
3. Run full test suite
4. Tag release: `v{version}`
5. Publish to package registry

### Version Numbering

Follow semantic versioning:
- **Major**: Breaking API changes
- **Minor**: New features, backward compatible
- **Patch**: Bug fixes, documentation

## Contributing New SDKs

To contribute a new SDK:

1. **Fork and clone** the repository
2. Create `sdk/{language}/` directory
3. Implement core services following existing patterns
4. Add comprehensive tests
5. Write README with quick start guide
6. Submit pull request

### Checklist for New SDKs

- [ ] Package manifest (pyproject.toml, package.json, etc.)
- [ ] README with installation and quick start
- [ ] Client class with configuration options
- [ ] All required services implemented
- [ ] Exception hierarchy
- [ ] Unit tests with >80% coverage
- [ ] CI pipeline configuration

## Current SDK Status

| Language | Status | Package |
|----------|--------|---------|
| Go | ✅ Official | `github.com/waas-platform/sdk-go` |
| Python | ✅ Official | `waas-sdk` (PyPI) |
| Node.js | ✅ Official | `@waas/sdk` (npm) |
| Java | ✅ Official | `com.waas:waas-sdk` (Maven) |
| Ruby | 🔨 Community | `waas-sdk` (RubyGems) |
| PHP | 🔨 Community | `waas/waas-sdk` (Packagist) |

## Getting Help

- **Issues**: Open a GitHub issue for bugs or feature requests
- **Discussions**: Use GitHub Discussions for questions
- **Slack**: Join `#sdk-dev` channel for real-time help
