-- Pattern analysis data for rate intelligence
CREATE TABLE IF NOT EXISTS endpoint_throughput_patterns (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    endpoint_id TEXT NOT NULL,
    hourly_rates JSONB DEFAULT '[]',
    hourly_latencies JSONB DEFAULT '[]',
    day_of_week_factors JSONB DEFAULT '[]',
    peak_rate DOUBLE PRECISION DEFAULT 0,
    baseline_rate DOUBLE PRECISION DEFAULT 0,
    avg_success_rate DOUBLE PRECISION DEFAULT 0,
    avg_latency_ms DOUBLE PRECISION DEFAULT 0,
    error_burst_score DOUBLE PRECISION DEFAULT 0,
    throughput_trend TEXT DEFAULT 'stable',
    sample_count INTEGER DEFAULT 0,
    last_analyzed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, endpoint_id)
);

CREATE INDEX IF NOT EXISTS idx_throughput_patterns_tenant ON endpoint_throughput_patterns(tenant_id);
CREATE INDEX IF NOT EXISTS idx_throughput_patterns_endpoint ON endpoint_throughput_patterns(tenant_id, endpoint_id);
