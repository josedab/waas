import React, { useState, useEffect, useCallback } from 'react';
import type { WaaSPortalProps, PortalTheme, Endpoint, Delivery } from '../types';
import { createApiClient } from '../api';
import { EndpointList } from './EndpointList';
import { DeliveryLogs } from './DeliveryLogs';
import { EndpointForm } from './EndpointForm';
import { PortalAnalytics } from './PortalAnalytics';

const defaultTheme: Required<PortalTheme> = {
  primaryColor: '#6366f1',
  backgroundColor: '#ffffff',
  surfaceColor: '#f9fafb',
  textColor: '#111827',
  mutedColor: '#6b7280',
  successColor: '#10b981',
  errorColor: '#ef4444',
  warningColor: '#f59e0b',
  borderRadius: '8px',
  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
};

/**
 * WaaSPortal - Drop-in embeddable webhook management portal.
 *
 * Usage:
 * ```tsx
 * <WaaSPortal
 *   config={{ apiUrl: 'https://your-waas.com', token: 'embed_...' }}
 *   theme={{ primaryColor: '#0066ff' }}
 *   onEndpointCreated={(ep) => console.log('Created:', ep)}
 * />
 * ```
 */
export function WaaSPortal({ config, theme, className, onEndpointCreated, onEndpointDeleted, onError }: WaaSPortalProps) {
  const mergedTheme = { ...defaultTheme, ...theme };
  const api = createApiClient(config);
  const features = { endpoints: true, deliveries: true, testSender: false, analytics: true, ...config.features };

  const [tab, setTab] = useState<'endpoints' | 'deliveries' | 'analytics'>('endpoints');
  const [endpoints, setEndpoints] = useState<Endpoint[]>([]);
  const [deliveries, setDeliveries] = useState<Delivery[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const cssVars = {
    '--waas-primary': mergedTheme.primaryColor,
    '--waas-bg': mergedTheme.backgroundColor,
    '--waas-surface': mergedTheme.surfaceColor,
    '--waas-text': mergedTheme.textColor,
    '--waas-muted': mergedTheme.mutedColor,
    '--waas-success': mergedTheme.successColor,
    '--waas-error': mergedTheme.errorColor,
    '--waas-warning': mergedTheme.warningColor,
    '--waas-radius': mergedTheme.borderRadius,
    '--waas-font': mergedTheme.fontFamily,
  } as React.CSSProperties;

  const loadEndpoints = useCallback(async () => {
    try {
      const eps = await api.listEndpoints();
      setEndpoints(eps);
      setError(null);
    } catch (err) {
      const e = err instanceof Error ? err : new Error(String(err));
      setError(e.message);
      onError?.(e);
    }
  }, [config.apiUrl, config.token]);

  const loadDeliveries = useCallback(async () => {
    try {
      const dels = await api.listDeliveries(undefined, 50);
      setDeliveries(dels);
    } catch (err) {
      const e = err instanceof Error ? err : new Error(String(err));
      onError?.(e);
    }
  }, [config.apiUrl, config.token]);

  useEffect(() => {
    setLoading(true);
    Promise.all([loadEndpoints(), loadDeliveries()]).finally(() => setLoading(false));
  }, [loadEndpoints, loadDeliveries]);

  // Poll for delivery updates
  useEffect(() => {
    if (!features.deliveries) return;
    const interval = setInterval(loadDeliveries, config.pollInterval || 10000);
    return () => clearInterval(interval);
  }, [loadDeliveries, config.pollInterval]);

  const handleCreateEndpoint = async (url: string, headers?: Record<string, string>) => {
    try {
      const ep = await api.createEndpoint(url, headers);
      setEndpoints((prev) => [...prev, ep]);
      setShowForm(false);
      onEndpointCreated?.(ep);
    } catch (err) {
      const e = err instanceof Error ? err : new Error(String(err));
      setError(e.message);
      onError?.(e);
    }
  };

  const handleDeleteEndpoint = async (id: string) => {
    try {
      await api.deleteEndpoint(id);
      setEndpoints((prev) => prev.filter((ep) => ep.id !== id));
      onEndpointDeleted?.(id);
    } catch (err) {
      const e = err instanceof Error ? err : new Error(String(err));
      setError(e.message);
      onError?.(e);
    }
  };

  const handleToggleEndpoint = async (id: string, active: boolean) => {
    try {
      const updated = await api.updateEndpoint(id, { is_active: active });
      setEndpoints((prev) => prev.map((ep) => (ep.id === id ? updated : ep)));
    } catch (err) {
      const e = err instanceof Error ? err : new Error(String(err));
      onError?.(e);
    }
  };

  return (
    <div className={`waas-portal ${className || ''}`} style={cssVars}>
      <style>{portalStyles}</style>

      {error && (
        <div className="waas-error-banner">
          <span>{error}</span>
          <button onClick={() => setError(null)}>×</button>
        </div>
      )}

      <div className="waas-tabs">
        {features.endpoints && (
          <button
            className={`waas-tab ${tab === 'endpoints' ? 'waas-tab--active' : ''}`}
            onClick={() => setTab('endpoints')}
          >
            Endpoints ({endpoints.length})
          </button>
        )}
        {features.deliveries && (
          <button
            className={`waas-tab ${tab === 'deliveries' ? 'waas-tab--active' : ''}`}
            onClick={() => setTab('deliveries')}
          >
            Delivery Logs
          </button>
        )}
        {features.analytics && (
          <button
            className={`waas-tab ${tab === 'analytics' ? 'waas-tab--active' : ''}`}
            onClick={() => setTab('analytics')}
          >
            Analytics
          </button>
        )}
      </div>

      <div className="waas-content">
        {loading ? (
          <div className="waas-loading">Loading...</div>
        ) : tab === 'endpoints' ? (
          <>
            <div className="waas-toolbar">
              <button className="waas-btn waas-btn--primary" onClick={() => setShowForm(!showForm)}>
                {showForm ? 'Cancel' : '+ New Endpoint'}
              </button>
            </div>
            {showForm && <EndpointForm onSubmit={handleCreateEndpoint} maxEndpoints={config.maxEndpoints} currentCount={endpoints.length} />}
            <EndpointList
              endpoints={endpoints}
              onDelete={handleDeleteEndpoint}
              onToggle={handleToggleEndpoint}
            />
          </>
        ) : tab === 'deliveries' ? (
          <DeliveryLogs deliveries={deliveries} onRefresh={loadDeliveries} />
        ) : tab === 'analytics' ? (
          <PortalAnalytics config={config} />
        ) : null}
      </div>
    </div>
  );
}

const portalStyles = `
.waas-portal {
  font-family: var(--waas-font);
  color: var(--waas-text);
  background: var(--waas-bg);
  border-radius: var(--waas-radius);
  border: 1px solid #e5e7eb;
  overflow: hidden;
}
.waas-tabs {
  display: flex;
  border-bottom: 1px solid #e5e7eb;
  background: var(--waas-surface);
}
.waas-tab {
  padding: 12px 20px;
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 14px;
  color: var(--waas-muted);
  border-bottom: 2px solid transparent;
  transition: all 0.15s;
}
.waas-tab:hover { color: var(--waas-text); }
.waas-tab--active {
  color: var(--waas-primary);
  border-bottom-color: var(--waas-primary);
  font-weight: 600;
}
.waas-content { padding: 16px; }
.waas-toolbar { margin-bottom: 16px; }
.waas-btn {
  padding: 8px 16px;
  border-radius: var(--waas-radius);
  border: 1px solid #d1d5db;
  background: white;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.15s;
}
.waas-btn:hover { background: #f3f4f6; }
.waas-btn--primary {
  background: var(--waas-primary);
  color: white;
  border-color: var(--waas-primary);
}
.waas-btn--primary:hover { opacity: 0.9; }
.waas-btn--danger { color: var(--waas-error); }
.waas-btn--sm { padding: 4px 10px; font-size: 12px; }
.waas-loading { text-align: center; padding: 40px; color: var(--waas-muted); }
.waas-error-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  background: #fef2f2;
  color: var(--waas-error);
  font-size: 13px;
}
.waas-error-banner button { background: none; border: none; cursor: pointer; font-size: 16px; color: inherit; }
.waas-empty { text-align: center; padding: 32px; color: var(--waas-muted); font-size: 14px; }
.waas-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.waas-table th { text-align: left; padding: 8px 12px; border-bottom: 2px solid #e5e7eb; color: var(--waas-muted); font-weight: 600; font-size: 11px; text-transform: uppercase; letter-spacing: 0.05em; }
.waas-table td { padding: 10px 12px; border-bottom: 1px solid #f3f4f6; }
.waas-table tr:hover td { background: var(--waas-surface); }
.waas-status { display: inline-block; padding: 2px 8px; border-radius: 12px; font-size: 11px; font-weight: 600; }
.waas-status--delivered, .waas-status--active { background: #d1fae5; color: #065f46; }
.waas-status--failed, .waas-status--inactive { background: #fee2e2; color: #991b1b; }
.waas-status--retrying, .waas-status--pending { background: #fef3c7; color: #92400e; }
.waas-form { padding: 16px; background: var(--waas-surface); border-radius: var(--waas-radius); margin-bottom: 16px; }
.waas-form label { display: block; font-size: 13px; font-weight: 600; margin-bottom: 4px; }
.waas-form input { width: 100%; padding: 8px 12px; border: 1px solid #d1d5db; border-radius: var(--waas-radius); font-size: 14px; box-sizing: border-box; }
.waas-form input:focus { outline: none; border-color: var(--waas-primary); box-shadow: 0 0 0 2px rgba(99,102,241,0.2); }
.waas-form-actions { margin-top: 12px; display: flex; gap: 8px; }
.waas-mono { font-family: "SF Mono", Monaco, Consolas, monospace; font-size: 12px; }
`;
