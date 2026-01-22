/**
 * WAAS SDK Client implementation
 */

import axios, { AxiosInstance, AxiosError } from 'axios';
import {
  WAASClientConfig,
  WebhookEndpoint,
  CreateEndpointOptions,
  UpdateEndpointOptions,
  DeliveryAttempt,
  SendWebhookOptions,
  SendWebhookResponse,
  BatchSendOptions,
  BatchSendResponse,
  Tenant,
  AnalyticsSummary,
  QuotaUsage,
  TimeSeriesDataPoint,
  AnalyticsFilters,
  Transformation,
  CreateTransformationOptions,
  TestTransformationOptions,
  TestTransformationResponse,
  TestWebhookOptions,
  TestWebhookResponse,
  PaginatedResponse,
  ListOptions,
  ListDeliveriesOptions,
  ApiError,
} from './types';
import {
  WAASApiError,
  WAASAuthenticationError,
  WAASRateLimitError,
  WAASValidationError,
  WAASNotFoundError,
  WAASConnectionError,
  WAASTimeoutError,
} from './errors';

const DEFAULT_BASE_URL = 'https://api.waas-platform.com/api/v1';
const DEFAULT_TIMEOUT = 30000;

// Helper to convert camelCase to snake_case
function toSnakeCase(obj: Record<string, unknown>): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(obj)) {
    const snakeKey = key.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`);
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      result[snakeKey] = toSnakeCase(value as Record<string, unknown>);
    } else {
      result[snakeKey] = value;
    }
  }
  return result;
}

// Helper to convert snake_case to camelCase
function toCamelCase<T>(obj: Record<string, unknown>): T {
  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(obj)) {
    const camelKey = key.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      result[camelKey] = toCamelCase(value as Record<string, unknown>);
    } else {
      result[camelKey] = value;
    }
  }
  return result as T;
}

/**
 * Main WAAS API Client
 */
export class WAASClient {
  private readonly http: AxiosInstance;

  public readonly endpoints: EndpointsService;
  public readonly deliveries: DeliveriesService;
  public readonly webhooks: WebhooksService;
  public readonly analytics: AnalyticsService;
  public readonly transformations: TransformationsService;
  public readonly testing: TestingService;
  public readonly tenant: TenantService;

  constructor(config: WAASClientConfig) {
    this.http = axios.create({
      baseURL: config.baseUrl || DEFAULT_BASE_URL,
      timeout: config.timeout || DEFAULT_TIMEOUT,
      headers: {
        'X-API-Key': config.apiKey,
        'Content-Type': 'application/json',
        'User-Agent': 'waas-sdk-node/1.0.0',
      },
    });

    this.http.interceptors.response.use(
      (response) => response,
      (error: AxiosError<ApiError>) => {
        throw this.handleError(error);
      }
    );

    // Initialize services
    this.endpoints = new EndpointsService(this.http);
    this.deliveries = new DeliveriesService(this.http);
    this.webhooks = new WebhooksService(this.http);
    this.analytics = new AnalyticsService(this.http);
    this.transformations = new TransformationsService(this.http);
    this.testing = new TestingService(this.http);
    this.tenant = new TenantService(this.http);
  }

  private handleError(error: AxiosError<ApiError>): Error {
    if (error.code === 'ECONNREFUSED' || error.code === 'ENOTFOUND') {
      return new WAASConnectionError(error.message);
    }
    if (error.code === 'ECONNABORTED' || error.code === 'ETIMEDOUT') {
      return new WAASTimeoutError(error.message);
    }

    const status = error.response?.status;
    const data = error.response?.data;

    switch (status) {
      case 401:
        return new WAASAuthenticationError(data?.message);
      case 404:
        return new WAASNotFoundError(data?.message);
      case 429:
        const retryAfter = error.response?.headers['retry-after'];
        return new WAASRateLimitError(
          data?.message,
          retryAfter ? parseInt(retryAfter, 10) : undefined
        );
      case 400:
        return new WAASValidationError(
          data?.message || 'Validation error',
          data?.details
        );
      default:
        return new WAASApiError(
          data?.message || error.message,
          status || 500,
          data?.code || 'UNKNOWN_ERROR',
          data?.details
        );
    }
  }
}

/**
 * Endpoints Service
 */
class EndpointsService {
  constructor(private readonly http: AxiosInstance) {}

  async list(options?: ListOptions): Promise<PaginatedResponse<WebhookEndpoint>> {
    const { data } = await this.http.get('/webhooks/endpoints', {
      params: { page: options?.page || 1, per_page: options?.perPage || 20 },
    });
    return {
      data: data.data.map((ep: Record<string, unknown>) => toCamelCase<WebhookEndpoint>(ep)),
      total: data.total,
      page: data.page,
      perPage: data.per_page,
      totalPages: data.total_pages,
    };
  }

  async get(endpointId: string): Promise<WebhookEndpoint> {
    const { data } = await this.http.get(`/webhooks/endpoints/${endpointId}`);
    return toCamelCase<WebhookEndpoint>(data);
  }

  async create(options: CreateEndpointOptions): Promise<WebhookEndpoint & { secret: string }> {
    const payload = toSnakeCase(options as unknown as Record<string, unknown>);
    const { data } = await this.http.post('/webhooks/endpoints', payload);
    return toCamelCase(data);
  }

  async update(endpointId: string, options: UpdateEndpointOptions): Promise<WebhookEndpoint> {
    const payload = toSnakeCase(options as unknown as Record<string, unknown>);
    const { data } = await this.http.patch(`/webhooks/endpoints/${endpointId}`, payload);
    return toCamelCase<WebhookEndpoint>(data);
  }

  async delete(endpointId: string): Promise<void> {
    await this.http.delete(`/webhooks/endpoints/${endpointId}`);
  }
}

/**
 * Deliveries Service
 */
class DeliveriesService {
  constructor(private readonly http: AxiosInstance) {}

  async list(options?: ListDeliveriesOptions): Promise<PaginatedResponse<DeliveryAttempt>> {
    const params: Record<string, unknown> = {
      page: options?.page || 1,
      per_page: options?.perPage || 20,
    };
    if (options?.status) params.status = options.status;
    if (options?.endpointId) params.endpoint_id = options.endpointId;

    const { data } = await this.http.get('/webhooks/deliveries', { params });
    return {
      data: data.data.map((d: Record<string, unknown>) => toCamelCase<DeliveryAttempt>(d)),
      total: data.total,
      page: data.page,
      perPage: data.per_page,
      totalPages: data.total_pages,
    };
  }

  async get(deliveryId: string): Promise<DeliveryAttempt> {
    const { data } = await this.http.get(`/webhooks/deliveries/${deliveryId}`);
    return toCamelCase<DeliveryAttempt>(data);
  }

  async retry(deliveryId: string): Promise<SendWebhookResponse> {
    const { data } = await this.http.post(`/webhooks/deliveries/${deliveryId}/retry`);
    return toCamelCase<SendWebhookResponse>(data);
  }
}

/**
 * Webhooks Service
 */
class WebhooksService {
  constructor(private readonly http: AxiosInstance) {}

  async send(options: SendWebhookOptions): Promise<SendWebhookResponse> {
    const payload = toSnakeCase(options as unknown as Record<string, unknown>);
    const { data } = await this.http.post('/webhooks/send', payload);
    return toCamelCase<SendWebhookResponse>(data);
  }

  async sendBatch(options: BatchSendOptions): Promise<BatchSendResponse> {
    const payload = toSnakeCase(options as unknown as Record<string, unknown>);
    const { data } = await this.http.post('/webhooks/send/batch', payload);
    return toCamelCase<BatchSendResponse>(data);
  }
}

/**
 * Analytics Service
 */
class AnalyticsService {
  constructor(private readonly http: AxiosInstance) {}

  async getSummary(filters?: AnalyticsFilters): Promise<AnalyticsSummary> {
    const params = filters ? toSnakeCase(filters as unknown as Record<string, unknown>) : {};
    const { data } = await this.http.get('/analytics/summary', { params });
    return toCamelCase<AnalyticsSummary>(data);
  }

  async getQuotaUsage(): Promise<QuotaUsage> {
    const { data } = await this.http.get('/analytics/quota');
    return toCamelCase<QuotaUsage>(data);
  }

  async getDeliveryTimeseries(filters?: AnalyticsFilters): Promise<TimeSeriesDataPoint[]> {
    const params = filters ? toSnakeCase(filters as unknown as Record<string, unknown>) : {};
    const { data } = await this.http.get('/analytics/deliveries/timeseries', { params });
    return data.map((dp: Record<string, unknown>) => toCamelCase<TimeSeriesDataPoint>(dp));
  }

  async getSuccessRateTimeseries(filters?: AnalyticsFilters): Promise<TimeSeriesDataPoint[]> {
    const params = filters ? toSnakeCase(filters as unknown as Record<string, unknown>) : {};
    const { data } = await this.http.get('/analytics/success-rate/timeseries', { params });
    return data.map((dp: Record<string, unknown>) => toCamelCase<TimeSeriesDataPoint>(dp));
  }
}

/**
 * Transformations Service
 */
class TransformationsService {
  constructor(private readonly http: AxiosInstance) {}

  async list(options?: ListOptions): Promise<Transformation[]> {
    const { data } = await this.http.get('/transformations', {
      params: { page: options?.page || 1, per_page: options?.perPage || 20 },
    });
    return data.map((t: Record<string, unknown>) => toCamelCase<Transformation>(t));
  }

  async get(transformationId: string): Promise<Transformation> {
    const { data } = await this.http.get(`/transformations/${transformationId}`);
    return toCamelCase<Transformation>(data);
  }

  async create(options: CreateTransformationOptions): Promise<Transformation> {
    const payload = toSnakeCase(options as unknown as Record<string, unknown>);
    const { data } = await this.http.post('/transformations', payload);
    return toCamelCase<Transformation>(data);
  }

  async delete(transformationId: string): Promise<void> {
    await this.http.delete(`/transformations/${transformationId}`);
  }

  async test(options: TestTransformationOptions): Promise<TestTransformationResponse> {
    const payload = toSnakeCase(options as unknown as Record<string, unknown>);
    const { data } = await this.http.post('/transformations/test', payload);
    return toCamelCase<TestTransformationResponse>(data);
  }
}

/**
 * Testing Service
 */
class TestingService {
  constructor(private readonly http: AxiosInstance) {}

  async testWebhook(options: TestWebhookOptions): Promise<TestWebhookResponse> {
    const payload = toSnakeCase(options as unknown as Record<string, unknown>);
    const { data } = await this.http.post('/testing/webhook', payload);
    return toCamelCase<TestWebhookResponse>(data);
  }
}

/**
 * Tenant Service
 */
class TenantService {
  constructor(private readonly http: AxiosInstance) {}

  async getCurrent(): Promise<Tenant> {
    const { data } = await this.http.get('/tenants/me');
    return toCamelCase<Tenant>(data);
  }
}
