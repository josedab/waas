# WaaS Next.js Integration

Webhook integration for Next.js applications (App Router & Pages Router).

## Installation

Since WaaS is self-hosted, install from the local SDK directory:

```bash
npm install /path/to/waas/backend/sdk/nextjs
```

### Requirements

- Node.js ≥ 18
- Next.js ≥ 13

## Quick Start

### Sending Webhooks (Server Action / API Route)

```typescript
import { WaaS } from "@waas/nextjs";

const waas = new WaaS({
  apiUrl: "http://localhost:8080",
  apiKey: process.env.WAAS_API_KEY!,
});

// In a Server Action or Route Handler
export async function POST(request: Request) {
  const body = await request.json();

  await waas.send({
    endpointId: "your-endpoint-id",
    payload: { event: "order.created", data: body },
  });

  return Response.json({ ok: true });
}
```

### Receiving Webhooks (Route Handler)

```typescript
import { verifyWebhook } from "@waas/nextjs";

export async function POST(request: Request) {
  const payload = await verifyWebhook(request, {
    secret: process.env.WAAS_SIGNING_SECRET!,
  });

  console.log("Received webhook:", payload);
  return new Response("OK", { status: 200 });
}
```

## Documentation

For detailed API documentation, see the [API docs](../../docs/README.md).

## License

MIT — see [LICENSE](../../LICENSE) for details.
