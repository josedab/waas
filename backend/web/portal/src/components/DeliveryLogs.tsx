import React from 'react';
import type { Delivery } from '../types';

interface DeliveryLogsProps {
  deliveries: Delivery[];
  onRefresh: () => void;
}

export function DeliveryLogs({ deliveries, onRefresh }: DeliveryLogsProps) {
  return (
    <>
      <div className="waas-toolbar" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span style={{ fontSize: 13, color: 'var(--waas-muted)' }}>
          Showing {deliveries.length} recent deliveries
        </span>
        <button className="waas-btn waas-btn--sm" onClick={onRefresh}>
          ↻ Refresh
        </button>
      </div>

      {deliveries.length === 0 ? (
        <div className="waas-empty">No deliveries yet. Send a webhook to see logs here.</div>
      ) : (
        <table className="waas-table">
          <thead>
            <tr>
              <th>Delivery ID</th>
              <th>Endpoint</th>
              <th>Status</th>
              <th>Attempts</th>
              <th>HTTP</th>
              <th>Time</th>
            </tr>
          </thead>
          <tbody>
            {deliveries.map((d) => (
              <tr key={d.id}>
                <td className="waas-mono">{d.id.substring(0, 12)}...</td>
                <td className="waas-mono">{d.endpoint_id.substring(0, 12)}...</td>
                <td>
                  <span className={`waas-status waas-status--${d.status}`}>
                    {d.status}
                  </span>
                </td>
                <td>{d.attempt_count}</td>
                <td>{d.last_http_status || '—'}</td>
                <td>{d.created_at ? new Date(d.created_at).toLocaleTimeString() : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </>
  );
}
