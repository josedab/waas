import apiClient from './api';
import { AnalyticsSummary, TimeSeriesDataPoint, QuotaUsage } from '@/types';

export interface AnalyticsFilters {
  start_date?: string;
  end_date?: string;
  endpoint_id?: string;
  interval?: 'hour' | 'day' | 'week' | 'month';
  [key: string]: string | undefined;
}

export const analyticsService = {
  async getSummary(filters?: AnalyticsFilters): Promise<AnalyticsSummary> {
    return apiClient.get('/analytics/summary', filters as Record<string, unknown>);
  },

  async getDeliveryTimeSeries(filters?: AnalyticsFilters): Promise<TimeSeriesDataPoint[]> {
    return apiClient.get('/analytics/deliveries/timeseries', filters as Record<string, unknown>);
  },

  async getSuccessRateTimeSeries(filters?: AnalyticsFilters): Promise<TimeSeriesDataPoint[]> {
    return apiClient.get('/analytics/success-rate/timeseries', filters as Record<string, unknown>);
  },

  async getLatencyTimeSeries(filters?: AnalyticsFilters): Promise<TimeSeriesDataPoint[]> {
    return apiClient.get('/analytics/latency/timeseries', filters as Record<string, unknown>);
  },

  async getQuotaUsage(): Promise<QuotaUsage> {
    return apiClient.get('/analytics/quota');
  },

  async getTopEndpoints(limit = 10): Promise<{
    endpoint_id: string;
    url: string;
    total_deliveries: number;
    success_rate: number;
  }[]> {
    return apiClient.get('/analytics/endpoints/top', { limit });
  },

  async getFailureReasons(): Promise<{ reason: string; count: number; percentage: number }[]> {
    return apiClient.get('/analytics/failures/reasons');
  },

  async exportData(
    format: 'csv' | 'json',
    filters?: AnalyticsFilters
  ): Promise<Blob> {
    const response = await fetch(
      `/api/v1/analytics/export?format=${format}&${new URLSearchParams(filters as Record<string, string>)}`,
      {
        headers: {
          'X-API-Key': localStorage.getItem('waas_api_key') || '',
        },
      }
    );
    return response.blob();
  },
};

export default analyticsService;
