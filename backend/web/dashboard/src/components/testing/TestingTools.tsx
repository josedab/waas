import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { BeakerIcon, PlusIcon, TrashIcon } from '@heroicons/react/24/outline';
import { testingService } from '@/services';
import { LoadingSpinner, EmptyState, StatusBadge } from '@/components/common';
import { useUIStore } from '@/store';
import { format } from 'date-fns';

export function TestingTools() {
  const [activeTab, setActiveTab] = useState<'test' | 'endpoints'>('test');

  return (
    <div>
      <div className="sm:flex sm:items-center">
        <div className="sm:flex-auto">
          <h1 className="text-2xl font-semibold leading-6 text-gray-900">Testing Tools</h1>
          <p className="mt-2 text-sm text-gray-700">
            Test webhooks and create temporary endpoints for debugging.
          </p>
        </div>
      </div>

      {/* Tabs */}
      <div className="mt-6 border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setActiveTab('test')}
            className={`py-4 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'test'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Test Webhook
          </button>
          <button
            onClick={() => setActiveTab('endpoints')}
            className={`py-4 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'endpoints'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Test Endpoints
          </button>
        </nav>
      </div>

      <div className="mt-6">
        {activeTab === 'test' ? <WebhookTester /> : <TestEndpointsList />}
      </div>
    </div>
  );
}

function WebhookTester() {
  const { addNotification } = useUIStore();
  const [url, setUrl] = useState('');
  const [payload, setPayload] = useState('{\n  "event": "test",\n  "data": {}\n}');
  const [headers, setHeaders] = useState('{}');
  const [result, setResult] = useState<{
    success: boolean;
    http_status?: number;
    response_body?: string;
    error_message?: string;
    latency_ms?: number;
  } | null>(null);

  const testMutation = useMutation({
    mutationFn: () => {
      let parsedPayload, parsedHeaders;
      try {
        parsedPayload = JSON.parse(payload);
        parsedHeaders = JSON.parse(headers);
      } catch {
        throw new Error('Invalid JSON in payload or headers');
      }
      return testingService.testWebhook({
        url,
        payload: parsedPayload,
        headers: parsedHeaders,
      });
    },
    onSuccess: (data) => {
      setResult({
        success: data.status === 'success',
        http_status: data.http_status,
        response_body: data.response_body,
        error_message: data.error_message,
        latency_ms: data.latency_ms,
      });
      addNotification(
        data.status === 'success' ? 'success' : 'warning',
        `Webhook test completed: ${data.status}`
      );
    },
    onError: (error: Error) => {
      setResult({
        success: false,
        error_message: error.message,
      });
      addNotification('error', error.message);
    },
  });

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      {/* Request Form */}
      <div className="card p-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Test Request</h3>
        <div className="space-y-4">
          <div>
            <label className="label">Webhook URL</label>
            <input
              type="url"
              className="input"
              placeholder="https://your-server.com/webhook"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
            />
          </div>
          <div>
            <label className="label">Payload (JSON)</label>
            <textarea
              className="input font-mono text-sm"
              rows={8}
              value={payload}
              onChange={(e) => setPayload(e.target.value)}
            />
          </div>
          <div>
            <label className="label">Custom Headers (JSON)</label>
            <textarea
              className="input font-mono text-sm"
              rows={3}
              value={headers}
              onChange={(e) => setHeaders(e.target.value)}
            />
          </div>
          <button
            onClick={() => testMutation.mutate()}
            disabled={!url || testMutation.isPending}
            className="btn-primary w-full flex items-center justify-center gap-2"
          >
            {testMutation.isPending ? (
              <>
                <LoadingSpinner size="sm" />
                Testing...
              </>
            ) : (
              <>
                <BeakerIcon className="h-5 w-5" />
                Send Test Webhook
              </>
            )}
          </button>
        </div>
      </div>

      {/* Result */}
      <div className="card p-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Result</h3>
        {result ? (
          <div className="space-y-4">
            <div className="flex items-center gap-3">
              <StatusBadge status={result.success ? 'success' : 'failed'} />
              {result.http_status && (
                <span className="text-sm text-gray-500">HTTP {result.http_status}</span>
              )}
              {result.latency_ms && (
                <span className="text-sm text-gray-500">{result.latency_ms}ms</span>
              )}
            </div>

            {result.error_message && (
              <div className="p-3 bg-red-50 rounded-lg">
                <p className="text-sm text-red-700">{result.error_message}</p>
              </div>
            )}

            {result.response_body && (
              <div>
                <label className="label">Response Body</label>
                <pre className="text-xs bg-gray-50 p-3 rounded-lg border overflow-x-auto max-h-64">
                  {result.response_body}
                </pre>
              </div>
            )}
          </div>
        ) : (
          <div className="text-center py-12 text-gray-500">
            <BeakerIcon className="h-12 w-12 mx-auto mb-4" />
            <p>Send a test webhook to see the result</p>
          </div>
        )}
      </div>
    </div>
  );
}

