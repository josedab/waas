# Webhook Platform Go SDK Examples

This directory contains practical examples demonstrating how to use the Webhook Platform Go SDK.

## Examples

### Basic Usage (`basic_usage.go`)
Demonstrates the fundamental operations:
- Creating webhook endpoints
- Listing endpoints
- Sending webhooks
- Getting delivery details
- Managing tenant information

### Testing Webhooks (`testing_webhooks.go`)
Shows how to use the testing and debugging features:
- Testing webhook endpoints
- Creating temporary test endpoints
- Inspecting delivery details
- Analyzing delivery timelines

### Advanced Usage (`advanced_usage.go`)
Covers advanced scenarios:
- Batch webhook sending
- Custom retry configurations
- Error handling patterns
- Rate limiting handling

### Error Handling (`error_handling.go`)
Demonstrates proper error handling:
- API error types
- Retry strategies
- Timeout handling
- Network error recovery

## Running the Examples

1. **Set your API key:**
   ```bash
   export WEBHOOK_PLATFORM_API_KEY="your-api-key-here"
   ```

2. **Run an example:**
   ```bash
   go run basic_usage.go
   ```

3. **Or modify the examples to use your API key directly:**
   ```go
   c := client.New("your-api-key-here")
   ```

## Common Use Cases

### Creating and Managing Endpoints

```go
// Create an endpoint
endpoint, err := c.Webhooks.CreateEndpoint(ctx, &client.CreateEndpointRequest{
    URL: "https://your-app.com/webhooks",
    CustomHeaders: map[string]string{
        "Authorization": "Bearer your-token",
    },
    RetryConfig: &client.RetryConfigurationReq{
        MaxAttempts:       5,
        InitialDelayMs:    1000,
        MaxDelayMs:        300000,
        BackoffMultiplier: 2,
    },
})

// Update an endpoint
updated, err := c.Webhooks.UpdateEndpoint(ctx, endpoint.ID, &client.UpdateEndpointRequest{
    IsActive: &[]bool{false}[0], // Disable the endpoint
})

// Delete an endpoint
err = c.Webhooks.DeleteEndpoint(ctx, endpoint.ID)
```

### Sending Webhooks

```go
// Send to a specific endpoint
delivery, err := c.Webhooks.Send(ctx, &client.SendWebhookRequest{
    EndpointID: &endpoint.ID,
    Payload: map[string]interface{}{
        "event": "user.created",
        "data": map[string]interface{}{
            "user_id": "12345",
            "email":   "user@example.com",
        },
    },
})

// Send to all active endpoints
delivery, err := c.Webhooks.Send(ctx, &client.SendWebhookRequest{
    // No EndpointID means send to all active endpoints
    Payload: map[string]interface{}{
        "event": "system.maintenance",
        "message": "System will be down for maintenance",
    },
})

// Batch send to multiple specific endpoints
batch, err := c.Webhooks.BatchSend(ctx, &client.BatchSendWebhookRequest{
    EndpointIDs: []uuid.UUID{endpoint1.ID, endpoint2.ID},
    Payload: map[string]interface{}{
        "event": "bulk.notification",
    },
})
```

### Monitoring and Debugging

```go
// Get delivery history with filters
history, err := c.Webhooks.GetDeliveryHistory(ctx, &client.DeliveryHistoryOptions{
    Statuses: []string{"failed", "retrying"},
    Limit:    50,
})

// Get detailed delivery information
details, err := c.Webhooks.GetDeliveryDetails(ctx, deliveryID)

// Inspect a delivery for debugging
inspection, err := c.Testing.InspectDelivery(ctx, deliveryID)
```

### Testing

```go
// Test any webhook URL
result, err := c.Testing.TestWebhook(ctx, &client.TestWebhookRequest{
    URL: "https://your-app.com/webhooks",
    Payload: map[string]interface{}{
        "test": true,
    },
    Timeout: 30,
})

// Create a temporary test endpoint
testEndpoint, err := c.Testing.CreateTestEndpoint(ctx, &client.CreateTestEndpointRequest{
    Name: "My Test Endpoint",
    TTL:  3600, // 1 hour
})
```

## Error Handling Patterns

### Basic Error Handling

```go
delivery, err := c.Webhooks.Send(ctx, req)
if err != nil {
    if apiErr, ok := err.(*client.APIError); ok {
        switch apiErr.StatusCode {
        case 400:
            log.Printf("Bad request: %s", apiErr.Message)
        case 401:
            log.Printf("Authentication failed: %s", apiErr.Message)
        case 429:
            log.Printf("Rate limited: %s", apiErr.Message)
            // Implement backoff and retry
        case 500:
            log.Printf("Server error: %s", apiErr.Message)
            // Implement retry logic
        default:
            log.Printf("API error %d: %s", apiErr.StatusCode, apiErr.Message)
        }
    } else {
        log.Printf("Network or other error: %v", err)
    }
    return
}
```

### Retry with Exponential Backoff

```go
func sendWithRetry(c *client.Client, ctx context.Context, req *client.SendWebhookRequest) (*client.SendWebhookResponse, error) {
    maxRetries := 3
    baseDelay := time.Second
    
    for attempt := 0; attempt <= maxRetries; attempt++ {
        delivery, err := c.Webhooks.Send(ctx, req)
        if err == nil {
            return delivery, nil
        }
        
        if apiErr, ok := err.(*client.APIError); ok {
            // Don't retry client errors (4xx)
            if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 && apiErr.StatusCode != 429 {
                return nil, err
            }
        }
        
        if attempt < maxRetries {
            delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff
            time.Sleep(delay)
        }
    }
    
    return nil, fmt.Errorf("max retries exceeded")
}
```

## Configuration Examples

### Using Environment Variables

```go
// Set environment variable
// export WEBHOOK_PLATFORM_API_KEY="your-api-key"

c := client.NewFromEnv()
```

### Custom HTTP Client

```go
httpClient := &http.Client{
    Timeout: 60 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}

c := client.NewWithHTTPClient("your-api-key", httpClient)
```

### Custom Configuration

```go
config := &client.Config{
    APIKey:  "your-api-key",
    BaseURL: "https://api.webhook-platform.com/api/v1",
    Timeout: 30 * time.Second,
}

c := client.NewWithConfig(config)
```

## Best Practices

1. **Always handle errors appropriately**
2. **Use context for timeouts and cancellation**
3. **Implement retry logic for transient failures**
4. **Use batch operations for multiple webhooks**
5. **Monitor delivery status for important webhooks**
6. **Test webhook endpoints before going live**
7. **Use structured logging for debugging**

## Need Help?

- Check the [API Documentation](https://docs.webhook-platform.com)
- Review the [SDK Reference](https://pkg.go.dev/github.com/webhook-platform/go-sdk)
- Open an issue on [GitHub](https://github.com/webhook-platform/go-sdk/issues)