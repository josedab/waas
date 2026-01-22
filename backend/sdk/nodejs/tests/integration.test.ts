/**
 * Integration tests for WAAS Node.js SDK.
 *
 * These tests require a running WAAS server. Set WAAS_API_KEY and WAAS_BASE_URL
 * environment variables to run these tests.
 *
 * Run with: npm test -- --grep "Integration"
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { WAASClient } from '../src/client';
import type { WebhookEndpoint } from '../src/types';

const isIntegrationTest = process.env.WAAS_API_KEY != null;

describe.skipIf(!isIntegrationTest)('Integration Tests', () => {
  let client: WAASClient;
  const createdEndpoints: string[] = [];

  beforeAll(() => {
    client = new WAASClient({
      apiKey: process.env.WAAS_API_KEY!,
      baseUrl: process.env.WAAS_BASE_URL || 'http://localhost:8080/api/v1',
    });
  });

  afterAll(async () => {
    // Clean up created endpoints
    for (const id of createdEndpoints) {
      try {
        await client.endpoints.delete(id);
      } catch {
        // Ignore cleanup errors
      }
    }
  });

  describe('Endpoints', () => {
    it('should create and retrieve an endpoint', async () => {
      // Create
      const endpoint = await client.endpoints.create({
        url: 'https://httpbin.org/post',
        retryConfig: {
          maxAttempts: 3,
          initialDelayMs: 500,
        },
      });

      createdEndpoints.push(endpoint.id);

      expect(endpoint.id).toBeDefined();
      expect(endpoint.url).toBe('https://httpbin.org/post');
      expect(endpoint.isActive).toBe(true);
      expect(endpoint.secret).toBeDefined(); // Secret returned on creation

      // Get
      const fetched = await client.endpoints.get(endpoint.id);
      expect(fetched.id).toBe(endpoint.id);
      expect(fetched.url).toBe(endpoint.url);
    });

    it('should list endpoints', async () => {
      // Create a few endpoints
      for (let i = 0; i < 3; i++) {
        const endpoint = await client.endpoints.create({
          url: `https://httpbin.org/post?test=${i}`,
        });
        createdEndpoints.push(endpoint.id);
      }

      // List
      const result = await client.endpoints.list({ perPage: 10 });
      expect(result.endpoints).toBeDefined();
      expect(result.endpoints.length).toBeGreaterThanOrEqual(3);
    });

    it('should update an endpoint', async () => {
      // Create
      const endpoint = await client.endpoints.create({
        url: 'https://httpbin.org/post',
      });
      createdEndpoints.push(endpoint.id);

      // Update
      const updated = await client.endpoints.update(endpoint.id, {
        url: 'https://httpbin.org/anything',
        isActive: false,
      });

      expect(updated.url).toBe('https://httpbin.org/anything');
      expect(updated.isActive).toBe(false);
    });

    it('should delete an endpoint', async () => {
      // Create
      const endpoint = await client.endpoints.create({
        url: 'https://httpbin.org/post',
      });

      // Delete
      await client.endpoints.delete(endpoint.id);

      // Verify deleted
      await expect(client.endpoints.get(endpoint.id)).rejects.toThrow();
    });
  });

  describe('Webhooks', () => {
    it('should send a webhook', async () => {
      // Create endpoint
      const endpoint = await client.endpoints.create({
        url: 'https://httpbin.org/post',
      });
      createdEndpoints.push(endpoint.id);

      // Send webhook
      const result = await client.webhooks.send({
        endpointId: endpoint.id,
        payload: { event: 'test.event', data: { message: 'Hello!' } },
      });

      expect(result.deliveryId).toBeDefined();
      expect(result.status).toBe('pending');

      // Get delivery
      const delivery = await client.deliveries.get(result.deliveryId);
      expect(delivery.endpointId).toBe(endpoint.id);
    });
  });

  describe('Deliveries', () => {
    it('should list deliveries', async () => {
      const result = await client.deliveries.list({ perPage: 5 });

      expect(result.deliveries).toBeDefined();
      expect(result.total).toBeDefined();
    });

    it('should retry a delivery', async () => {
      // Create endpoint and send webhook
      const endpoint = await client.endpoints.create({
        url: 'https://httpbin.org/post',
      });
      createdEndpoints.push(endpoint.id);

      const sendResult = await client.webhooks.send({
        endpointId: endpoint.id,
        payload: { test: true },
      });

      // Wait a bit for processing
      await new Promise((resolve) => setTimeout(resolve, 2000));

      // Retry
      const retryResult = await client.deliveries.retry(sendResult.deliveryId);
      expect(retryResult.deliveryId).toBeDefined();
    });
  });
});
