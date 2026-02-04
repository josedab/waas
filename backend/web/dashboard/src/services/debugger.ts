import { apiClient } from './api';

export interface DeliveryTrace {
  id: string;
  tenant_id: string;
  delivery_id: string;
  endpoint_id: string;
  endpoint_url: string;
  final_status: string;
  total_duration_ms: number;
  stages: TraceStage[];
  created_at: string;
}

export interface TraceStage {
  name: string;
  status: string;
  input: string;
  output: string;
  error: string;
  duration_ms: number;
  timestamp: string;
  metadata: Record<string, string>;
}

export interface PayloadDiff {
  delivery_id: string;
  identical: boolean;
  diffs: FieldDiff[];
}

export interface FieldDiff {
  path: string;
  type: string;
  old_value: string;
  new_value: string;
}

export interface DebugSession {
  id: string;
  delivery_id: string;
  current_step: number;
  status: string;
  breakpoints: string[];
  created_at: string;
  expires_at: string;
}

export interface ReplayRequest {
  delivery_id: string;
  payload_override?: string;
  header_override?: Record<string, string>;
  endpoint_override?: string;
}

export interface BulkReplayRequest {
  delivery_ids?: string[];
  endpoint_id?: string;
  status_filter?: string;
  dry_run?: boolean;
}

export interface BulkReplayResult {
  total_found: number;
  total_replayed: number;
  total_failed: number;
  dry_run: boolean;
}

export const debuggerService = {
  async listTraces(endpointId?: string, limit = 50, offset = 0): Promise<{ traces: DeliveryTrace[]; total: number }> {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
    if (endpointId) params.set('endpoint_id', endpointId);
    return apiClient.get(`/debugger/traces?${params}`);
  },

  async getTrace(deliveryId: string): Promise<DeliveryTrace> {
    return apiClient.get(`/debugger/traces/${deliveryId}`);
  },

  async getDiff(deliveryId: string): Promise<PayloadDiff> {
    return apiClient.get(`/debugger/traces/${deliveryId}/diff`);
  },

  async replay(request: ReplayRequest): Promise<DeliveryTrace> {
    return apiClient.post('/debugger/replay', request);
  },

  async bulkReplay(request: BulkReplayRequest): Promise<BulkReplayResult> {
    return apiClient.post('/debugger/replay/bulk', request);
  },

  async createDebugSession(deliveryId: string, breakpoints: string[]): Promise<DebugSession> {
    return apiClient.post('/debugger/sessions', { delivery_id: deliveryId, breakpoints });
  },

  async stepDebugSession(sessionId: string): Promise<{ session: DebugSession; stage: TraceStage | null }> {
    return apiClient.post(`/debugger/sessions/${sessionId}/step`, {});
  },

  async getDebugSession(sessionId: string): Promise<DebugSession> {
    return apiClient.get(`/debugger/sessions/${sessionId}`);
  },
};
