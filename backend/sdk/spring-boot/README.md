# WaaS Spring Boot Starter

Webhook integration auto-configuration for Spring Boot applications.

## Installation

Since WaaS is self-hosted, install the module locally with Maven:

```bash
cd /path/to/waas/backend/sdk/spring-boot
mvn install
```

Then add the dependency to your project's `pom.xml`:

```xml
<dependency>
    <groupId>dev.waas</groupId>
    <artifactId>waas-spring-boot-starter</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Requirements

- Java ≥ 17
- Spring Boot ≥ 3.0

## Quick Start

1. Add configuration to `application.properties`:

```properties
waas.api-url=http://localhost:8080
waas.api-key=${WAAS_API_KEY}
```

2. Inject and use the WaaS client:

```java
import dev.waas.springboot.WaaSClient;
import org.springframework.stereotype.Service;

@Service
public class OrderService {

    private final WaaSClient waas;

    public OrderService(WaaSClient waas) {
        this.waas = waas;
    }

    public void createOrder(Order order) {
        // ... save order ...
        waas.send("your-endpoint-id",
            Map.of("event", "order.created", "data", order));
    }
}
```

3. Receive webhooks with signature verification:

```java
import dev.waas.springboot.annotation.WebhookEndpoint;

@RestController
public class WebhookController {

    @WebhookEndpoint(path = "/webhooks", secret = "${waas.signing-secret}")
    public void handleWebhook(@RequestBody Map<String, Object> payload) {
        System.out.println("Received webhook: " + payload);
    }
}
```

## Documentation

For detailed API documentation, see the [API docs](../../docs/README.md).

## License

MIT — see [LICENSE](../../LICENSE) for details.
