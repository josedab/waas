import { DeliveryStatus } from '@/types';

interface StatusBadgeProps {
  status: DeliveryStatus | string;
  size?: 'sm' | 'md';
}

const statusConfig: Record<string, { className: string; label: string }> = {
  success: { className: 'status-success', label: 'Success' },
  delivered: { className: 'status-success', label: 'Delivered' },
  failed: { className: 'status-failed', label: 'Failed' },
  pending: { className: 'status-pending', label: 'Pending' },
  processing: { className: 'status-pending', label: 'Processing' },
  retrying: { className: 'status-retrying', label: 'Retrying' },
  active: { className: 'status-success', label: 'Active' },
  inactive: { className: 'status-failed', label: 'Inactive' },
};

export function StatusBadge({ status, size = 'md' }: StatusBadgeProps) {
  const config = statusConfig[status] || { className: 'status-pending', label: status };
  const sizeClass = size === 'sm' ? 'text-xs px-2 py-0.5' : 'text-sm px-2.5 py-0.5';

  return (
    <span className={`${config.className} ${sizeClass}`}>
      {config.label}
    </span>
  );
}

export default StatusBadge;
