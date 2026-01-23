# WAAS PHP SDK

Official PHP SDK for the WAAS (Webhook-as-a-Service) Platform.

## Requirements

- PHP >= 8.1
- Composer

## Installation

Since WaaS is self-hosted, the SDK is bundled with the repository under `backend/sdk/php/`.

```bash
# Add as a local repository in your composer.json:
composer config repositories.waas path /path/to/waas/backend/sdk/php
composer require waas/waas-sdk
```

## Quick Start

```php
<?php

use WAAS\Client;
use WAAS\Models\CreateEndpointRequest;
use WAAS\Models\SendWebhookRequest;

// Initialize client
$client = new Client('your-api-key');

// Create an endpoint
$endpoint = $client->endpoints()->create(new CreateEndpointRequest(
    url: 'https://your-server.com/webhook',
    retryConfig: [
        'max_attempts' => 5,
        'initial_delay_ms' => 1000,
    ]
));
echo "Created endpoint: {$endpoint->id}\n";

// Send a webhook
$result = $client->webhooks()->send(new SendWebhookRequest(
    endpointId: $endpoint->id,
    payload: ['event' => 'user.created', 'data' => ['id' => 123]]
));
echo "Delivery scheduled: {$result->deliveryId}\n";

// Check status
$delivery = $client->deliveries()->get($result->deliveryId);
echo "Status: {$delivery->status}\n";
```

## Configuration

```php
use WAAS\Client;
use WAAS\Config;

$config = new Config(
    apiKey: 'your-api-key',
    baseUrl: 'https://api.waas-platform.com/api/v1',
    timeout: 30.0
);

$client = new Client($config);
```

## Error Handling

```php
use WAAS\Exceptions\{
    WAASException,
    AuthenticationException,
    NotFoundException,
    RateLimitException,
    ValidationException
};

try {
    $endpoint = $client->endpoints()->get('invalid-id');
} catch (NotFoundException $e) {
    echo "Endpoint not found\n";
} catch (AuthenticationException $e) {
    echo "Invalid API key\n";
} catch (RateLimitException $e) {
    echo "Rate limited, retry after {$e->retryAfter} seconds\n";
} catch (WAASException $e) {
    echo "API error: {$e->getMessage()}\n";
}
```

## Documentation

For full documentation, visit [docs.waas-platform.com/sdks/php](https://docs.waas-platform.com/sdks/php)

## License

MIT License
