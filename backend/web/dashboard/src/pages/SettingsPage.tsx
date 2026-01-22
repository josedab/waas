import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useAuthStore, useUIStore } from '@/store';
import { apiClient } from '@/services';

export function SettingsPage() {
  const { tenant, logout } = useAuthStore();
  const { addNotification } = useUIStore();

  const [showRegenerateConfirm, setShowRegenerateConfirm] = useState(false);

  const regenerateKeyMutation = useMutation({
    mutationFn: async () => {
      const result = await apiClient.post<{ api_key: string }>('/tenants/me/regenerate-key');
      return result;
    },
    onSuccess: (data) => {
      addNotification('success', 'API key regenerated. Please save the new key.');
      alert(`New API Key: ${data.api_key}\n\nPlease save this key securely. You will need to use it to log in again.`);
      logout();
    },
    onError: (error: Error) => {
      addNotification('error', error.message);
    },
  });

  return (
    <div>
      <div className="sm:flex sm:items-center">
        <div className="sm:flex-auto">
          <h1 className="text-2xl font-semibold leading-6 text-gray-900">Settings</h1>
          <p className="mt-2 text-sm text-gray-700">
            Manage your account settings and subscription.
          </p>
        </div>
      </div>

      <div className="mt-8 space-y-8">
        {/* Account Information */}
        <div className="card p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Account Information</h3>
          <dl className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <dt className="text-sm font-medium text-gray-500">Tenant Name</dt>
              <dd className="mt-1 text-sm text-gray-900">{tenant?.name}</dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">Tenant ID</dt>
              <dd className="mt-1 text-sm font-mono text-gray-900">{tenant?.id}</dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">Status</dt>
              <dd className="mt-1">
                <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                  tenant?.is_active ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                }`}>
                  {tenant?.is_active ? 'Active' : 'Inactive'}
                </span>
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">Created</dt>
              <dd className="mt-1 text-sm text-gray-900">
                {tenant?.created_at ? new Date(tenant.created_at).toLocaleDateString() : '-'}
              </dd>
            </div>
          </dl>
        </div>

        {/* Subscription */}
        <div className="card p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Subscription</h3>
          <dl className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div>
              <dt className="text-sm font-medium text-gray-500">Plan</dt>
              <dd className="mt-1 text-sm text-gray-900 capitalize">
                {tenant?.subscription_tier || 'Free'}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">Monthly Quota</dt>
              <dd className="mt-1 text-sm text-gray-900">
                {tenant?.monthly_quota?.toLocaleString() || 0} requests
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">Rate Limit</dt>
              <dd className="mt-1 text-sm text-gray-900">
                {tenant?.rate_limit_per_minute?.toLocaleString() || 0} req/min
              </dd>
            </div>
          </dl>
          <div className="mt-4">
            <button className="btn-secondary">
              Upgrade Plan
            </button>
          </div>
        </div>

        {/* API Key Management */}
        <div className="card p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">API Key</h3>
          <p className="text-sm text-gray-500 mb-4">
            Your API key is used to authenticate requests to the WAAS API.
            Keep it secure and never share it publicly.
          </p>
          
          {showRegenerateConfirm ? (
            <div className="p-4 bg-yellow-50 rounded-lg">
              <p className="text-sm text-yellow-800 mb-4">
                <strong>Warning:</strong> Regenerating your API key will invalidate your current key.
                You will need to update any applications using the old key.
              </p>
              <div className="flex gap-3">
                <button
                  onClick={() => setShowRegenerateConfirm(false)}
                  className="btn-secondary"
                >
                  Cancel
                </button>
                <button
                  onClick={() => regenerateKeyMutation.mutate()}
                  disabled={regenerateKeyMutation.isPending}
                  className="btn-danger"
                >
                  {regenerateKeyMutation.isPending ? 'Regenerating...' : 'Confirm Regenerate'}
                </button>
              </div>
            </div>
          ) : (
            <button
              onClick={() => setShowRegenerateConfirm(true)}
              className="btn-secondary"
            >
              Regenerate API Key
            </button>
          )}
        </div>

        {/* Danger Zone */}
        <div className="card p-6 border-red-200">
          <h3 className="text-lg font-medium text-red-600 mb-4">Danger Zone</h3>
          <p className="text-sm text-gray-500 mb-4">
            Permanently delete your account and all associated data.
            This action cannot be undone.
          </p>
          <button className="btn-danger">
            Delete Account
          </button>
        </div>
      </div>
    </div>
  );
}

export default SettingsPage;
