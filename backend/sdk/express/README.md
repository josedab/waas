# WaaS Express.js Middleware

Webhook integration middleware for Express.js applications.

## Installation

Since WaaS is self-hosted, install from the local SDK directory:

```bash
npm install /path/to/waas/backend/sdk/express
```

### Requirements

- Node.js ≥ 18
- Express ≥ 4.18

## Quick Start

```typescript
import express from "express";
import { waas, webhookReceiver } from "@waas/express";

const app = express();

// Initialize the WaaS client
const client = waas({
  apiUrl: "http://localhost:8080",
  apiKey: "your-api-key",
});

// Send a webhook
app.post("/orders", async (req, res) => {
  await client.send({
    endpointId: "your-endpoint-id",
    payload: { event: "order.created", data: req.body },
  });
  res.json({ ok: true });
});

// Receive webhooks with signature verification
app.post("/webhooks", webhookReceiver({ secret: "your-signing-secret" }), (req, res) => {
  console.log("Received webhook:", req.body);
  res.sendStatus(200);
});

app.listen(3000);
```

## Documentation

For detailed API documentation, see the [API docs](../../docs/README.md).

## License

MIT — see [LICENSE](../../LICENSE) for details.
