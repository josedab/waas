import React from 'react';
import type { Endpoint } from '../types';

interface EndpointListProps {
  endpoints: Endpoint[];
  onDelete: (id: string) => void;
  onToggle: (id: string, active: boolean) => void;
}

export function EndpointList({ endpoints, onDelete, onToggle }: EndpointListProps) {
  if (endpoints.length === 0) {
    return <div className="waas-empty">No endpoints configured. Create one to get started.</div>;
  }

  return (
    <table className="waas-table">
      <thead>
        <tr>
          <th>URL</th>
          <th>Status</th>
          <th>Created</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        {endpoints.map((ep) => (
          <tr key={ep.id}>
            <td>
              <span className="waas-mono" title={ep.url}>
                {ep.url.length > 50 ? ep.url.substring(0, 47) + '...' : ep.url}
              </span>
            </td>
            <td>
              <span className={`waas-status waas-status--${ep.is_active ? 'active' : 'inactive'}`}>
                {ep.is_active ? 'Active' : 'Inactive'}
              </span>
            </td>
            <td>{new Date(ep.created_at).toLocaleDateString()}</td>
            <td>
              <button
                className="waas-btn waas-btn--sm"
                onClick={() => onToggle(ep.id, !ep.is_active)}
              >
                {ep.is_active ? 'Disable' : 'Enable'}
              </button>
              {' '}
              <button
                className="waas-btn waas-btn--sm waas-btn--danger"
                onClick={() => {
                  if (confirm('Delete this endpoint?')) onDelete(ep.id);
                }}
              >
                Delete
              </button>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
