import apiClient from './api';
import { TestWebhookRequest, TestWebhookResponse } from '@/types';

export interface TestEndpoint {
  id: string;
  url: string;
  name?: string;
  description?: string;
  headers?: Record<string, string>;
  created_at: string;
  expires_at: string;
}

export interface ReceivedWebhook {
  id: string;
  test_endpoint_id: string;
  method: string;
  headers: Record<string, string>;
  body: string;
  received_at: string;
}

export const testingService = {
  async testWebhook(data: TestWebhookRequest): Promise<TestWebhookResponse> {
    return apiClient.post('/testing/webhook', data);
  },

  async createTestEndpoint(data: {
    name?: string;
    description?: string;
    headers?: Record<string, string>;
    ttl?: number;
  }): Promise<TestEndpoint> {
    return apiClient.post('/testing/endpoints', data);
  },

  async getTestEndpoint(id: string): Promise<TestEndpoint> {
    return apiClient.get(`/testing/endpoints/${id}`);
  },

  async listTestEndpoints(): Promise<TestEndpoint[]> {
    return apiClient.get('/testing/endpoints');
  },

  async deleteTestEndpoint(id: string): Promise<void> {
    return apiClient.delete(`/testing/endpoints/${id}`);
  },

  async getReceivedWebhooks(testEndpointId: string): Promise<ReceivedWebhook[]> {
    return apiClient.get(`/testing/endpoints/${testEndpointId}/webhooks`);
  },

  async inspectDelivery(deliveryId: string): Promise<{
    delivery_id: string;
    endpoint_id: string;
    status: string;
    attempt_number: number;
    request: {
      method: string;
      url: string;
      headers: Record<string, string>;
      body: string;
    };
    response?: {
      status_code: number;
      headers: Record<string, string>;
      body: string;
      latency_ms: number;
    };
    timeline: { timestamp: string; event: string; details?: string }[];
    error_details?: {
      type: string;
      message: string;
      suggestion: string;
    };
  }> {
    return apiClient.get(`/testing/inspect/${deliveryId}`);
  },
};

export default testingService;
