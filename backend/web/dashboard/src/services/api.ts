import axios, { AxiosInstance, AxiosError } from 'axios';
import { ApiError } from '@/types';

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

class ApiClient {
  private client: AxiosInstance;
  private apiKey: string | null = null;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    this.client.interceptors.request.use((config) => {
      if (this.apiKey) {
        config.headers['X-API-Key'] = this.apiKey;
      }
      return config;
    });

    this.client.interceptors.response.use(
      (response) => response,
      (error: AxiosError<ApiError>) => {
        if (error.response?.status === 401) {
          this.clearApiKey();
          window.location.href = '/login';
        }
        return Promise.reject(this.formatError(error));
      }
    );
  }

  setApiKey(key: string) {
    this.apiKey = key;
    localStorage.setItem('waas_api_key', key);
  }

  clearApiKey() {
    this.apiKey = null;
    localStorage.removeItem('waas_api_key');
  }

  loadApiKey() {
    const key = localStorage.getItem('waas_api_key');
    if (key) {
      this.apiKey = key;
    }
    return key;
  }

  isAuthenticated(): boolean {
    return !!this.apiKey;
  }

  private formatError(error: AxiosError<ApiError>): Error {
    if (error.response?.data) {
      const apiError = error.response.data;
      return new Error(apiError.message || 'An error occurred');
    }
    return new Error(error.message || 'Network error');
  }

  async get<T>(url: string, params?: Record<string, unknown>): Promise<T> {
    const response = await this.client.get<T>(url, { params });
    return response.data;
  }

  async post<T>(url: string, data?: unknown): Promise<T> {
    const response = await this.client.post<T>(url, data);
    return response.data;
  }

  async put<T>(url: string, data?: unknown): Promise<T> {
    const response = await this.client.put<T>(url, data);
    return response.data;
  }

  async patch<T>(url: string, data?: unknown): Promise<T> {
    const response = await this.client.patch<T>(url, data);
    return response.data;
  }

  async delete<T>(url: string): Promise<T> {
    const response = await this.client.delete<T>(url);
    return response.data;
  }
}

export const apiClient = new ApiClient();
export default apiClient;
