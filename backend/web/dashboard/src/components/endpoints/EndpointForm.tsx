import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { endpointsService } from '@/services';
import { WebhookEndpoint, CreateEndpointRequest, UpdateEndpointRequest } from '@/types';
import { useUIStore } from '@/store';

interface EndpointFormProps {
  endpoint?: WebhookEndpoint;
  onSuccess: () => void;
  onCancel: () => void;
}

export function EndpointForm({ endpoint, onSuccess, onCancel }: EndpointFormProps) {
  const isEditing = !!endpoint;
  const { addNotification } = useUIStore();

  const [formData, setFormData] = useState({
    url: endpoint?.url || '',
    isActive: endpoint?.is_active ?? true,
    maxAttempts: endpoint?.retry_config.max_attempts || 5,
    initialDelayMs: endpoint?.retry_config.initial_delay_ms || 1000,
    maxDelayMs: endpoint?.retry_config.max_delay_ms || 300000,
    backoffMultiplier: endpoint?.retry_config.backoff_multiplier || 2,
    customHeaders: JSON.stringify(endpoint?.custom_headers || {}, null, 2),
  });

  const [secret, setSecret] = useState<string | null>(null);

  const createMutation = useMutation({
    mutationFn: (data: CreateEndpointRequest) => endpointsService.create(data),
    onSuccess: (result) => {
      setSecret(result.secret);
      addNotification('success', 'Endpoint created successfully');
    },
    onError: (error: Error) => {
      addNotification('error', error.message);
    },
  });

  const updateMutation = useMutation({
    mutationFn: (data: UpdateEndpointRequest) => endpointsService.update(endpoint!.id, data),
    onSuccess: () => {
      addNotification('success', 'Endpoint updated successfully');
      onSuccess();
    },
    onError: (error: Error) => {
      addNotification('error', error.message);
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    let customHeaders: Record<string, string> = {};
    try {
      customHeaders = JSON.parse(formData.customHeaders);
    } catch {
      addNotification('error', 'Invalid JSON in custom headers');
      return;
    }

    if (isEditing) {
      updateMutation.mutate({
        url: formData.url,
        is_active: formData.isActive,
        custom_headers: customHeaders,
        retry_config: {
          max_attempts: formData.maxAttempts,
          initial_delay_ms: formData.initialDelayMs,
          max_delay_ms: formData.maxDelayMs,
          backoff_multiplier: formData.backoffMultiplier,
        },
      });
    } else {
      createMutation.mutate({
        url: formData.url,
        custom_headers: customHeaders,
        retry_config: {
          max_attempts: formData.maxAttempts,
          initial_delay_ms: formData.initialDelayMs,
          max_delay_ms: formData.maxDelayMs,
          backoff_multiplier: formData.backoffMultiplier,
        },
      });
    }
  };

  if (secret) {
    return (
      <div className="space-y-4">
        <div className="p-4 bg-green-50 rounded-lg">
          <h4 className="font-medium text-green-800">Endpoint Created Successfully!</h4>
          <p className="mt-2 text-sm text-green-700">
            Save this secret key securely. It won't be shown again.
          </p>
        </div>
        <div className="p-4 bg-gray-100 rounded-lg">
          <label className="label">Signing Secret</label>
          <code className="block p-2 bg-white rounded border text-sm font-mono break-all">
            {secret}
          </code>
        </div>
        <div className="flex justify-end">
          <button onClick={onSuccess} className="btn-primary">
            Done
          </button>
        </div>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="label">Webhook URL</label>
        <input
          type="url"
          className="input"
          placeholder="https://your-server.com/webhook"
          value={formData.url}
          onChange={(e) => setFormData({ ...formData, url: e.target.value })}
          required
        />
      </div>

      {isEditing && (
        <div className="flex items-center">
          <input
            type="checkbox"
            id="isActive"
            checked={formData.isActive}
            onChange={(e) => setFormData({ ...formData, isActive: e.target.checked })}
            className="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <label htmlFor="isActive" className="ml-2 text-sm text-gray-700">
            Endpoint is active
          </label>
        </div>
      )}

      <div className="border-t pt-4">
        <h4 className="font-medium text-gray-900 mb-3">Retry Configuration</h4>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="label">Max Attempts</label>
            <input
              type="number"
              className="input"
              min="1"
              max="10"
              value={formData.maxAttempts}
              onChange={(e) => setFormData({ ...formData, maxAttempts: parseInt(e.target.value) })}
            />
          </div>
          <div>
            <label className="label">Initial Delay (ms)</label>
            <input
              type="number"
              className="input"
              min="100"
              value={formData.initialDelayMs}
              onChange={(e) => setFormData({ ...formData, initialDelayMs: parseInt(e.target.value) })}
            />
          </div>
          <div>
            <label className="label">Max Delay (ms)</label>
            <input
              type="number"
              className="input"
              min="1000"
              value={formData.maxDelayMs}
              onChange={(e) => setFormData({ ...formData, maxDelayMs: parseInt(e.target.value) })}
            />
          </div>
          <div>
            <label className="label">Backoff Multiplier</label>
            <input
              type="number"
              className="input"
              min="1"
              max="5"
              value={formData.backoffMultiplier}
              onChange={(e) => setFormData({ ...formData, backoffMultiplier: parseInt(e.target.value) })}
            />
          </div>
        </div>
      </div>

      <div>
        <label className="label">Custom Headers (JSON)</label>
        <textarea
          className="input font-mono text-sm"
          rows={3}
          placeholder='{"X-Custom-Header": "value"}'
          value={formData.customHeaders}
          onChange={(e) => setFormData({ ...formData, customHeaders: e.target.value })}
        />
      </div>

      <div className="flex justify-end gap-3 pt-4">
        <button type="button" onClick={onCancel} className="btn-secondary">
          Cancel
        </button>
        <button
          type="submit"
          className="btn-primary"
          disabled={createMutation.isPending || updateMutation.isPending}
        >
          {createMutation.isPending || updateMutation.isPending
            ? 'Saving...'
            : isEditing
            ? 'Update Endpoint'
            : 'Create Endpoint'}
        </button>
      </div>
    </form>
  );
}

export default EndpointForm;
