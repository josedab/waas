import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { Tenant } from '@/types';
import { apiClient } from '@/services';

interface AuthState {
  apiKey: string | null;
  tenant: Tenant | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  login: (apiKey: string) => Promise<void>;
  logout: () => void;
  loadSession: () => Promise<void>;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      apiKey: null,
      tenant: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,

      login: async (apiKey: string) => {
        set({ isLoading: true, error: null });
        try {
          apiClient.setApiKey(apiKey);
          const tenant = await apiClient.get<Tenant>('/tenants/me');
          set({
            apiKey,
            tenant,
            isAuthenticated: true,
            isLoading: false,
          });
        } catch (error) {
          apiClient.clearApiKey();
          set({
            isLoading: false,
            error: error instanceof Error ? error.message : 'Login failed',
          });
          throw error;
        }
      },

      logout: () => {
        apiClient.clearApiKey();
        set({
          apiKey: null,
          tenant: null,
          isAuthenticated: false,
          error: null,
        });
      },

      loadSession: async () => {
        const savedKey = apiClient.loadApiKey();
        if (savedKey) {
          try {
            const tenant = await apiClient.get<Tenant>('/tenants/me');
            set({
              apiKey: savedKey,
              tenant,
              isAuthenticated: true,
            });
          } catch {
            apiClient.clearApiKey();
            set({ apiKey: null, tenant: null, isAuthenticated: false });
          }
        }
      },
    }),
    {
      name: 'waas-auth',
      partialize: (state) => ({ apiKey: state.apiKey }),
    }
  )
);

export default useAuthStore;
