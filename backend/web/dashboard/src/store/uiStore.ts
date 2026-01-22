import { create } from 'zustand';
import { WebhookEndpoint, DeliveryAttempt } from '@/types';

interface UIState {
  // Sidebar
  sidebarOpen: boolean;
  setSidebarOpen: (open: boolean) => void;
  toggleSidebar: () => void;

  // Selected items
  selectedEndpoint: WebhookEndpoint | null;
  setSelectedEndpoint: (endpoint: WebhookEndpoint | null) => void;
  
  selectedDelivery: DeliveryAttempt | null;
  setSelectedDelivery: (delivery: DeliveryAttempt | null) => void;

  // Modals
  modals: {
    createEndpoint: boolean;
    editEndpoint: boolean;
    sendWebhook: boolean;
    testWebhook: boolean;
    deliveryDetails: boolean;
  };
  openModal: (modal: keyof UIState['modals']) => void;
  closeModal: (modal: keyof UIState['modals']) => void;
  closeAllModals: () => void;

  // Notifications
  notifications: Array<{
    id: string;
    type: 'success' | 'error' | 'warning' | 'info';
    message: string;
  }>;
  addNotification: (type: UIState['notifications'][0]['type'], message: string) => void;
  removeNotification: (id: string) => void;
}

export const useUIStore = create<UIState>((set) => ({
  sidebarOpen: true,
  setSidebarOpen: (open) => set({ sidebarOpen: open }),
  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),

  selectedEndpoint: null,
  setSelectedEndpoint: (endpoint) => set({ selectedEndpoint: endpoint }),

  selectedDelivery: null,
  setSelectedDelivery: (delivery) => set({ selectedDelivery: delivery }),

  modals: {
    createEndpoint: false,
    editEndpoint: false,
    sendWebhook: false,
    testWebhook: false,
    deliveryDetails: false,
  },
  openModal: (modal) =>
    set((state) => ({
      modals: { ...state.modals, [modal]: true },
    })),
  closeModal: (modal) =>
    set((state) => ({
      modals: { ...state.modals, [modal]: false },
    })),
  closeAllModals: () =>
    set({
      modals: {
        createEndpoint: false,
        editEndpoint: false,
        sendWebhook: false,
        testWebhook: false,
        deliveryDetails: false,
      },
    }),

  notifications: [],
  addNotification: (type, message) =>
    set((state) => ({
      notifications: [
        ...state.notifications,
        { id: Date.now().toString(), type, message },
      ],
    })),
  removeNotification: (id) =>
    set((state) => ({
      notifications: state.notifications.filter((n) => n.id !== id),
    })),
}));

export default useUIStore;
