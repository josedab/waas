# Webhook Service Platform API Documentation

This directory contains the complete API documentation for the Webhook Service Platform, including OpenAPI specifications, interactive documentation, and SDK resources.

## 📚 Documentation Overview

### API Documentation
- **OpenAPI Specification**: `swagger.json` and `swagger.yaml`
- **Interactive Documentation**: Available via Swagger UI at `/docs/`
- **Generated Documentation**: Auto-generated from Go code annotations

### SDK Documentation
- **Go SDK**: Complete SDK with examples in `../sdk/go/`
- **Code Examples**: Practical usage examples for common scenarios
- **API Reference**: Comprehensive reference documentation
- **Testing Tools**: Webhook testing and debugging utilities — see [testing_tools.md](testing_tools.md)

## 🚀 Quick Start

### Viewing the Documentation

1. **Start the API server:**
   ```bash
   make run-api
   ```

2. **Access the interactive documentation:**
   - Swagger UI: http://localhost:8080/docs/
   - JSON spec: http://localhost:8080/docs/swagger.json
   - YAML spec: http://localhost:8080/docs/swagger.yaml

### Generating Documentation

```bash
# Generate all documentation
make docs-generate

# Generate only OpenAPI specs
make docs

# Run documentation tests
make docs-test
```

## 📖 API Reference

### Authentication
All API endpoints (except tenant creation) require authentication using an API key:

```http
Authorization: Bearer your-api-key-here
```

### Base URL
```
http://localhost:8080/api/v1
```

> Replace with your deployment URL in production (e.g., `https://webhooks.example.com/api/v1`).

### Core Endpoints

#### Tenant Management
- `POST /tenants` - Create a new tenant account
- `GET /tenant` - Get current tenant information
- `PUT /tenant` - Update tenant settings
- `POST /tenant/regenerate-key` - Generate new API key

#### Webhook Endpoints
- `POST /webhooks/endpoints` - Create webhook endpoint
- `GET /webhooks/endpoints` - List webhook endpoints
- `GET /webhooks/endpoints/{id}` - Get endpoint details
- `PUT /webhooks/endpoints/{id}` - Update endpoint
- `DELETE /webhooks/endpoints/{id}` - Delete endpoint

#### Webhook Sending
- `POST /webhooks/send` - Send webhook to endpoint(s)
- `POST /webhooks/send/batch` - Batch send webhooks

#### Monitoring & Analytics
- `GET /webhooks/deliveries` - Get delivery history
- `GET /webhooks/deliveries/{id}` - Get delivery details

#### Testing & Debugging
- `POST /webhooks/test` - Test webhook endpoint
- `POST /webhooks/test/endpoints` - Create test endpoint
- `GET /webhooks/deliveries/{id}/inspect` - Inspect delivery

## 🛠 SDK Usage

### Enterprise & Extended Endpoints

Beyond the core endpoints above, the platform registers the following feature-specific route groups under `/api/v1` (all require authentication):

#### Core Feature Routes

| Prefix | Package | Description |
|--------|---------|-------------|
| `/schemas` | `pkg/schema` | Event schema registry: create, validate, assign to endpoints |
| `/flows` | `pkg/flow` | Visual flow builder: event routing workflows |
| `/meta-events/subscriptions` | `pkg/metaevents` | Webhooks-about-webhooks: subscribe to platform events |
| `/geo` | `pkg/georouting` | Geographic routing: region-based delivery rules |
| `/embed/tokens` | `pkg/embed` | Embeddable analytics SDK: token management |
| `/mocks` | `pkg/mocking` | Webhook mocking service: simulated endpoints |
| `/costs` | `pkg/costing` | Cost attribution dashboard |
| `/otel` | `pkg/otel` | OpenTelemetry integration: traces and metrics export |
| `/protocols` | `pkg/protocols` | Custom webhook protocols: HTTP, gRPC, MQTT, WebSocket |

#### Enterprise Platform Routes

