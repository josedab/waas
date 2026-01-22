# Webhook Testing and Debugging Tools

This document describes the webhook testing and debugging tools implemented as part of the webhook service platform.

## Overview

The testing tools provide developers with comprehensive capabilities to test, debug, and monitor webhook deliveries without needing to set up complex infrastructure. These tools are essential for validating webhook implementations and troubleshooting delivery issues.

## Features

### 1. Webhook Testing Interface

Test webhook deliveries to any endpoint with custom payloads and headers.

**Endpoint:** `POST /api/v1/webhooks/test`

**Request:**
```json
{
  "url": "https://your-endpoint.com/webhook",
  "payload": {
    "event": "test.event",
    "data": {"key": "value"}
  },
  "headers": {
    "X-Custom-Header": "value"
  },
  "method": "POST",
  "timeout": 30
}
```

**Response:**
```json
{
  "test_id": "uuid",
  "url": "https://your-endpoint.com/webhook",
  "status": "success",
  "http_status": 200,
  "latency_ms": 150,
  "request_id": "test_abc123",
  "tested_at": "2024-01-01T12:00:00Z"
}
```

### 2. Test Endpoint Generation

Create temporary webhook endpoints for immediate testing.

**Endpoint:** `POST /api/v1/webhooks/test/endpoints`

**Request:**
```json
{
  "name": "My Test Endpoint",
  "description": "For testing webhook delivery",
  "headers": {
    "Authorization": "Bearer token"
  },
  "ttl": 3600
}
```

**Response:**
```json
{
  "id": "uuid",
  "url": "http://localhost:8080/test/uuid",
  "name": "My Test Endpoint",
  "description": "For testing webhook delivery",
  "headers": {
    "Authorization": "Bearer token"
  },
  "created_at": "2024-01-01T12:00:00Z",
  "expires_at": "2024-01-01T13:00:00Z"
}
```

### 3. Delivery Inspection Tools

Get detailed information about webhook deliveries for debugging.

**Endpoint:** `GET /api/v1/webhooks/deliveries/{delivery-id}/inspect`

**Response:**
```json
{
  "delivery_id": "uuid",
  "endpoint_id": "uuid",
  "status": "success",
  "attempt_number": 2,
  "request": {
    "url": "https://example.com/webhook",
    "method": "POST",
    "headers": {"Content-Type": "application/json"},
    "payload_hash": "abc123",
    "payload_size": 256,
    "signature": "sha256=...",
    "scheduled_at": "2024-01-01T12:00:00Z"
  },
  "response": {
    "http_status": 200,
    "headers": {"Content-Type": "application/json"},
    "body": "OK",
    "body_size": 2,
    "delivered_at": "2024-01-01T12:00:01Z",
    "latency_ms": 150
  },
  "timeline": [
    {
      "timestamp": "2024-01-01T12:00:00Z",
      "event": "scheduled",
      "description": "Delivery attempt 1 scheduled",
      "details": {"attempt_number": 1}
    },
    {
      "timestamp": "2024-01-01T12:00:01Z",
      "event": "delivered",
      "description": "Delivery attempt 1 completed",
      "details": {"attempt_number": 1, "status": "success", "http_status": 200}
    }
  ],
  "error_details": null
}
```

### 4. Delivery Logs

View detailed logs for webhook delivery attempts.

**Endpoint:** `GET /api/v1/webhooks/deliveries/{delivery-id}/logs`

**Response:**
```json
{
  "delivery_id": "uuid",
  "logs": [
    {
      "attempt_number": 1,
      "status": "failed",
      "http_status": 500,
      "error_message": "Internal Server Error",
      "scheduled_at": "2024-01-01T12:00:00Z",
      "delivered_at": "2024-01-01T12:00:01Z",
      "created_at": "2024-01-01T12:00:00Z"
    },
    {
      "attempt_number": 2,
      "status": "success",
      "http_status": 200,
      "response_body": "OK",
      "scheduled_at": "2024-01-01T12:01:00Z",
      "delivered_at": "2024-01-01T12:01:01Z",
      "created_at": "2024-01-01T12:01:00Z"
    }
  ],
  "total_attempts": 2
}
```

### 5. Real-time Updates

Connect to WebSocket for real-time delivery status updates.

**Endpoint:** `GET /api/v1/webhooks/realtime` (WebSocket)

**Messages:**
```json
{
  "type": "delivery_update",
  "data": {
    "delivery_id": "uuid",
    "status": "success",
    "http_status": 200,
    "timestamp": "2024-01-01T12:00:01Z"
  },
  "timestamp": "2024-01-01T12:00:01Z"
}
```

### 6. Test Endpoint Receivers

Receive and inspect webhooks sent to test endpoints.

**Receive Webhook:** `ANY /test/{endpoint-id}`
**View Receives:** `GET /test/{endpoint-id}/receives`
**View Specific Receive:** `GET /test/{endpoint-id}/receives/{receive-id}`
**Clear Receives:** `DELETE /test/{endpoint-id}/receives`

## Error Analysis

The inspection tools provide intelligent error analysis with suggestions:

### Error Types

- **timeout**: Request timed out
- **connection**: Connection failed
- **dns**: DNS resolution failed
- **client_error**: HTTP 4xx errors
- **server_error**: HTTP 5xx errors
- **authentication**: HTTP 401/403 errors
- **not_found**: HTTP 404 errors

### Error Suggestions

The system provides contextual suggestions based on error types:

- For timeout errors: Check endpoint response time, increase timeout
- For connection errors: Verify URL accessibility, check network
- For authentication errors: Verify credentials and permissions
- For server errors: Check endpoint logs, retry may succeed

## Usage Examples

### Testing a Webhook Endpoint

```bash
curl -X POST http://localhost:8080/api/v1/webhooks/test \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-endpoint.com/webhook",
    "payload": {
      "event": "user.created",
      "user_id": "123"
    },
    "headers": {
      "X-Event-Type": "user.created"
    }
  }'
```

### Creating a Test Endpoint

```bash
curl -X POST http://localhost:8080/api/v1/webhooks/test/endpoints \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "User Events Test",
    "description": "Testing user event webhooks",
    "ttl": 7200
  }'
```

### Inspecting a Delivery

```bash
curl -X GET http://localhost:8080/api/v1/webhooks/deliveries/uuid/inspect \
  -H "Authorization: Bearer your-api-key"
```

## Integration with Development Workflow

### 1. Development Phase
- Create test endpoints for local development
- Test webhook payloads and formats
- Validate endpoint implementations

### 2. Testing Phase
- Use webhook testing interface for automated tests
- Inspect delivery details for debugging
- Monitor real-time delivery status

### 3. Debugging Phase
- Analyze failed deliveries with inspection tools
- Review delivery timelines and error details
- Use suggested fixes for common issues

### 4. Monitoring Phase
- Set up real-time monitoring via WebSocket
- Track delivery success rates
- Identify patterns in delivery failures

## Security Considerations

- Test endpoints are temporary and expire automatically
- All testing operations require authentication
- Test endpoint URLs are unpredictable (UUID-based)
- Received webhook data is not persisted long-term
- Rate limiting applies to testing endpoints

## Implementation Details

The testing tools are implemented with the following components:

- **TestingHandler**: Main handler for testing operations
- **TestEndpointHandler**: Handles test endpoint receivers
- **WebSocket Manager**: Manages real-time connections
- **Error Analyzer**: Provides intelligent error analysis
- **Timeline Builder**: Creates delivery event timelines

## API Authentication

All testing endpoints require authentication via API key:

```
Authorization: Bearer your-api-key
```

The API key should be included in the request headers for all testing operations.