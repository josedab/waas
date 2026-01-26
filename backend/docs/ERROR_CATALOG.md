# Error Catalog

Complete reference for all error codes returned by the WaaS API. Use the error `code` field to programmatically handle errors.

## Error Response Format

```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Authentication required",
    "category": "AUTHENTICATION",
    "timestamp": "2024-01-15T10:30:00Z",
    "request_id": "req_abc123",
    "debugging_hints": [
      "Ensure you have included a valid API key in the Authorization header"
    ]
  }
}
```

## Authentication & Authorization Errors

| Code | HTTP Status | Message | Remediation |
|------|------------|---------|-------------|
| `UNAUTHORIZED` | 401 | Authentication required | Include a valid API key in the `Authorization: Bearer <key>` header |
| `INVALID_API_KEY` | 401 | Invalid or expired API key | Check that the key is correct, not expired, and belongs to the right tenant |
| `FORBIDDEN` | 403 | Access denied to this resource | Verify your permissions and subscription tier |
| `TENANT_NOT_FOUND` | 401 | Tenant not found in context | Internal error — contact support if this persists |

## Validation Errors

| Code | HTTP Status | Message | Remediation |
|------|------------|---------|-------------|
| `INVALID_REQUEST` | 400 | Invalid request format | Check that the body is valid JSON with all required fields |
| `INVALID_REQUEST_BODY` | 400 | Request body could not be parsed | Ensure valid JSON and correct field types |
| `INVALID_URL` | 400 | Invalid webhook URL | URLs must use HTTPS protocol |
| `INVALID_PAYLOAD` | 400 | Invalid webhook payload | Check payload format and size |
| `PAYLOAD_TOO_LARGE` | 413 | Payload exceeds maximum size | Reduce payload size (see `details.max_size_bytes`) |
| `INVALID_ID` | 400 | Invalid resource identifier | IDs must be valid UUIDs |

## Resource Errors

| Code | HTTP Status | Message | Remediation |
|------|------------|---------|-------------|
| `ENDPOINT_NOT_FOUND` | 404 | Webhook endpoint not found | Verify the endpoint ID and that it belongs to your tenant |
| `DELIVERY_NOT_FOUND` | 404 | Delivery record not found | Check the delivery ID |
| `RESOURCE_NOT_FOUND` | 404 | The requested resource was not found | Verify the resource ID exists |
| `RESOURCE_ALREADY_EXISTS` | 409 | A resource with these details already exists | Use a different identifier or update the existing resource |
| `INVALID_REFERENCE` | 400 | Referenced resource does not exist | Ensure all referenced IDs point to existing resources |
| `ENDPOINT_INACTIVE` | 400 | Endpoint is disabled | Re-enable the endpoint before sending webhooks |
| `NO_ACTIVE_ENDPOINTS` | 400 | No active endpoints found | Create and enable at least one endpoint |

## Rate Limit & Quota Errors

| Code | HTTP Status | Message | Remediation |
|------|------------|---------|-------------|
| `RATE_LIMIT_EXCEEDED` | 429 | Rate limit exceeded | Wait and retry after the `Retry-After` header value (seconds) |
| `QUOTA_EXCEEDED` | 429 | Quota exceeded | Upgrade your plan or wait for the quota to reset |

## Server Errors

| Code | HTTP Status | Message | Remediation |
|------|------------|---------|-------------|
| `INTERNAL_ERROR` | 500 | Internal server error | Retry the request. If persistent, contact support with `request_id` |
| `DATABASE_ERROR` | 500 | Database operation failed | Transient error — retry. If persistent, the service may be degraded |
| `QUEUE_ERROR` | 500 | Message queue operation failed | Retry. The delivery queue may be experiencing issues |
| `EXTERNAL_API_ERROR` | 502 | External service error | An upstream dependency failed. Retry after a delay |
| `TIMEOUT` | 504 | Operation timed out | The operation took too long. Retry with a smaller payload or simpler query |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily unavailable | The service is restarting or under maintenance. Retry shortly |

## Delivery Errors

| Code | HTTP Status | Message | Remediation |
|------|------------|---------|-------------|
| `DELIVERY_FAILED` | 502 | Webhook delivery failed | Check endpoint health. Failed deliveries are retried automatically |
| `SIGNATURE_VERIFICATION_FAILED` | 400 | Signature verification failed | Verify your signing secret matches the endpoint configuration |
| `WEBHOOK_CLIENT_ERROR` | 502 | Endpoint returned 4xx | Fix the receiving endpoint — it's rejecting the payload |
| `WEBHOOK_SERVER_ERROR` | 502 | Endpoint returned 5xx | The receiving endpoint has server issues. Will be retried |
| `WEBHOOK_TIMEOUT` | 504 | Delivery timed out | The endpoint took too long to respond. Optimize its response time |
| `WEBHOOK_UNREACHABLE` | 502 | Endpoint is unreachable | Check URL, firewall settings, and that the server is running |
| `WEBHOOK_TLS_ERROR` | 502 | TLS/SSL error | Check the endpoint's SSL certificate (valid, not expired, correct domain) |
| `WEBHOOK_NETWORK_ERROR` | 502 | Network error | General network issue. Delivery will be retried |

## Handling Errors in Code

### By HTTP Status

```
4xx → Client error (fix your request)
  400 → Validation issue
  401 → Authentication issue
  403 → Authorization issue
  404 → Resource not found
  409 → Conflict (duplicate)
  413 → Payload too large
  429 → Rate limited / over quota

5xx → Server error (retry with backoff)
  500 → Internal error
  502 → Upstream error
  503 → Service unavailable
  504 → Timeout
```

### By Category

```
VALIDATION      → Fix request parameters
AUTHENTICATION  → Fix API key
AUTHORIZATION   → Check permissions
NOT_FOUND       → Check resource IDs
RATE_LIMIT      → Implement backoff
QUOTA_EXCEEDED  → Upgrade plan
INTERNAL        → Retry with exponential backoff
DATABASE        → Retry (transient)
DELIVERY_FAILED → Check endpoint health
```

### Retry Strategy

Errors with these categories are safe to retry with exponential backoff:
- `INTERNAL`, `DATABASE`, `QUEUE`, `TIMEOUT`, `UNAVAILABLE`
- `DELIVERY_FAILED` (retried automatically by the delivery engine)

Do **not** retry:
- `VALIDATION`, `AUTHENTICATION`, `AUTHORIZATION`, `NOT_FOUND` — fix the request first
- `RATE_LIMIT`, `QUOTA_EXCEEDED` — wait for the specified retry-after period
