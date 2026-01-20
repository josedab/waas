-- GraphQL Subscriptions Gateway tables (extends 000039 graphql gateway tables)

-- GraphQL subscription schemas
CREATE TABLE IF NOT EXISTS graphql_subscription_schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version INT NOT NULL DEFAULT 1,
    schema_sdl TEXT NOT NULL,
    types JSONB NOT NULL DEFAULT '[]',
    subscriptions JSONB NOT NULL DEFAULT '[]',
    is_default BOOLEAN NOT NULL DEFAULT false,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_graphql_schema_name UNIQUE(tenant_id, name)
);

CREATE INDEX idx_graphql_subscription_schemas_tenant ON graphql_subscription_schemas(tenant_id);
CREATE INDEX idx_graphql_subscription_schemas_active ON graphql_subscription_schemas(tenant_id, active);

-- GraphQL subscriptions (active subscriptions)
CREATE TABLE IF NOT EXISTS graphql_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    client_id UUID NOT NULL,
    subscription_id VARCHAR(255) NOT NULL,
    query TEXT NOT NULL,
    variables JSONB DEFAULT '{}',
    operation_name VARCHAR(255),
    filters JSONB DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, paused, completed
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_graphql_subscription UNIQUE(client_id, subscription_id)
);

CREATE INDEX idx_graphql_subscriptions_tenant ON graphql_subscriptions(tenant_id);
CREATE INDEX idx_graphql_subscriptions_client ON graphql_subscriptions(client_id);
CREATE INDEX idx_graphql_subscriptions_status ON graphql_subscriptions(tenant_id, status);

-- GraphQL clients (WebSocket connections)
CREATE TABLE IF NOT EXISTS graphql_clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    protocol VARCHAR(50) NOT NULL DEFAULT 'graphql-transport-ws',
    user_agent VARCHAR(500),
    ip_address VARCHAR(45),
    connected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_ping_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL DEFAULT 'connected', -- connected, disconnected
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_graphql_clients_tenant ON graphql_clients(tenant_id);
CREATE INDEX idx_graphql_clients_status ON graphql_clients(tenant_id, status);
CREATE INDEX idx_graphql_clients_connected ON graphql_clients(connected_at);

-- GraphQL events (published to subscriptions)
CREATE TABLE IF NOT EXISTS graphql_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    source VARCHAR(255),
    published_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    subscribers_notified INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_graphql_events_tenant ON graphql_events(tenant_id);
CREATE INDEX idx_graphql_events_type ON graphql_events(tenant_id, event_type);
CREATE INDEX idx_graphql_events_published ON graphql_events(published_at);

-- Partition by published_at for better performance
-- CREATE TABLE graphql_events_partitioned (LIKE graphql_events INCLUDING ALL)
-- PARTITION BY RANGE (published_at);

-- GraphQL subscription stats
CREATE TABLE IF NOT EXISTS graphql_subscription_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    date DATE NOT NULL,
    total_connections BIGINT NOT NULL DEFAULT 0,
    peak_connections INT NOT NULL DEFAULT 0,
    total_subscriptions BIGINT NOT NULL DEFAULT 0,
    total_events BIGINT NOT NULL DEFAULT 0,
    total_messages_sent BIGINT NOT NULL DEFAULT 0,
    avg_latency_ms INT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_graphql_stats UNIQUE(tenant_id, date)
);

CREATE INDEX idx_graphql_subscription_stats_tenant ON graphql_subscription_stats(tenant_id, date);

-- Triggers
CREATE TRIGGER trigger_graphql_subscription_schemas_updated
    BEFORE UPDATE ON graphql_subscription_schemas
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();

CREATE TRIGGER trigger_graphql_subscriptions_updated
    BEFORE UPDATE ON graphql_subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();
