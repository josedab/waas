import type { PortalConfig, ApiClient, Endpoint, Delivery } from './types';

export function createApiClient(config: PortalConfig): ApiClient {
  const { apiUrl, token } = config;

  async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const res = await fetch(`${apiUrl}${path}`, {
      method,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!res.ok) {
      const errBody = await res.text();
      throw new Error(`API error ${res.status}: ${errBody}`);
    }

    if (res.status === 204) return undefined as T;
    return res.json();
  }

  return {
    async listEndpoints(): Promise<Endpoint[]> {
      const result = await request<{ endpoints: Endpoint[] }>('GET', '/api/v1/webhooks/endpoints');
      return result.endpoints || [];
    },

    async createEndpoint(url: string, headers?: Record<string, string>): Promise<Endpoint> {
      return request<Endpoint>('POST', '/api/v1/webhooks/endpoints', {
        url,
        custom_headers: headers,
      });
    },

    async deleteEndpoint(id: string): Promise<void> {
      await request<void>('DELETE', `/api/v1/webhooks/endpoints/${id}`);
    },

    async updateEndpoint(id: string, updates: Partial<Endpoint>): Promise<Endpoint> {
      return request<Endpoint>('PUT', `/api/v1/webhooks/endpoints/${id}`, updates);
    },

    async listDeliveries(endpointId?: string, limit = 20): Promise<Delivery[]> {
      const path = endpointId
        ? `/api/v1/webhooks/endpoints/${endpointId}/deliveries?limit=${limit}`
        : `/api/v1/webhooks/deliveries?limit=${limit}`;
      const result = await request<{ deliveries: Delivery[] }>('GET', path);
      return result.deliveries || [];
    },

    async sendTestWebhook(endpointId: string, payload: string): Promise<{ delivery_id: string }> {
      return request('POST', '/api/v1/webhooks/send', {
        endpoint_id: endpointId,
        payload: JSON.parse(payload),
      });
    },
  };
}
