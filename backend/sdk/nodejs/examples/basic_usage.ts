/**
 * Basic usage example for the WAAS Node.js SDK.
 *
 * Prerequisites:
 *   1. npm install @waas/sdk  (or: npm link from sdk/nodejs/)
 *   2. A running WaaS API:    cd backend && make dev-setup && make run-api
 *
 * Run:
 *   npx tsx examples/basic_usage.ts
 *
 * This script walks through the core webhook workflow:
 *   - Create a tenant → get an API key
 *   - Create a webhook endpoint
 *   - Send a webhook
 *   - Check delivery status
 */

import axios from 'axios';
import { WAASClient } from '../src';

const BASE_URL = 'http://localhost:8080/api/v1';

async function createTenant(): Promise<string> {
  const { data } = await axios.post(`${BASE_URL}/tenants`, {
    name: 'nodejs-sdk-demo',
    email: 'demo@example.com',
  });
  console.log(`✅ Tenant created: ${data.name} (id: ${data.id})`);
  return data.api_key;
}

async function main(): Promise<void> {
  // Step 1: Create a tenant to get an API key
  console.log('1. Creating tenant...');
  const apiKey = await createTenant();

  // Step 2: Initialize the SDK client
  const client = new WAASClient({ apiKey, baseUrl: BASE_URL });

  // Step 3: Create a webhook endpoint
  console.log('\n2. Creating webhook endpoint...');
  const endpoint = await client.endpoints.create({
    url: 'https://httpbin.org/post',
    customHeaders: { 'X-Source': 'waas-node-demo' },
    retryConfig: { maxAttempts: 3, initialDelayMs: 1000 },
  });
  console.log(`✅ Endpoint created: ${endpoint.id} → ${endpoint.url}`);

  // Step 4: Send a webhook
  console.log('\n3. Sending webhook...');
  const delivery = await client.webhooks.send({
    endpointId: endpoint.id,
    payload: {
      event: 'user.created',
      data: { userId: '42', email: 'alice@example.com' },
    },
    headers: { 'X-Event-Type': 'user.created' },
  });
  console.log(`✅ Webhook queued: deliveryId=${delivery.deliveryId}`);

  // Step 5: List endpoints
  console.log('\n4. Listing endpoints...');
  const result = await client.endpoints.list();
  for (const ep of result.data) {
    console.log(`   - ${ep.id}  active=${ep.isActive}  url=${ep.url}`);
  }

  console.log(
    '\n🎉 Done! Check http://localhost:8080/docs/ for the full API reference.'
  );
}

main().catch((err) => {
  console.error('Error:', err.message);
  process.exit(1);
});
