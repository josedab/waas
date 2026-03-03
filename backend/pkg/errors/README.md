# Comprehensive Error Handling System

This package provides a comprehensive, structured error handling system for the webhook platform. It implements standardized error responses, categorization, logging, alerting, and debugging support.

## Features

### 1. Standardized Error Response Format
- Consistent JSON error responses across all APIs
- Structured error information with codes, categories, and severity levels
- Request/trace ID tracking for debugging
- Client-friendly error messages with debugging hints

### 2. Error Categorization and Handling
- Automatic error categorization (validation, authentication, database, etc.)
- Severity levels (low, medium, high, critical)
- Retryable vs non-retryable error classification
- HTTP status code mapping

### 3. Error Logging and Alerting
- Structured logging with appropriate severity levels
- Automatic alerting for high-severity errors
- Rate limiting and deduplication for alerts
- Integration with Slack, email, and other notification channels

### 4. Client-Friendly Error Messages
- Debugging hints to help users resolve issues
- Links to relevant documentation
- Context-specific error details
- Production-safe error sanitization

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/josedab/waas/pkg/errors"
    "github.com/josedab/waas/pkg/utils"
    "github.com/gin-gonic/gin"
)

func main() {
    // Set up error handling
    logger := utils.NewLogger("webhook-api")
    alerter := errors.NewAlerter(context.Background(), errors.DefaultAlerterConfig(), logger)
    errorHandler := errors.NewErrorHandler(logger, alerter)
    
    // Set up Gin router with error handling middleware
    router := gin.New()
    router.Use(errors.RequestIDMiddleware())
    router.Use(errors.TraceIDMiddleware())
    router.Use(errorHandler.Middleware())
    
    // Use error handling in routes
    router.POST("/webhooks", func(c *gin.Context) {
        var request WebhookRequest
        if err := c.ShouldBindJSON(&request); err != nil {
            errors.AbortWithValidationError(c, "request", err.Error())
            return
        }
        
        // Business logic...
        if someCondition {
            errors.AbortWithForbidden(c)
            return
        }
        
        c.JSON(200, gin.H{"message": "success"})
    })
}
```

### Error Response Format

All errors follow this standardized format:

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Invalid request format",
    "category": "VALIDATION",
    "timestamp": "2024-01-01T00:00:00Z",
    "request_id": "req_123456",
    "trace_id": "trace_789012",
    "details": {
      "field": "email",
      "reason": "invalid format"
    },
    "debugging_hints": [
      "Check that your request body is valid JSON",
      "Verify all required fields are present"
    ],
    "documentation": "https://docs.webhook-platform.com/api-reference"
  }
}
```

## Error Categories

- **VALIDATION**: Input validation errors (400)
- **AUTHENTICATION**: Authentication failures (401)
- **AUTHORIZATION**: Permission denied (403)
- **NOT_FOUND**: Resource not found (404)
- **RATE_LIMIT**: Rate limit exceeded (429)
- **QUOTA_EXCEEDED**: Usage quota exceeded (402)
- **PAYLOAD_TOO_LARGE**: Request too large (400)
- **INTERNAL**: Internal server errors (500)
- **DATABASE**: Database operation failures (500)
- **QUEUE**: Message queue failures (500)
- **EXTERNAL_API**: External service failures (502)
- **TIMEOUT**: Operation timeouts (504)
- **UNAVAILABLE**: Service unavailable (503)
- **DELIVERY_FAILED**: Webhook delivery failures (502)

## Severity Levels

- **LOW**: Minor issues, user errors
- **MEDIUM**: Operational issues, temporary failures
- **HIGH**: System errors, service degradation
- **CRITICAL**: System failures, service outages

## Helper Functions

### Abort Functions
```go
// Quick error responses in handlers
errors.AbortWithValidationError(c, "email", "invalid format")
errors.AbortWithUnauthorized(c)
errors.AbortWithForbidden(c)
errors.AbortWithNotFound(c, "webhook endpoint")
errors.AbortWithRateLimit(c, 60) // retry after 60 seconds
errors.AbortWithQuotaExceeded(c, 1500, 1000) // current usage, limit
```

