import React, { useState, useEffect, useCallback } from 'react';
import type { PortalConfig } from '../types';
import { createApiClient } from '../api';

interface AnalyticsData {
  total_deliveries: number;
  successful_deliveries: number;
  failed_deliveries: number;
  success_rate: number;
  avg_latency_ms: number;
  p95_latency_ms: number;
  active_endpoints: number;
  total_endpoints: number;
  deliveries_today: number;
  deliveries_this_week: number;
}

interface PortalAnalyticsProps {
  config: PortalConfig;
}

export function PortalAnalytics({ config }: PortalAnalyticsProps) {
  const [analytics, setAnalytics] = useState<AnalyticsData | null>(null);
  const [loading, setLoading] = useState(true);

  const loadAnalytics = useCallback(async () => {
    try {
      const api = createApiClient(config);
      const endpoints = await api.listEndpoints();
      const deliveries = await api.listDeliveries(undefined, 100);

      const now = new Date();
      const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate());
      const weekStart = new Date(todayStart.getTime() - 7 * 24 * 60 * 60 * 1000);

      const successful = deliveries.filter(d => d.status === 'delivered').length;
      const failed = deliveries.filter(d => d.status === 'failed').length;
      const today = deliveries.filter(d => new Date(d.created_at) >= todayStart).length;
      const thisWeek = deliveries.filter(d => new Date(d.created_at) >= weekStart).length;

      setAnalytics({
        total_deliveries: deliveries.length,
        successful_deliveries: successful,
        failed_deliveries: failed,
        success_rate: deliveries.length > 0 ? (successful / deliveries.length) * 100 : 0,
        avg_latency_ms: 0,
        p95_latency_ms: 0,
        active_endpoints: endpoints.filter(e => e.is_active).length,
        total_endpoints: endpoints.length,
        deliveries_today: today,
        deliveries_this_week: thisWeek,
      });
    } catch {
      // Silently handle errors
    } finally {
      setLoading(false);
    }
  }, [config]);

  useEffect(() => {
    loadAnalytics();
    const interval = setInterval(loadAnalytics, 30000);
    return () => clearInterval(interval);
  }, [loadAnalytics]);

  if (loading) {
    return <div className="waas-loading">Loading analytics...</div>;
  }

  if (!analytics) {
    return <div className="waas-empty">Analytics not available</div>;
  }

  return (
    <div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: '12px', marginBottom: '20px' }}>
        <StatCard label="Total Deliveries" value={analytics.total_deliveries.toLocaleString()} />
        <StatCard label="Success Rate" value={`${analytics.success_rate.toFixed(1)}%`} color={analytics.success_rate >= 95 ? 'var(--waas-success)' : analytics.success_rate >= 80 ? 'var(--waas-warning)' : 'var(--waas-error)'} />
        <StatCard label="Active Endpoints" value={`${analytics.active_endpoints}/${analytics.total_endpoints}`} />
        <StatCard label="Today" value={analytics.deliveries_today.toLocaleString()} />
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
        <div style={{ background: 'var(--waas-surface)', borderRadius: 'var(--waas-radius)', padding: '16px' }}>
          <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px' }}>Delivery Status</h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            <StatusBar label="Delivered" count={analytics.successful_deliveries} total={analytics.total_deliveries} color="var(--waas-success)" />
            <StatusBar label="Failed" count={analytics.failed_deliveries} total={analytics.total_deliveries} color="var(--waas-error)" />
            <StatusBar label="Other" count={analytics.total_deliveries - analytics.successful_deliveries - analytics.failed_deliveries} total={analytics.total_deliveries} color="var(--waas-warning)" />
          </div>
        </div>

        <div style={{ background: 'var(--waas-surface)', borderRadius: 'var(--waas-radius)', padding: '16px' }}>
          <h3 style={{ fontSize: '14px', fontWeight: 600, marginBottom: '12px' }}>Quick Stats</h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', fontSize: '13px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--waas-muted)' }}>This Week</span>
              <span style={{ fontWeight: 600 }}>{analytics.deliveries_this_week.toLocaleString()}</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--waas-muted)' }}>Successful</span>
              <span style={{ fontWeight: 600, color: 'var(--waas-success)' }}>{analytics.successful_deliveries.toLocaleString()}</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--waas-muted)' }}>Failed</span>
              <span style={{ fontWeight: 600, color: 'var(--waas-error)' }}>{analytics.failed_deliveries.toLocaleString()}</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--waas-muted)' }}>Endpoints</span>
              <span style={{ fontWeight: 600 }}>{analytics.total_endpoints}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function StatCard({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div style={{ background: 'var(--waas-surface)', borderRadius: 'var(--waas-radius)', padding: '16px' }}>
      <div style={{ fontSize: '11px', color: 'var(--waas-muted)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: '4px' }}>{label}</div>
      <div style={{ fontSize: '24px', fontWeight: 700, color: color || 'var(--waas-text)' }}>{value}</div>
    </div>
  );
}

function StatusBar({ label, count, total, color }: { label: string; count: number; total: number; color: string }) {
  const pct = total > 0 ? (count / total) * 100 : 0;
  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '12px', marginBottom: '2px' }}>
        <span style={{ color: 'var(--waas-muted)' }}>{label}</span>
        <span style={{ fontWeight: 600 }}>{count} ({pct.toFixed(1)}%)</span>
      </div>
      <div style={{ height: '6px', background: '#e5e7eb', borderRadius: '3px', overflow: 'hidden' }}>
        <div style={{ width: `${pct}%`, height: '100%', background: color, borderRadius: '3px', transition: 'width 0.3s' }} />
      </div>
    </div>
  );
}
