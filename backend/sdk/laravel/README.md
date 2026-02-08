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
WAAS_API_KEY=your-api-key
```

3. Send webhooks:

```php
use WaaS\Laravel\Facades\WaaS;

WaaS::send('your-endpoint-id', [
    'event' => 'order.created',
    'data'  => ['id' => 123, 'total' => 49.99],
]);
```

## Documentation

For detailed API documentation, see the [API docs](../../docs/README.md).

## License

MIT — see [LICENSE](../../LICENSE) for details.