### Error Conversion
```go
// Convert repository errors
if err := repo.GetUser(id); err != nil {
    webhookErr := errors.HandleRepositoryError(err)
    if webhookErr != nil {
        errors.AbortWithError(c, webhookErr)
        return
    }
}

// Convert network errors
if err := httpClient.Post(url, payload); err != nil {
    webhookErr := errors.HandleNetworkError(err, url)
    if webhookErr != nil {
        return webhookErr
    }
}
```

### Custom Errors
```go
// Create custom errors
customErr := errors.NewWebhookError(
    "CUSTOM_ERROR",
    "Custom error message",
    errors.CategoryValidation,
    http.StatusBadRequest,
    errors.SeverityLow,
).WithDetails(map[string]interface{}{
    "field": "custom_field",
    "value": "invalid_value",
}).WithDebuggingHints(
    "Check the custom field format",
    "Refer to the API documentation",
).WithDocumentation("https://docs.example.com/custom-errors")
```

## Alerting Configuration

```go
alerterConfig := &errors.AlerterConfig{
    Enabled:                true,
    MaxAlertsPerMinute:     10,
    MaxAlertsPerHour:       100,
    SlackWebhookURL:        "https://hooks.slack.com/...",
    EmailRecipients:        []string{"alerts@company.com"},
    CriticalAlertThreshold: 1 * time.Minute,
    HighAlertThreshold:     5 * time.Minute,
}

alerter := errors.NewAlerter(context.Background(), alerterConfig, logger)
```

## Predefined Errors

The package includes predefined errors for common scenarios:

- `ErrUnauthorized` - Authentication required
- `ErrInvalidAPIKey` - Invalid or expired API key
- `ErrForbidden` - Access denied
- `ErrInvalidRequest` - Invalid request format
- `ErrInvalidURL` - Invalid webhook URL
- `ErrPayloadTooLarge` - Payload exceeds size limit
- `ErrRateLimitExceeded` - Rate limit exceeded
- `ErrQuotaExceeded` - Monthly quota exceeded
- `ErrInternalServer` - Internal server error
- `ErrDatabaseError` - Database operation failed
- `ErrDeliveryFailed` - Webhook delivery failed

## Testing

The package includes comprehensive test coverage:

```bash
go test ./pkg/errors/...
```

Test categories:
- Unit tests for error types and functions
- Integration tests for middleware and handlers
- Mock tests for alerting system
- End-to-end tests for complete error flows

## Best Practices

1. **Use Predefined Errors**: Use existing error definitions when possible
2. **Add Context**: Include relevant details and debugging hints
3. **Categorize Correctly**: Choose appropriate category and severity
4. **Log Appropriately**: Use structured logging with correlation IDs
5. **Handle Retryable Errors**: Distinguish between retryable and permanent failures
6. **Sanitize in Production**: Remove sensitive information from error responses
7. **Monitor and Alert**: Set up appropriate alerting for high-severity errors

## Integration with Existing Code

The error handling system is designed to integrate seamlessly with existing handlers:

```go
// Before
func (h *Handler) CreateWebhook(c *gin.Context) {
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
}

// After
func (h *Handler) CreateWebhook(c *gin.Context) {
    if err := c.ShouldBindJSON(&req); err != nil {
        errors.AbortWithValidationError(c, "request", err.Error())
        return
    }
}
```

## Production Considerations

- **Error Sanitization**: Sensitive information is automatically removed in production mode
- **Rate Limiting**: Alerts are rate-limited to prevent spam
- **Performance**: Minimal overhead with efficient error categorization
- **Monitoring**: Integration with observability tools for error tracking
- **Documentation**: Automatic links to relevant documentation for common errors

This error handling system provides a robust foundation for building reliable, user-friendly APIs with comprehensive error management and debugging support.