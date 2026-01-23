# Webhook Platform Go SDK

The official Go SDK for the Webhook Service Platform API.

## Installation

Since WaaS is self-hosted, the SDK is bundled with the repository under `backend/sdk/go/`.

**Option 1: Copy into your project**
```bash
cp -r /path/to/waas/backend/sdk/go/client ./your-project/waas-client
```

**Option 2: Go workspace (for local development against the WaaS source)**
```bash
go work init
go work use . /path/to/waas/backend/sdk/go
```

**Option 3: Replace directive in go.mod**
```
require github.com/webhook-platform/go-sdk v0.0.0
replace github.com/webhook-platform/go-sdk => /path/to/waas/backend/sdk/go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/webhook-platform/go-sdk/client"
)

func main() {
    // Initialize the client with your API key
    c := client.New("your-api-key-here")
    
    // Create a webhook endpoint
    endpoint, err := c.Webhooks.CreateEndpoint(context.Background(), &client.CreateEndpointRequest{
        URL: "https://your-app.com/webhooks",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Created endpoint: %s\n", endpoint.ID)
    
    // Send a webhook
    delivery, err := c.Webhooks.Send(context.Background(), &client.SendWebhookRequest{
        EndpointID: &endpoint.ID,
        Payload:    map[string]interface{}{"message": "Hello, World!"},
    })
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Webhook sent with delivery ID: %s\n", delivery.DeliveryID)
}
```

## Features

- **Webhook Management**: Create, update, delete, and list webhook endpoints
- **Webhook Sending**: Send webhooks to individual or multiple endpoints
- **Delivery Monitoring**: Track delivery status and view detailed logs
- **Testing Tools**: Test webhook endpoints and inspect delivery details
- **Tenant Management**: Manage your account and API keys

## Documentation

For detailed documentation and examples, visit [https://docs.webhook-platform.com/sdk/go](https://docs.webhook-platform.com/sdk/go)

## API Reference

The SDK provides access to all Webhook Platform API endpoints:

### Webhook Endpoints
- `CreateEndpoint()` - Create a new webhook endpoint
- `GetEndpoint()` - Get endpoint details
- `ListEndpoints()` - List all endpoints
- `UpdateEndpoint()` - Update endpoint configuration
- `DeleteEndpoint()` - Delete an endpoint

### Webhook Sending
- `Send()` - Send a webhook to one or all endpoints
- `BatchSend()` - Send webhooks to multiple endpoints

### Monitoring
- `GetDeliveryHistory()` - Get delivery history with filtering
- `GetDeliveryDetails()` - Get detailed delivery information

### Testing
- `TestWebhook()` - Test a webhook endpoint
- `CreateTestEndpoint()` - Create a temporary test endpoint
- `InspectDelivery()` - Get detailed debugging information

### Tenant Management
- `GetTenant()` - Get current tenant information
- `UpdateTenant()` - Update tenant settings
- `RegenerateAPIKey()` - Generate a new API key

## Error Handling

The SDK provides structured error handling:

```go
delivery, err := c.Webhooks.Send(ctx, req)
if err != nil {
    if apiErr, ok := err.(*client.APIError); ok {
        fmt.Printf("API Error: %s (Code: %s)\n", apiErr.Message, apiErr.Code)
        if apiErr.StatusCode == 429 {
            // Handle rate limiting
        }
    } else {
        // Handle other errors (network, etc.)
        log.Fatal(err)
    }
}
```

## Configuration

### Authentication

The SDK uses API key authentication. You can provide your API key in several ways:

```go
// 1. Directly in the constructor
c := client.New("your-api-key")

// 2. Using environment variable WEBHOOK_PLATFORM_API_KEY
c := client.NewFromEnv()

// 3. Using a configuration struct
config := &client.Config{
    APIKey:  "your-api-key",
    BaseURL: "https://api.webhook-platform.com", // optional
    Timeout: 30 * time.Second,                   // optional
}
c := client.NewWithConfig(config)
```

### Custom HTTP Client

You can provide your own HTTP client for advanced configuration:

```go
httpClient := &http.Client{
    Timeout: 60 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns: 100,
    },
}

c := client.NewWithHTTPClient("your-api-key", httpClient)
```

## Examples

See the [examples](./examples) directory for more detailed usage examples.

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

## License

This SDK is licensed under the MIT License. See [LICENSE](LICENSE) for details.