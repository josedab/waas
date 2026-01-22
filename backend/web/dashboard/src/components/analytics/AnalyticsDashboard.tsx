import { useQuery } from '@tanstack/react-query';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend,
  ArcElement,
  Filler,
} from 'chart.js';
import { Line, Bar, Doughnut } from 'react-chartjs-2';
import { analyticsService } from '@/services';
import { PageLoader } from '@/components/common';
import { format, subDays } from 'date-fns';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  Filler
);

export function AnalyticsDashboard() {
  const startDate = format(subDays(new Date(), 30), 'yyyy-MM-dd');
  const endDate = format(new Date(), 'yyyy-MM-dd');

  const { data: summary, isLoading: summaryLoading } = useQuery({
    queryKey: ['analytics-summary'],
    queryFn: () => analyticsService.getSummary({ start_date: startDate, end_date: endDate }),
  });

  const { data: deliveryTimeSeries, isLoading: timeSeriesLoading } = useQuery({
    queryKey: ['analytics-timeseries'],
    queryFn: () => analyticsService.getDeliveryTimeSeries({ 
      start_date: startDate, 
      end_date: endDate,
      interval: 'day'
    }),
  });

  const { data: failureReasons } = useQuery({
    queryKey: ['analytics-failures'],
    queryFn: () => analyticsService.getFailureReasons(),
  });

  const { data: quota } = useQuery({
    queryKey: ['analytics-quota'],
    queryFn: () => analyticsService.getQuotaUsage(),
  });

  if (summaryLoading || timeSeriesLoading) return <PageLoader />;

  const deliveryChartData = {
    labels: deliveryTimeSeries?.map(d => format(new Date(d.timestamp), 'MMM d')) || [],
    datasets: [
      {
        label: 'Deliveries',
        data: deliveryTimeSeries?.map(d => d.value) || [],
        borderColor: 'rgb(14, 165, 233)',
        backgroundColor: 'rgba(14, 165, 233, 0.1)',
        fill: true,
        tension: 0.4,
      },
    ],
  };

  const statusChartData = {
    labels: ['Successful', 'Failed'],
    datasets: [
      {
        data: [
          summary?.successful_deliveries || 0,
          summary?.failed_deliveries || 0,
        ],
        backgroundColor: ['rgb(34, 197, 94)', 'rgb(239, 68, 68)'],
        borderWidth: 0,
      },
    ],
  };

  const failureChartData = {
    labels: failureReasons?.map(r => r.reason) || [],
    datasets: [
      {
        label: 'Failures',
        data: failureReasons?.map(r => r.count) || [],
        backgroundColor: 'rgba(239, 68, 68, 0.8)',
      },
    ],
  };

  return (
    <div>
      <div className="sm:flex sm:items-center">
        <div className="sm:flex-auto">
          <h1 className="text-2xl font-semibold leading-6 text-gray-900">Analytics</h1>
          <p className="mt-2 text-sm text-gray-700">
            Monitor your webhook delivery performance and usage.
          </p>
        </div>
      </div>

      {/* Summary Stats */}
      <div className="mt-6 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
        <div className="card p-6">
          <dt className="text-sm font-medium text-gray-500 truncate">Total Deliveries</dt>
          <dd className="mt-2 text-3xl font-semibold text-gray-900">
            {summary?.total_deliveries?.toLocaleString() || 0}
          </dd>
        </div>
        <div className="card p-6">
          <dt className="text-sm font-medium text-gray-500 truncate">Success Rate</dt>
          <dd className="mt-2 text-3xl font-semibold text-green-600">
            {summary?.success_rate?.toFixed(1) || 0}%
          </dd>
        </div>
        <div className="card p-6">
          <dt className="text-sm font-medium text-gray-500 truncate">Avg Latency</dt>
          <dd className="mt-2 text-3xl font-semibold text-gray-900">
            {summary?.avg_latency_ms?.toFixed(0) || 0}ms
          </dd>
        </div>
        <div className="card p-6">
          <dt className="text-sm font-medium text-gray-500 truncate">P99 Latency</dt>
          <dd className="mt-2 text-3xl font-semibold text-gray-900">
            {summary?.p99_latency_ms?.toFixed(0) || 0}ms
          </dd>
        </div>
      </div>

      {/* Quota Usage */}
      {quota && (
        <div className="mt-6 card p-6">
          <h3 className="text-lg font-medium text-gray-900">Monthly Quota Usage</h3>
          <div className="mt-4">
            <div className="flex justify-between text-sm mb-2">
              <span className="text-gray-500">
                {quota.request_count.toLocaleString()} of monthly quota used
              </span>
              <span className="text-gray-700 font-medium">
                {((quota.request_count / (quota.request_count + 1000)) * 100).toFixed(1)}%
              </span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-2.5">
              <div
                className="bg-primary-600 h-2.5 rounded-full"
                style={{ width: `${Math.min((quota.request_count / (quota.request_count + 1000)) * 100, 100)}%` }}
              />
            </div>
          </div>
          <div className="mt-4 grid grid-cols-3 gap-4 text-sm">
            <div>
              <span className="text-gray-500">Successful</span>
              <p className="text-green-600 font-medium">{quota.success_count.toLocaleString()}</p>
            </div>
            <div>
              <span className="text-gray-500">Failed</span>
              <p className="text-red-600 font-medium">{quota.failure_count.toLocaleString()}</p>
            </div>
            <div>
              <span className="text-gray-500">Overage</span>
              <p className="text-orange-600 font-medium">{quota.overage_count.toLocaleString()}</p>
            </div>
          </div>
        </div>
      )}

      {/* Charts */}
      <div className="mt-6 grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Delivery Trend */}
        <div className="card p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Delivery Trend (30 days)</h3>
          <div className="h-64">
            <Line
              data={deliveryChartData}
              options={{
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                  legend: { display: false },
                },
                scales: {
                  y: { beginAtZero: true },
                },
              }}
            />
          </div>
        </div>

        {/* Success/Failure Ratio */}
        <div className="card p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Delivery Status</h3>
          <div className="h-64 flex items-center justify-center">
            <div className="w-48">
              <Doughnut
                data={statusChartData}
                options={{
                  responsive: true,
                  maintainAspectRatio: false,
                  plugins: {
                    legend: { position: 'bottom' },
                  },
                }}
              />
            </div>
          </div>
        </div>

        {/* Failure Reasons */}
        {failureReasons && failureReasons.length > 0 && (
          <div className="card p-6 lg:col-span-2">
            <h3 className="text-lg font-medium text-gray-900 mb-4">Top Failure Reasons</h3>
            <div className="h-64">
              <Bar
                data={failureChartData}
                options={{
                  responsive: true,
                  maintainAspectRatio: false,
                  plugins: {
                    legend: { display: false },
                  },
                  scales: {
                    y: { beginAtZero: true },
                  },
                }}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default AnalyticsDashboard;
