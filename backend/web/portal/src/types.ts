export interface PortalTheme {
  /** Primary brand color */
  primaryColor?: string;
  /** Background color */
  backgroundColor?: string;
  /** Surface/card color */
  surfaceColor?: string;
  /** Text color */
  textColor?: string;
  /** Muted text color */
  mutedColor?: string;
  /** Success color */
  successColor?: string;
  /** Error color */
  errorColor?: string;
  /** Warning color */
  warningColor?: string;
  /** Border radius */
  borderRadius?: string;
  /** Font family */
  fontFamily?: string;
}

export interface PortalConfig {
  /** WaaS API base URL */
  apiUrl: string;
  /** Embed token for authentication */
  token: string;
  /** Features to enable */
  features?: {
    endpoints?: boolean;
    deliveries?: boolean;
    testSender?: boolean;
    analytics?: boolean;
  };
  /** Maximum endpoints allowed */
  maxEndpoints?: number;
  /** Polling interval for delivery logs (ms) */
  pollInterval?: number;
}

export interface WaaSPortalProps {
  config: PortalConfig;
  theme?: PortalTheme;
  /** Custom CSS class for the root container */
  className?: string;
  /** Callback when an endpoint is created */
  onEndpointCreated?: (endpoint: Endpoint) => void;
  /** Callback when an endpoint is deleted */
  onEndpointDeleted?: (id: string) => void;
  /** Callback on errors */
  onError?: (error: Error) => void;
}

export interface Endpoint {
  id: string;
  url: string;
  is_active: boolean;
  custom_headers?: Record<string, string>;
  retry_config?: {
    max_attempts: number;
    initial_delay_ms: number;
    max_delay_ms: number;
  };
  created_at: string;
  updated_at: string;
}

export interface Delivery {
  id: string;
  endpoint_id: string;
  status: 'pending' | 'delivered' | 'failed' | 'retrying';
  attempt_count: number;
  last_http_status?: number;
  last_error?: string;
  created_at: string;
  last_attempt_at?: string;
}

export interface ApiClient {
  listEndpoints(): Promise<Endpoint[]>;
  createEndpoint(url: string, headers?: Record<string, string>): Promise<Endpoint>;
  deleteEndpoint(id: string): Promise<void>;
  updateEndpoint(id: string, updates: Partial<Endpoint>): Promise<Endpoint>;
  listDeliveries(endpointId?: string, limit?: number): Promise<Delivery[]>;
  sendTestWebhook(endpointId: string, payload: string): Promise<{ delivery_id: string }>;
}
