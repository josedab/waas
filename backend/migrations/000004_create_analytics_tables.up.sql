-- Analytics tables for metrics and aggregated data

-- Delivery metrics table for storing individual delivery events
CREATE TABLE delivery_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID NOT NULL REFERENCES webhook_endpoints(id),
    delivery_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL, -- success, failed, retrying
    http_status INTEGER,
    latency_ms INTEGER NOT NULL,
    attempt_number INTEGER NOT NULL DEFAULT 1,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_delivery_metrics_tenant_time ON delivery_metrics(tenant_id, created_at);
CREATE INDEX idx_delivery_metrics_endpoint_time ON delivery_metrics(endpoint_id, created_at);
CREATE INDEX idx_delivery_metrics_status ON delivery_metrics(status, created_at);

-- Hourly aggregated metrics for faster dashboard queries
CREATE TABLE hourly_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID REFERENCES webhook_endpoints(id), -- NULL for tenant-wide metrics
    hour_timestamp TIMESTAMP NOT NULL,
    total_deliveries INTEGER NOT NULL DEFAULT 0,
    successful_deliveries INTEGER NOT NULL DEFAULT 0,
    failed_deliveries INTEGER NOT NULL DEFAULT 0,
    retrying_deliveries INTEGER NOT NULL DEFAULT 0,
    avg_latency_ms NUMERIC(10,2),
    p95_latency_ms NUMERIC(10,2),
    p99_latency_ms NUMERIC(10,2),
    total_retries INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Unique constraint to prevent duplicate hourly records
CREATE UNIQUE INDEX idx_hourly_metrics_unique ON hourly_metrics(tenant_id, COALESCE(endpoint_id, '00000000-0000-0000-0000-000000000000'::UUID), hour_timestamp);
CREATE INDEX idx_hourly_metrics_tenant_time ON hourly_metrics(tenant_id, hour_timestamp);

-- Daily aggregated metrics for historical analysis
CREATE TABLE daily_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID REFERENCES webhook_endpoints(id), -- NULL for tenant-wide metrics
    date_timestamp DATE NOT NULL,
    total_deliveries INTEGER NOT NULL DEFAULT 0,
    successful_deliveries INTEGER NOT NULL DEFAULT 0,
    failed_deliveries INTEGER NOT NULL DEFAULT 0,
    retrying_deliveries INTEGER NOT NULL DEFAULT 0,
    avg_latency_ms NUMERIC(10,2),
    p95_latency_ms NUMERIC(10,2),
    p99_latency_ms NUMERIC(10,2),
    total_retries INTEGER NOT NULL DEFAULT 0,
    unique_endpoints INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Unique constraint to prevent duplicate daily records
CREATE UNIQUE INDEX idx_daily_metrics_unique ON daily_metrics(tenant_id, COALESCE(endpoint_id, '00000000-0000-0000-0000-000000000000'::UUID), date_timestamp);
CREATE INDEX idx_daily_metrics_tenant_date ON daily_metrics(tenant_id, date_timestamp);

-- Real-time metrics for WebSocket updates
CREATE TABLE realtime_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    metric_type VARCHAR(50) NOT NULL, -- delivery_rate, error_rate, latency, queue_size
    metric_value NUMERIC(15,4) NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    metadata JSONB -- Additional context data
);

-- Index for efficient real-time queries (keep only recent data)
CREATE INDEX idx_realtime_metrics_tenant_type_time ON realtime_metrics(tenant_id, metric_type, timestamp);

-- Function to automatically clean old real-time metrics (keep only last 24 hours)
CREATE OR REPLACE FUNCTION cleanup_realtime_metrics()
RETURNS void AS $$
BEGIN
    DELETE FROM realtime_metrics 
    WHERE timestamp < NOW() - INTERVAL '24 hours';
END;
$$ LANGUAGE plpgsql;

-- Alert configurations table
CREATE TABLE alert_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    alert_type VARCHAR(50) NOT NULL, -- delivery_failure_rate, high_latency, queue_backlog
    threshold_value NUMERIC(10,4) NOT NULL,
    time_window_minutes INTEGER NOT NULL DEFAULT 5,
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    notification_channels JSONB, -- webhook URLs, email addresses, etc.
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alert_configs_tenant ON alert_configs(tenant_id, is_enabled);

-- Alert history table
CREATE TABLE alert_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_config_id UUID NOT NULL REFERENCES alert_configs(id),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    alert_type VARCHAR(50) NOT NULL,
    triggered_value NUMERIC(10,4) NOT NULL,
    threshold_value NUMERIC(10,4) NOT NULL,
    message TEXT NOT NULL,
    resolved_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alert_history_tenant_time ON alert_history(tenant_id, created_at);
CREATE INDEX idx_alert_history_config ON alert_history(alert_config_id, created_at);