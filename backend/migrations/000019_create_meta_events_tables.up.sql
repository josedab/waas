-- Meta-events subscription tables
CREATE TABLE IF NOT EXISTS meta_event_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    secret VARCHAR(255) NOT NULL,
    event_types TEXT[] NOT NULL,
    filters JSONB,
    is_active BOOLEAN DEFAULT true,
    headers JSONB DEFAULT '{}',
    retry_policy JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_meta_event_subscriptions_tenant_id ON meta_event_subscriptions(tenant_id);
CREATE INDEX idx_meta_event_subscriptions_is_active ON meta_event_subscriptions(is_active);
CREATE INDEX idx_meta_event_subscriptions_event_types ON meta_event_subscriptions USING GIN(event_types);

-- Meta-events table
CREATE TABLE IF NOT EXISTS meta_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    type VARCHAR(100) NOT NULL,
    source VARCHAR(100) NOT NULL,
    source_id VARCHAR(255) NOT NULL,
    data JSONB NOT NULL DEFAULT '{}',
    metadata JSONB,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL,
    delivered_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_meta_events_tenant_id ON meta_events(tenant_id);
CREATE INDEX idx_meta_events_type ON meta_events(type);
CREATE INDEX idx_meta_events_source ON meta_events(source);
CREATE INDEX idx_meta_events_occurred_at ON meta_events(occurred_at);
CREATE INDEX idx_meta_events_created_at ON meta_events(created_at);

-- Meta-event deliveries table
CREATE TABLE IF NOT EXISTS meta_event_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES meta_event_subscriptions(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES meta_events(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    attempt INTEGER DEFAULT 1,
    status_code INTEGER,
    response_body TEXT,
    error TEXT,
    latency_ms INTEGER DEFAULT 0,
    next_retry TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_meta_event_deliveries_subscription_id ON meta_event_deliveries(subscription_id);
CREATE INDEX idx_meta_event_deliveries_event_id ON meta_event_deliveries(event_id);
CREATE INDEX idx_meta_event_deliveries_tenant_id ON meta_event_deliveries(tenant_id);
CREATE INDEX idx_meta_event_deliveries_status ON meta_event_deliveries(status);
CREATE INDEX idx_meta_event_deliveries_created_at ON meta_event_deliveries(created_at);
