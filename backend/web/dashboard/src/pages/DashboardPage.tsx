import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import {
  ArrowTrendingUpIcon,
  ArrowTrendingDownIcon,
  LinkIcon,
  PaperAirplaneIcon,
  CheckCircleIcon,
} from '@heroicons/react/24/outline';
import { analyticsService, endpointsService, deliveriesService } from '@/services';
import { PageLoader, StatusBadge } from '@/components/common';
import { useAuthStore } from '@/store';
import { formatDistanceToNow } from 'date-fns';

export function DashboardPage() {
  const { tenant } = useAuthStore();

  const { data: summary, isLoading: summaryLoading } = useQuery({
    queryKey: ['dashboard-summary'],
    queryFn: () => analyticsService.getSummary(),
  });

  const { data: endpoints, isLoading: endpointsLoading } = useQuery({
    queryKey: ['dashboard-endpoints'],
    queryFn: () => endpointsService.list(1, 5),
  });

  const { data: recentDeliveries, isLoading: deliveriesLoading } = useQuery({
    queryKey: ['dashboard-deliveries'],
    queryFn: () => deliveriesService.list(1, 10),
  });

  if (summaryLoading || endpointsLoading || deliveriesLoading) return <PageLoader />;

  const stats = [
    {
      name: 'Total Deliveries',
      value: summary?.total_deliveries?.toLocaleString() || '0',
      icon: PaperAirplaneIcon,
      change: '+12%',
      changeType: 'positive',
    },
    {
      name: 'Success Rate',
      value: `${summary?.success_rate?.toFixed(1) || 0}%`,
      icon: CheckCircleIcon,
      change: summary?.success_rate && summary.success_rate >= 95 ? 'Healthy' : 'Needs attention',
      changeType: summary?.success_rate && summary.success_rate >= 95 ? 'positive' : 'negative',
    },
    {
      name: 'Active Endpoints',
      value: endpoints?.data?.filter(e => e.is_active).length.toString() || '0',
      icon: LinkIcon,
      change: `of ${endpoints?.total || 0}`,
      changeType: 'neutral',
    },
    {
      name: 'Avg Latency',
      value: `${summary?.avg_latency_ms?.toFixed(0) || 0}ms`,
      icon: ArrowTrendingUpIcon,
      change: summary?.avg_latency_ms && summary.avg_latency_ms < 500 ? 'Good' : 'High',
      changeType: summary?.avg_latency_ms && summary.avg_latency_ms < 500 ? 'positive' : 'negative',
    },
  ];

  return (
    <div>
      {/* Header */}
      <div className="md:flex md:items-center md:justify-between">
        <div className="min-w-0 flex-1">
          <h1 className="text-2xl font-bold leading-7 text-gray-900 sm:truncate sm:text-3xl">
            Welcome back, {tenant?.name || 'User'}
          </h1>
          <p className="mt-1 text-sm text-gray-500">
            Here's what's happening with your webhooks
          </p>
        </div>
        <div className="mt-4 flex md:ml-4 md:mt-0 gap-3">
          <Link to="/endpoints" className="btn-secondary">
            Manage Endpoints
          </Link>
          <Link to="/testing" className="btn-primary">
            Send Test Webhook
          </Link>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => (
          <div key={stat.name} className="card p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <stat.icon className="h-8 w-8 text-gray-400" aria-hidden="true" />
              </div>
              <div className="ml-5 w-0 flex-1">
                <dl>
                  <dt className="text-sm font-medium text-gray-500 truncate">{stat.name}</dt>
                  <dd className="flex items-baseline">
                    <p className="text-2xl font-semibold text-gray-900">{stat.value}</p>
                    <p
                      className={`ml-2 flex items-baseline text-sm font-semibold ${
                        stat.changeType === 'positive'
                          ? 'text-green-600'
                          : stat.changeType === 'negative'
                          ? 'text-red-600'
                          : 'text-gray-500'
                      }`}
                    >
                      {stat.changeType === 'positive' && (
                        <ArrowTrendingUpIcon className="h-4 w-4 flex-shrink-0 mr-0.5" />
                      )}
                      {stat.changeType === 'negative' && (
                        <ArrowTrendingDownIcon className="h-4 w-4 flex-shrink-0 mr-0.5" />
                      )}
                      {stat.change}
                    </p>
                  </dd>
                </dl>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Two Column Layout */}
      <div className="mt-8 grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Recent Endpoints */}
        <div className="card">
          <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
            <h3 className="text-lg font-medium text-gray-900">Recent Endpoints</h3>
            <Link to="/endpoints" className="text-sm text-primary-600 hover:text-primary-700">
              View all
            </Link>
          </div>
          <ul className="divide-y divide-gray-200">
            {endpoints?.data?.slice(0, 5).map((endpoint) => (
              <li key={endpoint.id} className="px-6 py-4">
                <div className="flex items-center justify-between">
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium text-gray-900 truncate">
                      {endpoint.url}
                    </p>
                    <p className="text-sm text-gray-500">
                      Created {formatDistanceToNow(new Date(endpoint.created_at), { addSuffix: true })}
                    </p>
                  </div>
                  <StatusBadge status={endpoint.is_active ? 'active' : 'inactive'} size="sm" />
                </div>
              </li>
            ))}
            {(!endpoints?.data || endpoints.data.length === 0) && (
              <li className="px-6 py-8 text-center text-gray-500 text-sm">
                No endpoints yet. Create one to get started.
              </li>
            )}
          </ul>
        </div>

        {/* Recent Deliveries */}
        <div className="card">
          <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
            <h3 className="text-lg font-medium text-gray-900">Recent Deliveries</h3>
            <Link to="/deliveries" className="text-sm text-primary-600 hover:text-primary-700">
              View all
            </Link>
          </div>
          <ul className="divide-y divide-gray-200">
            {recentDeliveries?.data?.slice(0, 5).map((delivery) => (
              <li key={delivery.id} className="px-6 py-4">
                <div className="flex items-center justify-between">
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-mono text-gray-900">
                      {delivery.id.slice(0, 8)}...
                    </p>
                    <p className="text-sm text-gray-500">
                      {formatDistanceToNow(new Date(delivery.scheduled_at), { addSuffix: true })}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    {delivery.http_status && (
                      <span className="text-xs text-gray-500">HTTP {delivery.http_status}</span>
                    )}
                    <StatusBadge status={delivery.status} size="sm" />
                  </div>
                </div>
              </li>
            ))}
            {(!recentDeliveries?.data || recentDeliveries.data.length === 0) && (
              <li className="px-6 py-8 text-center text-gray-500 text-sm">
                No deliveries yet. Send a webhook to see activity here.
              </li>
            )}
          </ul>
        </div>
      </div>
    </div>
  );
}

export default DashboardPage;
