import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { PlusIcon, LinkIcon } from '@heroicons/react/24/outline';
import { endpointsService } from '@/services';
import { WebhookEndpoint } from '@/types';
import { PageLoader, EmptyState, StatusBadge, Pagination, Modal } from '@/components/common';
import { useUIStore } from '@/store';
import { formatDistanceToNow } from 'date-fns';
import { EndpointForm } from './EndpointForm';

export function EndpointsList() {
  const [page, setPage] = useState(1);
  const queryClient = useQueryClient();
  const { openModal, closeModal, modals, setSelectedEndpoint } = useUIStore();

  const { data, isLoading, error } = useQuery({
    queryKey: ['endpoints', page],
    queryFn: () => endpointsService.list(page, 20),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => endpointsService.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['endpoints'] });
    },
  });

  if (isLoading) return <PageLoader />;
  if (error) return <div className="text-red-600">Error loading endpoints</div>;

  const endpoints = data?.data || [];
  const totalPages = data?.total_pages || 1;

  return (
    <div>
      <div className="sm:flex sm:items-center">
        <div className="sm:flex-auto">
          <h1 className="text-2xl font-semibold leading-6 text-gray-900">Endpoints</h1>
          <p className="mt-2 text-sm text-gray-700">
            Manage your webhook endpoints and their configurations.
          </p>
        </div>
        <div className="mt-4 sm:ml-16 sm:mt-0 sm:flex-none">
          <button
            type="button"
            onClick={() => openModal('createEndpoint')}
            className="btn-primary flex items-center gap-x-2"
          >
            <PlusIcon className="h-5 w-5" />
            Add Endpoint
          </button>
        </div>
      </div>

      {endpoints.length === 0 ? (
        <EmptyState
          title="No endpoints"
          description="Get started by creating a new webhook endpoint."
          icon={<LinkIcon className="h-12 w-12" />}
          action={
            <button
              onClick={() => openModal('createEndpoint')}
              className="btn-primary"
            >
              Create Endpoint
            </button>
          }
        />
      ) : (
        <div className="mt-8 card overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  URL
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Retry Config
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Created
                </th>
                <th className="relative px-6 py-3">
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {endpoints.map((endpoint: WebhookEndpoint) => (
                <tr key={endpoint.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4">
                    <div className="flex items-center">
                      <div>
                        <div className="text-sm font-medium text-gray-900 truncate max-w-md">
                          {endpoint.url}
                        </div>
                        <div className="text-sm text-gray-500">
                          ID: {endpoint.id.slice(0, 8)}...
                        </div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <StatusBadge status={endpoint.is_active ? 'active' : 'inactive'} />
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {endpoint.retry_config.max_attempts} attempts
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {formatDistanceToNow(new Date(endpoint.created_at), { addSuffix: true })}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <button
                      onClick={() => {
                        setSelectedEndpoint(endpoint);
                        openModal('editEndpoint');
                      }}
                      className="text-primary-600 hover:text-primary-900 mr-4"
                    >
                      Edit
                    </button>
                    <button
                      onClick={() => {
                        if (confirm('Are you sure you want to delete this endpoint?')) {
                          deleteMutation.mutate(endpoint.id);
                        }
                      }}
                      className="text-red-600 hover:text-red-900"
                    >
                      Delete
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
        isOpen={modals.createEndpoint}
        onClose={() => closeModal('createEndpoint')}
        title="Create Endpoint"
        size="lg"
      >
        <EndpointForm
          onSuccess={() => {
            closeModal('createEndpoint');
            queryClient.invalidateQueries({ queryKey: ['endpoints'] });
          }}
          onCancel={() => closeModal('createEndpoint')}
        />
      </Modal>

      <Modal
        isOpen={modals.editEndpoint}
        onClose={() => closeModal('editEndpoint')}
        title="Edit Endpoint"
        size="lg"
      >
        <EndpointForm
          endpoint={useUIStore.getState().selectedEndpoint || undefined}
          onSuccess={() => {
            closeModal('editEndpoint');
            queryClient.invalidateQueries({ queryKey: ['endpoints'] });
          }}
          onCancel={() => closeModal('editEndpoint')}
        />
      </Modal>
    </div>
  );
}

export default EndpointsList;
