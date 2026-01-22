import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { DeliveryAttempt } from '@/types';
import { testingService, deliveriesService } from '@/services';
import { StatusBadge, LoadingSpinner } from '@/components/common';
import { useUIStore } from '@/store';
import { format } from 'date-fns';

interface DeliveryDetailsProps {
  delivery: DeliveryAttempt;
}

export function DeliveryDetails({ delivery }: DeliveryDetailsProps) {
  const queryClient = useQueryClient();
  const { addNotification } = useUIStore();

  const { data: inspection, isLoading } = useQuery({
    queryKey: ['delivery-inspection', delivery.id],
    queryFn: () => testingService.inspectDelivery(delivery.id),
    retry: false,
  });

  const retryMutation = useMutation({
    mutationFn: () => deliveriesService.retry(delivery.id),
    onSuccess: () => {
      addNotification('success', 'Retry scheduled successfully');
      queryClient.invalidateQueries({ queryKey: ['deliveries'] });
    },
    onError: (error: Error) => {
      addNotification('error', error.message);
    },
  });

  if (isLoading) {
    return <LoadingSpinner className="py-8" />;
  }

  return (
    <div className="space-y-6">
      {/* Status Overview */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <StatusBadge status={delivery.status} />
          <span className="text-sm text-gray-500">
            Attempt #{delivery.attempt_number}
          </span>
        </div>
        {delivery.status === 'failed' && (
          <button
            onClick={() => retryMutation.mutate()}
            disabled={retryMutation.isPending}
            className="btn-secondary text-sm"
          >
            {retryMutation.isPending ? 'Retrying...' : 'Retry Delivery'}
          </button>
        )}
      </div>

      {/* Error Message */}
      {delivery.error_message && (
        <div className="p-4 bg-red-50 rounded-lg">
          <h4 className="font-medium text-red-800">Error</h4>
          <p className="mt-1 text-sm text-red-700">{delivery.error_message}</p>
          {inspection?.error_details && (
            <div className="mt-2 text-sm text-red-600">
              <strong>Suggestion:</strong> {inspection.error_details.suggestion}
            </div>
          )}
        </div>
      )}

      {/* Request Details */}
      {inspection?.request && (
        <div>
          <h4 className="font-medium text-gray-900 mb-2">Request</h4>
          <div className="bg-gray-50 rounded-lg p-4 space-y-2">
            <div className="flex gap-2">
              <span className="font-medium text-gray-500 w-20">Method:</span>
              <span className="font-mono text-sm">{inspection.request.method}</span>
            </div>
            <div className="flex gap-2">
              <span className="font-medium text-gray-500 w-20">URL:</span>
              <span className="font-mono text-sm break-all">{inspection.request.url}</span>
            </div>
            <div>
              <span className="font-medium text-gray-500">Headers:</span>
              <pre className="mt-1 text-xs bg-white p-2 rounded border overflow-x-auto">
                {JSON.stringify(inspection.request.headers, null, 2)}
              </pre>
            </div>
            <div>
              <span className="font-medium text-gray-500">Body:</span>
              <pre className="mt-1 text-xs bg-white p-2 rounded border overflow-x-auto max-h-40">
                {inspection.request.body}
              </pre>
            </div>
          </div>
        </div>
      )}

      {/* Response Details */}
      {inspection?.response && (
        <div>
          <h4 className="font-medium text-gray-900 mb-2">Response</h4>
          <div className="bg-gray-50 rounded-lg p-4 space-y-2">
            <div className="flex gap-2">
              <span className="font-medium text-gray-500 w-24">Status Code:</span>
              <span className={`font-mono text-sm ${
                inspection.response.status_code >= 200 && inspection.response.status_code < 300
                  ? 'text-green-600'
                  : 'text-red-600'
              }`}>
                {inspection.response.status_code}
              </span>
            </div>
            <div className="flex gap-2">
              <span className="font-medium text-gray-500 w-24">Latency:</span>
              <span className="font-mono text-sm">{inspection.response.latency_ms}ms</span>
            </div>
            <div>
              <span className="font-medium text-gray-500">Body:</span>
              <pre className="mt-1 text-xs bg-white p-2 rounded border overflow-x-auto max-h-40">
                {inspection.response.body || '(empty)'}
              </pre>
            </div>
          </div>
        </div>
      )}

      {/* Timeline */}
      {inspection?.timeline && inspection.timeline.length > 0 && (
        <div>
          <h4 className="font-medium text-gray-900 mb-2">Timeline</h4>
          <div className="space-y-2">
            {inspection.timeline.map((event, index) => (
              <div key={index} className="flex gap-3 text-sm">
                <span className="text-gray-500 font-mono w-24">
                  {format(new Date(event.timestamp), 'HH:mm:ss.SSS')}
                </span>
                <span className="text-gray-900">{event.event}</span>
                {event.details && (
                  <span className="text-gray-500">- {event.details}</span>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Basic Info (fallback if inspection fails) */}
      {!inspection && (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <span className="text-sm font-medium text-gray-500">Delivery ID</span>
              <p className="font-mono text-sm">{delivery.id}</p>
            </div>
            <div>
              <span className="text-sm font-medium text-gray-500">Endpoint ID</span>
              <p className="font-mono text-sm">{delivery.endpoint_id}</p>
            </div>
            <div>
              <span className="text-sm font-medium text-gray-500">Payload Size</span>
              <p className="text-sm">{delivery.payload_size} bytes</p>
            </div>
            <div>
              <span className="text-sm font-medium text-gray-500">HTTP Status</span>
              <p className="text-sm">{delivery.http_status || '-'}</p>
            </div>
            <div>
              <span className="text-sm font-medium text-gray-500">Scheduled At</span>
              <p className="text-sm">
                {format(new Date(delivery.scheduled_at), 'PPpp')}
              </p>
            </div>
            <div>
              <span className="text-sm font-medium text-gray-500">Delivered At</span>
              <p className="text-sm">
                {delivery.delivered_at
                  ? format(new Date(delivery.delivered_at), 'PPpp')
                  : '-'}
              </p>
            </div>
          </div>

          {delivery.response_body && (
            <div>
              <span className="text-sm font-medium text-gray-500">Response Body</span>
              <pre className="mt-1 text-xs bg-gray-50 p-2 rounded border overflow-x-auto max-h-40">
                {delivery.response_body}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default DeliveryDetails;
