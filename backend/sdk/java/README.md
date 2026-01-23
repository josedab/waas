# WAAS Java SDK

Official Java SDK for the WAAS (Webhook-as-a-Service) Platform.

## Requirements

- Java 17 or higher
- Maven 3.8+ or Gradle 8+

## Installation

Since WaaS is self-hosted, the SDK is bundled with the repository under `backend/sdk/java/`.

### Maven (local install)

```bash
cd /path/to/waas/backend/sdk/java
mvn install
```

Then add to your project's `pom.xml`:

```xml
<dependency>
    <groupId>com.waas</groupId>
    <artifactId>waas-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle

```groovy
implementation files('/path/to/waas/backend/sdk/java/build/libs/waas-sdk-1.0.0.jar')
```

## Quick Start

```java
import com.waas.sdk.client.WAASClient;
import com.waas.sdk.models.*;

public class Example {
    public static void main(String[] args) {
        // Initialize client
        WAASClient client = WAASClient.builder()
            .apiKey("your-api-key")
            .build();

        // Create a webhook endpoint
        WebhookEndpoint endpoint = client.endpoints().create(
            CreateEndpointRequest.builder()
                .url("https://your-server.com/webhook")
                .retryConfig(RetryConfiguration.builder()
                    .maxAttempts(5)
                    .initialDelayMs(1000)
                    .build())
                .build()
        );
        System.out.println("Created endpoint: " + endpoint.getId());

        // Send a webhook
        SendWebhookResponse delivery = client.webhooks().send(
            SendWebhookRequest.builder()
                .endpointId(endpoint.getId())
                .payload(Map.of("event", "user.created", "data", Map.of("id", 123)))
                .build()
        );
        System.out.println("Delivery scheduled: " + delivery.getDeliveryId());

        // Check delivery status
        DeliveryAttempt status = client.deliveries().get(delivery.getDeliveryId());
        System.out.println("Status: " + status.getStatus());
    }
}
```

## Features

- Full API coverage for endpoints, deliveries, analytics, and transformations
- Builder pattern for all request objects
- Comprehensive exception handling
- Thread-safe HTTP client

## Documentation

For full documentation, visit [docs.waas-platform.com/sdks/java](https://docs.waas-platform.com/sdks/java)

## License

MIT License
