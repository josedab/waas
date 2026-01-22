import apiClient from './api';
import {
  WebhookEndpoint,
  CreateEndpointRequest,
  UpdateEndpointRequest,
  DeliveryAttempt,
  PaginatedResponse,
  SendWebhookRequest,
  SendWebhookResponse,
  EndpointStats,
} from '@/types';

export const endpointsService = {
  async list(page = 1, perPage = 20): Promise<PaginatedResponse<WebhookEndpoint>> {
    return apiClient.get('/webhooks/endpoints', { page, per_page: perPage });
  },

  async get(id: string): Promise<WebhookEndpoint> {
    return apiClient.get(`/webhooks/endpoints/${id}`);
  },

  async create(data: CreateEndpointRequest): Promise<WebhookEndpoint & { secret: string }> {
    return apiClient.post('/webhooks/endpoints', data);
  },

  async update(id: string, data: UpdateEndpointRequest): Promise<WebhookEndpoint> {
    return apiClient.patch(`/webhooks/endpoints/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete(`/webhooks/endpoints/${id}`);
  },

  async getStats(id: string): Promise<EndpointStats> {
    return apiClient.get(`/webhooks/endpoints/${id}/stats`);
  },

  async getDeliveries(
    id: string,
    page = 1,
    perPage = 20,
    status?: string
  ): Promise<PaginatedResponse<DeliveryAttempt>> {
    return apiClient.get(`/webhooks/endpoints/${id}/deliveries`, {
      page,
      per_page: perPage,
      status,
    });
  },
};

export const deliveriesService = {
  async list(
    page = 1,
    perPage = 20,
    filters?: { status?: string; endpoint_id?: string; start_date?: string; end_date?: string }
  ): Promise<PaginatedResponse<DeliveryAttempt>> {
    return apiClient.get('/webhooks/deliveries', { page, per_page: perPage, ...filters });
  },

  async get(id: string): Promise<DeliveryAttempt> {
    return apiClient.get(`/webhooks/deliveries/${id}`);
  },

  async retry(id: string): Promise<SendWebhookResponse> {
    return apiClient.post(`/webhooks/deliveries/${id}/retry`);
  },

  async send(data: SendWebhookRequest): Promise<SendWebhookResponse> {
    return apiClient.post('/webhooks/send', data);
  },

  async batchSend(
    endpointIds: string[],
    payload: unknown,
    headers?: Record<string, string>
  ): Promise<{ deliveries: SendWebhookResponse[]; total: number; queued: number; failed: number }> {
    return apiClient.post('/webhooks/send/batch', {
      endpoint_ids: endpointIds,
      payload,
      headers,
    });
  },
};

export default { endpoints: endpointsService, deliveries: deliveriesService };
