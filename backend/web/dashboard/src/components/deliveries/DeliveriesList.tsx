import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { PaperAirplaneIcon } from '@heroicons/react/24/outline';
import { deliveriesService } from '@/services';
import { DeliveryAttempt } from '@/types';
import { PageLoader, EmptyState, StatusBadge, Pagination, Modal } from '@/components/common';
import { useUIStore } from '@/store';
import { formatDistanceToNow, format } from 'date-fns';
import { DeliveryDetails } from './DeliveryDetails';

export function DeliveriesList() {
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<string>('');
  const { openModal, closeModal, modals, setSelectedDelivery, selectedDelivery } = useUIStore();

  const { data, isLoading, error } = useQuery({
    queryKey: ['deliveries', page, statusFilter],
    queryFn: () => deliveriesService.list(page, 20, { status: statusFilter || undefined }),
  });

  if (isLoading) return <PageLoader />;
  if (error) return <div className="text-red-600">Error loading deliveries</div>;

  const deliveries = data?.data || [];
  const totalPages = data?.total_pages || 1;

  return (
    <div>
      <div className="sm:flex sm:items-center">
        <div className="sm:flex-auto">
          <h1 className="text-2xl font-semibold leading-6 text-gray-900">Deliveries</h1>
          <p className="mt-2 text-sm text-gray-700">
            View and monitor webhook delivery attempts.
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className="mt-4 flex gap-4">
        <select
          className="input w-auto"
          value={statusFilter}
          onChange={(e) => {
            setStatusFilter(e.target.value);
            setPage(1);
          }}
        >
          <option value="">All Statuses</option>
          <option value="success">Success</option>
          <option value="failed">Failed</option>
          <option value="pending">Pending</option>
          <option value="retrying">Retrying</option>
        </select>
      </div>

      {deliveries.length === 0 ? (
        <EmptyState
          title="No deliveries"
          description="Deliveries will appear here once you start sending webhooks."
          icon={<PaperAirplaneIcon className="h-12 w-12" />}
        />
      ) : (
        <div className="mt-6 card overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Delivery ID
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  HTTP Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Attempt
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Scheduled
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Delivered
                </th>
                <th className="relative px-6 py-3">
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {deliveries.map((delivery: DeliveryAttempt) => (
                <tr key={delivery.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-mono text-gray-900">
                      {delivery.id.slice(0, 8)}...
                    </div>
                    <div className="text-xs text-gray-500">
                      Endpoint: {delivery.endpoint_id.slice(0, 8)}...
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <StatusBadge status={delivery.status} />
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {delivery.http_status || '-'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    #{delivery.attempt_number}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatDistanceToNow(new Date(delivery.scheduled_at), { addSuffix: true })}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {delivery.delivered_at
                      ? format(new Date(delivery.delivered_at), 'MMM d, HH:mm:ss')
                      : '-'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <button
                      onClick={() => {
                        setSelectedDelivery(delivery);
                        openModal('deliveryDetails');
                      }}
                      className="text-primary-600 hover:text-primary-900"
                    >
                      Details
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {totalPages > 1 && (
            <Pagination
              currentPage={page}
              totalPages={totalPages}
              totalItems={data?.total || 0}
              perPage={20}
              onPageChange={setPage}
            />
          )}
        </div>
      )}

      <Modal
        isOpen={modals.deliveryDetails}
        onClose={() => closeModal('deliveryDetails')}
        title="Delivery Details"
        size="xl"
      >
        {selectedDelivery && <DeliveryDetails delivery={selectedDelivery} />}
      </Modal>
    </div>
  );
}

export default DeliveriesList;