| Prefix | Package | Description |
|--------|---------|-------------|
| `/traces` | `pkg/observability` | Observability suite: distributed traces and dashboards |
| `/rate-limits` | `pkg/smartlimit` | Intelligent adaptive rate limiting |
| `/chaos` | `pkg/chaos` | Chaos engineering: fault injection testing |
| `/cdc` | `pkg/cdc` | Change data capture: database change notifications |
| `/workflows` | `pkg/workflow` | Visual workflow builder v2 |
| `/signatures` | `pkg/signatures` | Webhook signature standardization |
| `/push` | `pkg/pushbridge` | Mobile SDK & push notification bridge |
| `/billing` | `pkg/billing` | Real-time billing alerts and Stripe integration |
| `/versions` | `pkg/versioning` | API versioning & deprecation management |
| `/federation` | `pkg/federation` | Federated webhook network |

#### Enterprise Advanced Routes

| Prefix | Package | Description |
|--------|---------|-------------|
| `/sla` | `pkg/sla` | SLA dashboard & alerting |
| `/mtls` | `pkg/mtls` | Mutual TLS certificate management |
| `/contracts` | `pkg/contracts` | Webhook contract testing |
| `/marketplace` | `pkg/marketplacetpl` | Webhook marketplace & templates |
| `/eventmesh` | `pkg/eventmesh` | Event mesh routing engine |
| `/debugger` | `pkg/debugger` | Webhook debugger & time-travel replay |
| `/cloud` | `pkg/cloud`, `pkg/cloudctl` | Cloud control plane, billing, teams, audit |
| `/terraform` | `pkg/tfprovider` | Terraform provider API |
| `/portal` | `pkg/portal` | Embeddable webhook portal |

#### Delivery, Edge & Observability Routes

| Prefix | Package | Description |
|--------|---------|-------------|
| `/streaming` | `pkg/streaming` | Event streaming bridge |
| `/remediation` | `pkg/remediation` | AI auto-remediation |
| `/edge` | `pkg/edge` | Edge function runtime |
| `/blockchain` | `pkg/blockchain` | Smart contract triggers & immutable audit trail |
| `/compliance` | `pkg/compliancecenter` | Compliance center: GDPR, HIPAA, SOC2 |
| `/prediction` | `pkg/prediction` | Predictive failure prevention |
| `/graphql` | `pkg/graphqlsub` | GraphQL subscriptions gateway |
| `/monetization` | `pkg/monetization` | Webhook monetization platform |
| `/canary` | `pkg/canary` | Canary deployments |
| `/catalog` | `pkg/catalog` | Event catalog & schema registry |
| `/sandbox` | `pkg/sandbox` | Webhook replay sandbox |
| `/analytics/embed` | `pkg/analyticsembed` | Embeddable analytics widgets |
| `/gitops` | `pkg/gitops` | GitOps configuration management |
| `/migrations` | `pkg/livemigration` | Live migration toolkit |

#### Inbound, Fan-out & Ecosystem Routes

| Prefix | Package | Description |
|--------|---------|-------------|
| `/inbound` | `pkg/inbound` | Inbound webhook gateway |
| `/fanout` | `pkg/fanout` | Fan-out & topic-based routing |
| `/mobile` | `pkg/mobilesdk` | Mobile SDK management |
| `/marketplace/plugins` | `pkg/pluginmarket` | Plugin marketplace |
| `/intelligence` | `pkg/intelligence` | AI-powered webhook intelligence |
| `/flow-builder` | `pkg/flowbuilder` | Visual webhook workflow builder |
| `/timetravel` | `pkg/timetravel` | Webhook replay & time travel |
| `/callbacks` | `pkg/callback` | Bi-directional webhooks & callbacks |
| `/collab-debug` | `pkg/collabdebug` | Real-time collaborative debugging |
| `/waf` | `pkg/waf` | Webhook security scanner & WAF |
| `/docgen` | `pkg/docgen` | API-first documentation generator |
| `/whitelabel` | `pkg/whitelabel` | Multi-tenant white-label support |
| `/dlq` | `pkg/dlq` | Dead letter queue & observability |
| `/openapi` | `pkg/openapigen` | OpenAPI-to-webhook generator |
| `/playground` | `pkg/playground` | Interactive webhook playground |
| `/pipelines` | `pkg/pipeline` | Delivery pipeline composition |

