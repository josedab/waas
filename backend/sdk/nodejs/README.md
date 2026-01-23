# WAAS Node.js SDK

Official Node.js/TypeScript SDK for the WAAS (Webhook-as-a-Service) Platform.

## Installation

Since WaaS is self-hosted, the SDK is bundled with the repository under `backend/sdk/nodejs/`.

```bash
# Install from local source
npm install /path/to/waas/backend/sdk/nodejs
# or
pnpm add /path/to/waas/backend/sdk/nodejs
```

## Quick Start

```typescript
import { WAASClient } from '@waas/sdk';

// Initialize client
const client = new WAASClient({
  apiKey: 'your-api-key',
});

// Create a webhook endpoint
const endpoint = await client.endpoints.create({
  url: 'https://your-server.com/webhook',
  retryConfig: {
    maxAttempts: 5,
    initialDelayMs: 1000,
  },
});
console.log(`Created endpoint: ${endpoint.id}`);

// Send a webhook
const delivery = await client.webhooks.send({
  endpointId: endpoint.id,
  payload: { event: 'user.created', data: { id: 123 } },
});
console.log(`Delivery scheduled: ${delivery.deliveryId}`);

// Check delivery status
const status = await client.deliveries.get(delivery.deliveryId);
console.log(`Status: ${status.status}`);
```

## Features

- Full TypeScript support with comprehensive types
- Promise-based API
- Automatic retries with exponential backoff
- Full API coverage (endpoints, deliveries, analytics, transformations, testing)
- ESM and CommonJS support

## Examples

See [`examples/basic_usage.ts`](examples/basic_usage.ts) for a complete walkthrough: create a tenant, register an endpoint, send a webhook, and list endpoints — all against a local WaaS instance.

Run it with: `npx tsx examples/basic_usage.ts`

## Documentation

For full documentation, visit [docs.waas-platform.com/sdks/nodejs](https://docs.waas-platform.com/sdks/nodejs)

## License

MIT License
