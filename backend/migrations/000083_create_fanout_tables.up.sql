CREATE TABLE IF NOT EXISTS fanout_topics (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    max_subscribers INTEGER NOT NULL DEFAULT 100,
    retention_days INTEGER NOT NULL DEFAULT 30,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_fanout_topics_tenant_id ON fanout_topics(tenant_id);

CREATE TABLE IF NOT EXISTS fanout_subscriptions (
    id UUID PRIMARY KEY,
    topic_id UUID NOT NULL REFERENCES fanout_topics(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    endpoint_id UUID NOT NULL,
    filter_expression TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fanout_subscriptions_topic_id ON fanout_subscriptions(topic_id);
CREATE INDEX idx_fanout_subscriptions_tenant_id ON fanout_subscriptions(tenant_id);

CREATE TABLE IF NOT EXISTS fanout_events (
    id UUID PRIMARY KEY,
    topic_id UUID NOT NULL REFERENCES fanout_topics(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}',
    fan_out_count INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'published',
    published_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fanout_events_topic_id ON fanout_events(topic_id);
CREATE INDEX idx_fanout_events_tenant_id ON fanout_events(tenant_id);
CREATE INDEX idx_fanout_events_published_at ON fanout_events(published_at);
