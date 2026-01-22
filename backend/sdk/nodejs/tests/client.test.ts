import { describe, it, expect, vi, beforeEach } from 'vitest';
import { WAASClient } from '../src/client';
import {
  WAASError,
  WAASAuthenticationError,
  WAASNotFoundError,
  WAASRateLimitError,
} from '../src/errors';
import type { WebhookEndpoint, DeliveryAttempt } from '../src/types';

describe('WAASClient', () => {
  describe('initialization', () => {
    it('should initialize with API key', () => {
      const client = new WAASClient({ apiKey: 'test-key' });
      expect(client).toBeDefined();
      expect(client.endpoints).toBeDefined();
      expect(client.deliveries).toBeDefined();
      expect(client.webhooks).toBeDefined();
    });

    it('should throw error without API key', () => {
      expect(() => new WAASClient({ apiKey: '' })).toThrow('API key is required');
    });

    it('should accept custom base URL', () => {
      const client = new WAASClient({
        apiKey: 'test-key',
        baseUrl: 'https://custom.api.com/v1',
      });
      expect(client).toBeDefined();
    });

    it('should accept custom timeout', () => {
      const client = new WAASClient({
        apiKey: 'test-key',
        timeout: 60000,
      });
      expect(client).toBeDefined();
    });
  });

  describe('endpoints service', () => {
    let client: WAASClient;

    beforeEach(() => {
      client = new WAASClient({ apiKey: 'test-key' });
      // Mock axios
      vi.spyOn(client['http'], 'post').mockImplementation(async () => ({
        data: {
          id: '123e4567-e89b-12d3-a456-426614174000',
          tenant_id: 'tenant-123',
          url: 'https://example.com/webhook',
          is_active: true,
          retry_config: {
            max_attempts: 5,
            initial_delay_ms: 1000,
            max_delay_ms: 300000,
            backoff_multiplier: 2,
          },
          custom_headers: {},
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          secret: 'webhook-secret-123',
        },
      }));
    });

    it('should create an endpoint', async () => {
      const endpoint = await client.endpoints.create({
        url: 'https://example.com/webhook',
      });

      expect(endpoint.id).toBe('123e4567-e89b-12d3-a456-426614174000');
      expect(endpoint.url).toBe('https://example.com/webhook');
      expect(endpoint.isActive).toBe(true);
      expect(endpoint.secret).toBe('webhook-secret-123');
    });

    it('should get an endpoint', async () => {
      vi.spyOn(client['http'], 'get').mockImplementation(async () => ({
        data: {
          id: '123e4567-e89b-12d3-a456-426614174000',
          tenant_id: 'tenant-123',
          url: 'https://example.com/webhook',
          is_active: true,
          retry_config: {
            max_attempts: 5,
            initial_delay_ms: 1000,
          },
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
      }));

      const endpoint = await client.endpoints.get('123e4567-e89b-12d3-a456-426614174000');

      expect(endpoint.id).toBe('123e4567-e89b-12d3-a456-426614174000');
      expect(endpoint.url).toBe('https://example.com/webhook');
    });
  });

  describe('deliveries service', () => {
    let client: WAASClient;

    beforeEach(() => {
      client = new WAASClient({ apiKey: 'test-key' });
    });

    it('should get a delivery', async () => {
      vi.spyOn(client['http'], 'get').mockImplementation(async () => ({
        data: {
          id: 'delivery-123',
          endpoint_id: 'endpoint-456',
          payload_hash: 'abc123',
          payload_size: 256,
          status: 'success',
          http_status: 200,
          response_body: '{"received": true}',
          attempt_number: 1,
          scheduled_at: '2024-01-01T00:00:00Z',
          delivered_at: '2024-01-01T00:00:01Z',
          created_at: '2024-01-01T00:00:00Z',
        },
      }));

      const delivery = await client.deliveries.get('delivery-123');

      expect(delivery.id).toBe('delivery-123');
      expect(delivery.status).toBe('success');
      expect(delivery.httpStatus).toBe(200);
    });

    it('should retry a delivery', async () => {
      vi.spyOn(client['http'], 'post').mockImplementation(async () => ({
        data: {
          delivery_id: 'new-delivery-456',
          endpoint_id: 'endpoint-456',
          status: 'pending',
          scheduled_at: '2024-01-01T00:00:00Z',
        },
      }));

      const result = await client.deliveries.retry('delivery-123');

      expect(result.deliveryId).toBe('new-delivery-456');
      expect(result.status).toBe('pending');
    });
  });

  describe('webhooks service', () => {
    let client: WAASClient;

    beforeEach(() => {
      client = new WAASClient({ apiKey: 'test-key' });
    });

    it('should send a webhook', async () => {
      vi.spyOn(client['http'], 'post').mockImplementation(async () => ({
        data: {
          delivery_id: 'delivery-789',
          endpoint_id: 'endpoint-456',
          status: 'pending',
          scheduled_at: '2024-01-01T00:00:00Z',
        },
      }));

      const result = await client.webhooks.send({
        endpointId: 'endpoint-456',
        payload: { event: 'test.event', data: { message: 'Hello!' } },
      });

      expect(result.deliveryId).toBe('delivery-789');
      expect(result.status).toBe('pending');
    });
  });
});

describe('Error handling', () => {
  describe('WAASAuthenticationError', () => {
    it('should have correct status code', () => {
      const error = new WAASAuthenticationError();
      expect(error.statusCode).toBe(401);
      expect(error.code).toBe('AUTHENTICATION_FAILED');
    });
  });

  describe('WAASNotFoundError', () => {
    it('should have correct status code', () => {
      const error = new WAASNotFoundError('Resource not found');
      expect(error.statusCode).toBe(404);
      expect(error.message).toBe('Resource not found');
    });
  });

  describe('WAASRateLimitError', () => {
    it('should include retry after', () => {
      const error = new WAASRateLimitError('Rate limit exceeded', 60);
      expect(error.statusCode).toBe(429);
      expect(error.retryAfter).toBe(60);
    });
  });
});

describe('Case conversion', () => {
  it('should convert snake_case to camelCase in responses', async () => {
    const client = new WAASClient({ apiKey: 'test-key' });
    vi.spyOn(client['http'], 'get').mockImplementation(async () => ({
      data: {
        id: 'test-id',
        endpoint_id: 'endpoint-123',
        payload_hash: 'hash-456',
        payload_size: 100,
        http_status: 200,
        response_body: 'body',
        attempt_number: 1,
        scheduled_at: '2024-01-01T00:00:00Z',
        delivered_at: '2024-01-01T00:00:01Z',
        created_at: '2024-01-01T00:00:00Z',
        status: 'success',
      },
    }));

    const delivery = await client.deliveries.get('test-id');

    // Should have camelCase properties
    expect(delivery.endpointId).toBe('endpoint-123');
    expect(delivery.payloadHash).toBe('hash-456');
    expect(delivery.payloadSize).toBe(100);
    expect(delivery.httpStatus).toBe(200);
    expect(delivery.responseBody).toBe('body');
    expect(delivery.attemptNumber).toBe(1);
  });
});