#### Admin Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/admin/tenants` | List all tenants |
| `PUT` | `/api/v1/admin/tenants/:tenant_id` | Update tenant (admin) |
| `GET` | `/api/v1/admin/alerts/active` | Get active alerts |
| `GET` | `/api/v1/admin/alerts/history` | Get alert history |

### Go SDK Installation

The Go SDK is bundled in the repository under `sdk/go/`. See the
[Go SDK README](../sdk/go/README.md) for installation options (copy,
Go workspace, or replace directive). For example:

```
require github.com/josedab/waas/sdk/go v0.0.0
replace github.com/josedab/waas/sdk/go => /path/to/waas/backend/sdk/go
```

### Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/josedab/waas/sdk/go/client"
)

func main() {
    // Initialize client
    c := client.New("your-api-key")
    
    // Create webhook endpoint
    endpoint, err := c.Webhooks.CreateEndpoint(context.Background(), &client.CreateEndpointRequest{
        URL: "https://your-app.com/webhooks",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Send webhook
    delivery, err := c.Webhooks.Send(context.Background(), &client.SendWebhookRequest{
        EndpointID: &endpoint.ID,
        Payload:    map[string]interface{}{"message": "Hello, World!"},
    })
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Webhook sent: %s", delivery.DeliveryID)
}
```

## 📋 Request/Response Examples

### Create Webhook Endpoint

**Request:**
```http
POST /api/v1/webhooks/endpoints
Content-Type: application/json
Authorization: Bearer your-api-key

{
  "url": "https://your-app.com/webhooks",
  "custom_headers": {
    "Authorization": "Bearer token"
  },
  "retry_config": {
    "max_attempts": 5,
    "initial_delay_ms": 1000,
    "max_delay_ms": 300000,
    "backoff_multiplier": 2
  }
}
```

**Response:**
```http
HTTP/1.1 201 Created
Content-Type: application/json

{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "url": "https://your-app.com/webhooks",
  "secret": "wh_secret_abc123...",
  "is_active": true,
  "retry_config": {
    "max_attempts": 5,
    "initial_delay_ms": 1000,
    "max_delay_ms": 300000,
    "backoff_multiplier": 2
  },
  "custom_headers": {
    "Authorization": "Bearer token"
  },
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Send Webhook

**Request:**
```http
POST /api/v1/webhooks/send
Content-Type: application/json
Authorization: Bearer your-api-key

{
  "endpoint_id": "550e8400-e29b-41d4-a716-446655440000",
  "payload": {
    "event": "user.created",
    "data": {
      "user_id": "12345",
      "email": "user@example.com"
    }
  },
  "headers": {
    "X-Event-Type": "user.created"
  }
}
```

**Response:**
```http
HTTP/1.1 202 Accepted
Content-Type: application/json

{
  "delivery_id": "660e8400-e29b-41d4-a716-446655440000",
  "endpoint_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "scheduled_at": "2024-01-01T00:00:00Z"
}
```

## ❌ Error Handling

All API errors follow a consistent format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      "additional": "context"
    }
  }
}
```

### Common Error Codes

| Code | Status | Description |
|------|--------|-------------|
| `INVALID_REQUEST` | 400 | Request format or validation error |
| `UNAUTHORIZED` | 401 | Invalid or missing API key |
| `FORBIDDEN` | 403 | Access denied to resource |
| `NOT_FOUND` | 404 | Resource not found |
| `RATE_LIMITED` | 429 | Rate limit exceeded |
| `PAYLOAD_TOO_LARGE` | 400 | Webhook payload exceeds size limit |
| `ENDPOINT_INACTIVE` | 400 | Webhook endpoint is disabled |
| `INTERNAL_ERROR` | 500 | Server error |

## 🔧 Development

### API Versioning

The API uses URL-based versioning with the prefix `/api/v1`. All current endpoints are served under this version.

**Versioning strategy:**
- **URL prefix**: All endpoints are prefixed with `/api/v1/` (e.g., `/api/v1/webhooks/endpoints`)
- **New versions**: Breaking changes will be introduced under a new version prefix (e.g., `/api/v2/`)
- **Parallel availability**: When a new version is released, the previous version will continue to be served alongside it

**Backward compatibility guarantees:**
- Additive changes (new fields, new endpoints) within the same version are non-breaking
- Existing response fields will not be removed or have their types changed within a version
- New optional request parameters may be added without a version bump
- Required request parameters will not be added to existing endpoints within a version

**Deprecation policy:**
- Deprecated endpoints will be marked with a `Deprecated` header in responses
- Deprecation notices will appear in the Swagger documentation and CHANGELOG
- Deprecated versions will remain available for a minimum of 6 months after the successor version is released
- Clients should monitor the `Sunset` response header for the planned removal date

**Example deprecation headers:**
```http
Deprecation: true
Sunset: Sat, 01 Mar 2026 00:00:00 GMT
Link: </api/v2/webhooks/endpoints>; rel="successor-version"
```

### Regenerating Documentation

The API documentation is automatically generated from Go code annotations using Swagger/OpenAPI. To regenerate:

```bash
# Install swag tool (if not already installed)
go install github.com/swaggo/swag/cmd/swag@latest

