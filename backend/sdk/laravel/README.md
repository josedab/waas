# WaaS Laravel Integration

Webhook integration for Laravel applications with auto-discovery support.

## Installation

Since WaaS is self-hosted, install from the local SDK directory:

```bash
composer config repositories.waas path /path/to/waas/backend/sdk/laravel
composer require waas/laravel
```

### Requirements

- PHP ≥ 8.1
- Laravel ≥ 10.0

## Quick Start

1. The service provider is auto-discovered. Publish the config:

```bash
php artisan vendor:publish --tag=waas-config
```

2. Set your credentials in `.env`:

```env
WAAS_API_URL=http://localhost:8080
WAAS_API_KEY=wh_your_api_key
WAAS_SIGNING_SECRET=your_signing_secret
```

3. Send webhooks:

```php
use WaaS\Laravel\WaaS;

// Send to a specific endpoint
WaaS::sendWebhook('endpoint-uuid', 'order.created', [
    'id' => 123,
    'total' => 49.99,
]);

// Broadcast to all active endpoints
WaaS::broadcastWebhook('user.created', ['user_id' => 42]);

// List endpoints
$endpoints = WaaS::listEndpoints();
```

## Receiving Webhooks

Use the verification middleware to protect your webhook routes:

```php
// In routes/api.php
Route::post('/webhooks', [WebhookController::class, 'handle'])
    ->middleware(\WaaS\Laravel\Middleware\VerifyWebhookSignature::class);
```

Or register the middleware alias in `app/Http/Kernel.php`:

```php
protected $middlewareAliases = [
    'waas.verify' => \WaaS\Laravel\Middleware\VerifyWebhookSignature::class,
];
```

## API Reference

```php
$client = app(\WaaS\Laravel\WaaSClient::class);

// Endpoints
$client->createEndpoint('https://example.com/webhook');
$client->listEndpoints($limit, $offset);
$client->getEndpoint($endpointId);
$client->deleteEndpoint($endpointId);

// Sending
$client->sendWebhook($endpointId, $eventType, $payload);
$client->broadcastWebhook($eventType, $payload);

// Deliveries
$client->getDelivery($deliveryId);
$client->listDeliveries($filters, $limit, $offset);

// Tenant
$client->getTenant();
```

## Documentation

For detailed API documentation, see the [API docs](../../docs/README.md).

## License

MIT — see [LICENSE](../../LICENSE) for details.
