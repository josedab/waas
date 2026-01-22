/**
 * Type definitions for WAAS SDK
 */

export type DeliveryStatus = 'pending' | 'processing' | 'success' | 'failed' | 'retrying';

export interface RetryConfiguration {
  maxAttempts: number;
  initialDelayMs: number;
  maxDelayMs: number;
  backoffMultiplier: number;
}

export interface WebhookEndpoint {
  id: string;
  tenantId: string;
  url: string;
  isActive: boolean;
  retryConfig: RetryConfiguration;
  customHeaders: Record<string, string>;
  createdAt: string;
  updatedAt: string;
}

export interface CreateEndpointOptions {
  url: string;
  customHeaders?: Record<string, string>;
  retryConfig?: Partial<RetryConfiguration>;
}

export interface UpdateEndpointOptions {
  url?: string;
  isActive?: boolean;
  customHeaders?: Record<string, string>;
  retryConfig?: Partial<RetryConfiguration>;
}

export interface DeliveryAttempt {
  id: string;
  endpointId: string;
  payloadHash: string;
  payloadSize: number;
  status: DeliveryStatus;
  httpStatus?: number;
  responseBody?: string;
  errorMessage?: string;
  attemptNumber: number;
  scheduledAt: string;
  deliveredAt?: string;
  createdAt: string;
}

export interface SendWebhookOptions {
  endpointId?: string;
  payload: unknown;
  headers?: Record<string, string>;
}

export interface SendWebhookResponse {
  deliveryId: string;
  endpointId: string;
  status: string;
  scheduledAt: string;
}

export interface BatchSendOptions {
  endpointIds?: string[];
  payload: unknown;
  headers?: Record<string, string>;
}

export interface BatchSendResponse {
  deliveries: SendWebhookResponse[];
  total: number;
  queued: number;
  failed: number;
}

export interface Tenant {
  id: string;
  name: string;
  subscriptionTier: string;
  monthlyQuota: number;
  rateLimitPerMinute: number;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface AnalyticsSummary {
  totalDeliveries: number;
  successfulDeliveries: number;
  failedDeliveries: number;
  successRate: number;
  avgLatencyMs: number;
  p95LatencyMs: number;
  p99LatencyMs: number;
}

export interface QuotaUsage {
  id: string;
  tenantId: string;
  month: string;
  requestCount: number;
  successCount: number;
  failureCount: number;
  overageCount: number;
}

export interface TimeSeriesDataPoint {
  timestamp: string;
  value: number;
}

export interface AnalyticsFilters {
  startDate?: string;
  endDate?: string;
  endpointId?: string;
  interval?: 'hour' | 'day' | 'week' | 'month';
}

export interface TransformConfig {
  timeoutMs: number;
  maxMemoryMb: number;
  allowHttp: boolean;
  enableLogging: boolean;
}

export interface Transformation {
  id: string;
  tenantId: string;
  name: string;
  description?: string;
  script: string;
  enabled: boolean;
  version: number;
  config: TransformConfig;
  createdAt: string;
  updatedAt: string;
}

export interface CreateTransformationOptions {
  name: string;
  script: string;
  description?: string;
  enabled?: boolean;
  config?: Partial<TransformConfig>;
}

export interface TestTransformationOptions {
  script: string;
  inputPayload: unknown;
}

export interface TestTransformationResponse {
  success: boolean;
  outputPayload?: unknown;
  error?: string;
  executionTimeMs: number;
  logs: string[];
}

export interface TestWebhookOptions {
  url: string;
  payload: unknown;
  headers?: Record<string, string>;
  method?: string;
  timeout?: number;
}

export interface TestWebhookResponse {
  testId: string;
  url: string;
  status: string;
  httpStatus?: number;
  responseBody?: string;
  errorMessage?: string;
  latencyMs?: number;
  requestId: string;
  testedAt: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  perPage: number;
  totalPages: number;
}

export interface ListOptions {
  page?: number;
  perPage?: number;
}

export interface ListDeliveriesOptions extends ListOptions {
  status?: DeliveryStatus;
  endpointId?: string;
}

export interface WAASClientConfig {
  apiKey: string;
  baseUrl?: string;
  timeout?: number;
}

export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}