function TestEndpointsList() {
  const queryClient = useQueryClient();
  const { addNotification } = useUIStore();
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newEndpoint, setNewEndpoint] = useState({ name: '', description: '', ttl: 3600 });

  const { data: endpoints, isLoading } = useQuery({
    queryKey: ['test-endpoints'],
    queryFn: () => testingService.listTestEndpoints(),
  });

  const createMutation = useMutation({
    mutationFn: () => testingService.createTestEndpoint(newEndpoint),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['test-endpoints'] });
      setShowCreateForm(false);
      setNewEndpoint({ name: '', description: '', ttl: 3600 });
      addNotification('success', 'Test endpoint created');
    },
    onError: (error: Error) => {
      addNotification('error', error.message);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => testingService.deleteTestEndpoint(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['test-endpoints'] });
      addNotification('success', 'Test endpoint deleted');
    },
  });

  if (isLoading) return <LoadingSpinner className="py-8" />;

  return (
    <div className="space-y-6">
      {/* Create Form */}
      {showCreateForm ? (
        <div className="card p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Create Test Endpoint</h3>
          <div className="space-y-4">
            <div>
              <label className="label">Name (optional)</label>
              <input
                type="text"
                className="input"
                placeholder="My Test Endpoint"
                value={newEndpoint.name}
                onChange={(e) => setNewEndpoint({ ...newEndpoint, name: e.target.value })}
              />
            </div>
            <div>
              <label className="label">Description (optional)</label>
              <textarea
                className="input"
                rows={2}
                placeholder="Description of this test endpoint"
                value={newEndpoint.description}
                onChange={(e) => setNewEndpoint({ ...newEndpoint, description: e.target.value })}
              />
            </div>
            <div>
              <label className="label">TTL (seconds)</label>
              <input
                type="number"
                className="input"
                min="60"
                max="86400"
                value={newEndpoint.ttl}
                onChange={(e) => setNewEndpoint({ ...newEndpoint, ttl: parseInt(e.target.value) })}
              />
              <p className="mt-1 text-xs text-gray-500">
                How long the endpoint stays active (60-86400 seconds)
              </p>
            </div>
            <div className="flex gap-3">
              <button
                onClick={() => setShowCreateForm(false)}
                className="btn-secondary"
              >
                Cancel
              </button>
              <button
                onClick={() => createMutation.mutate()}
                disabled={createMutation.isPending}
                className="btn-primary"
              >
                {createMutation.isPending ? 'Creating...' : 'Create Endpoint'}
              </button>
            </div>
          </div>
        </div>
      ) : (
        <button
          onClick={() => setShowCreateForm(true)}
          className="btn-primary flex items-center gap-2"
        >
          <PlusIcon className="h-5 w-5" />
          Create Test Endpoint
        </button>
      )}

      {/* Endpoints List */}
      {!endpoints || endpoints.length === 0 ? (
        <EmptyState
          title="No test endpoints"
          description="Create a temporary endpoint to receive and inspect webhooks."
          icon={<BeakerIcon className="h-12 w-12" />}
        />
      ) : (
        <div className="space-y-4">
          {endpoints.map((endpoint) => (
            <div key={endpoint.id} className="card p-4">
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <h4 className="font-medium text-gray-900">
                    {endpoint.name || 'Unnamed Endpoint'}
                  </h4>
                  {endpoint.description && (
                    <p className="text-sm text-gray-500 mt-1">{endpoint.description}</p>
                  )}
                  <div className="mt-2">
                    <label className="text-xs text-gray-500">URL</label>
                    <code className="block text-sm font-mono bg-gray-50 p-2 rounded mt-1 break-all">
                      {endpoint.url}
                    </code>
                  </div>
                  <div className="mt-2 flex gap-4 text-xs text-gray-500">
                    <span>Created: {format(new Date(endpoint.created_at), 'PPp')}</span>
                    <span>Expires: {format(new Date(endpoint.expires_at), 'PPp')}</span>
                  </div>
                </div>
                <button
                  onClick={() => deleteMutation.mutate(endpoint.id)}
                  className="p-1 text-gray-400 hover:text-red-600"
                >
                  <TrashIcon className="h-5 w-5" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default TestingTools;
