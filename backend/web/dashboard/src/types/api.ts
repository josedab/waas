// API Types matching the Go backend models

export interface Tenant {
  id: string;
  name: string;
  subscription_tier: 'free' | 'starter' | 'professional' | 'enterprise';
  monthly_quota: number;
  rate_limit_per_minute: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface WebhookEndpoint {
  id: string;
  tenant_id: string;
  url: string;
  is_active: boolean;
  retry_config: RetryConfiguration;
  custom_headers: Record<string, string>;
  created_at: string;
  updated_at: string;
}

export interface RetryConfiguration {
  max_attempts: number;
  initial_delay_ms: number;
  max_delay_ms: number;
  backoff_multiplier: number;
}

export interface DeliveryAttempt {
  id: string;
  endpoint_id: string;
  payload_hash: string;
  payload_size: number;
  status: DeliveryStatus;
  http_status?: number;
  response_body?: string;
  error_message?: string;
  attempt_number: number;
  scheduled_at: string;
  delivered_at?: string;
  created_at: string;
}

export type DeliveryStatus = 'pending' | 'processing' | 'success' | 'failed' | 'retrying';

export interface QuotaUsage {
  id: string;
  tenant_id: string;
  month: string;
  request_count: number;
  success_count: number;
  failure_count: number;
  overage_count: number;
  last_updated: string;
}

export interface AnalyticsSummary {
  total_deliveries: number;
  successful_deliveries: number;
  failed_deliveries: number;
  success_rate: number;
  avg_latency_ms: number;
  p95_latency_ms: number;
  p99_latency_ms: number;
}

export interface TimeSeriesDataPoint {
  timestamp: string;
  value: number;
}

export interface EndpointStats {
  endpoint_id: string;
  url: string;
  total_attempts: number;
  success_count: number;
  failure_count: number;
  success_rate: number;
  avg_latency_ms: number;
  last_delivery_at?: string;
}

export interface CreateEndpointRequest {
  url: string;
  custom_headers?: Record<string, string>;
  retry_config?: Partial<RetryConfiguration>;
}

export interface UpdateEndpointRequest {
  url?: string;
  custom_headers?: Record<string, string>;
  retry_config?: Partial<RetryConfiguration>;
  is_active?: boolean;
}

export interface SendWebhookRequest {
  endpoint_id?: string;
  payload: unknown;
  headers?: Record<string, string>;
}

export interface SendWebhookResponse {
  delivery_id: string;
  endpoint_id: string;
  status: string;
  scheduled_at: string;
}

export interface TestWebhookRequest {
  url: string;
  payload: unknown;
  headers?: Record<string, string>;
  method?: string;
  timeout?: number;
}

export interface TestWebhookResponse {
  test_id: string;
  url: string;
  status: string;
  http_status?: number;
  response_body?: string;
  error_message?: string;
  latency_ms?: number;
  request_id: string;
  tested_at: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

// Transformation types (Feature 2)
export interface Transformation {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  script: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface TransformationTestRequest {
  script: string;
  input_payload: unknown;
}

export interface TransformationTestResponse {
  success: boolean;
  output_payload?: unknown;
  error?: string;
  execution_time_ms: number;
}