# Generate documentation
make docs

# Or use the comprehensive generation script
./scripts/generate-docs.sh
```

### Adding New Endpoints

1. **Add Swagger annotations** to your handler functions:
   ```go
   // CreateWebhook creates a new webhook
   // @Summary Create webhook
   // @Description Create a new webhook endpoint
   // @Tags webhooks
   // @Accept json
   // @Produce json
   // @Param request body CreateWebhookRequest true "Webhook creation request"
   // @Success 201 {object} WebhookResponse
   // @Failure 400 {object} ErrorResponse
   // @Security ApiKeyAuth
   // @Router /webhooks [post]
   func (h *Handler) CreateWebhook(c *gin.Context) {
       // Implementation
   }
   ```

2. **Regenerate documentation**:
   ```bash
   make docs
   ```

3. **Test the documentation**:
   ```bash
   make docs-test
   ```

### Documentation Tests

The documentation includes comprehensive tests to ensure accuracy:

- **Structure validation**: Verifies OpenAPI spec structure
- **Endpoint coverage**: Ensures all endpoints are documented
- **Response format**: Validates response structures
- **SDK examples**: Tests SDK code examples

Run tests with:
```bash
go test ./docs -v
```

## 📝 Contributing

When adding new features:

1. **Add comprehensive Swagger annotations**
2. **Update SDK if needed**
3. **Add usage examples**
4. **Run documentation tests**
5. **Update this README if necessary**

## 🔗 Related Resources

- [API Specification](swagger.json) - Complete OpenAPI 3.0 specification
- [Error Catalog](ERROR_CATALOG.md) - Complete reference for all API error codes
- [Feature Matrix](FEATURE_MATRIX.md) - Platform capabilities and maturity status
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues and solutions
- [Testing Guide](TESTING.md) - How to write, run, and debug tests
- [Testing Tools](testing_tools.md) - Webhook testing and debugging utilities
- [Go SDK](../sdk/go/) - Official Go SDK with examples
- [Testing Examples](../sdk/go/examples/testing_webhooks.go) - Webhook testing examples
- [Backend Examples](../examples/) - Backend code examples and demos
- [Error Handling](../sdk/go/examples/error_handling.go) - Error handling patterns
- [Architecture Decision Records](adrs/) - Key design decisions and rationale

## 📞 Support

- **Documentation Issues**: Open an issue in the repository
- **API Questions**: Check the interactive documentation at `/docs/`
- **SDK Support**: See the SDK README and examples
- **General Support**: Open a [GitHub Discussion](https://github.com/josedab/waas/discussions) or file an issue