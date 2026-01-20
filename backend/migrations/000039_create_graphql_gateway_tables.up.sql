-- Feature 5: GraphQL Subscriptions Gateway
-- Enables real-time GraphQL subscriptions to be delivered as webhooks

-- GraphQL schema registry for storing and versioning schemas
CREATE TABLE IF NOT EXISTS graphql_schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    schema_sdl TEXT NOT NULL,
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    introspection_endpoint VARCHAR(500),
    federation_enabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, name, version)
);

-- GraphQL subscription definitions
CREATE TABLE IF NOT EXISTS graphql_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    schema_id UUID NOT NULL REFERENCES graphql_schemas(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    subscription_query TEXT NOT NULL,
    variables JSONB DEFAULT '{}',
    filter_expression TEXT,
    field_selection JSONB DEFAULT '[]',
    transform_js TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    delivery_config JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, name)
);

-- GraphQL subscription events tracking
CREATE TABLE IF NOT EXISTS graphql_subscription_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES graphql_subscriptions(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    filtered_payload JSONB,
    delivered BOOLEAN NOT NULL DEFAULT false,
    delivery_id UUID,
    received_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE
);

-- GraphQL federation sources for federated schema support
CREATE TABLE IF NOT EXISTS graphql_federation_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schema_id UUID NOT NULL REFERENCES graphql_schemas(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    endpoint_url VARCHAR(500) NOT NULL,
    subgraph_sdl TEXT,
    auth_config JSONB DEFAULT '{}',
    health_status VARCHAR(50) NOT NULL DEFAULT 'unknown',
    last_health_check TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(schema_id, name)
);

-- GraphQL type mappings for webhook event conversion
CREATE TABLE IF NOT EXISTS graphql_type_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schema_id UUID NOT NULL REFERENCES graphql_schemas(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    graphql_type VARCHAR(255) NOT NULL,
    webhook_event_type VARCHAR(255) NOT NULL,
    field_mappings JSONB NOT NULL DEFAULT '{}',
    auto_generated BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(schema_id, graphql_type)
);

-- Indexes for query optimization
CREATE INDEX IF NOT EXISTS idx_graphql_schemas_tenant ON graphql_schemas(tenant_id);
CREATE INDEX IF NOT EXISTS idx_graphql_schemas_status ON graphql_schemas(status);
CREATE INDEX IF NOT EXISTS idx_graphql_subscriptions_tenant ON graphql_subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_graphql_subscriptions_schema ON graphql_subscriptions(schema_id);
CREATE INDEX IF NOT EXISTS idx_graphql_subscriptions_status ON graphql_subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_graphql_subscription_events_sub ON graphql_subscription_events(subscription_id);
CREATE INDEX IF NOT EXISTS idx_graphql_subscription_events_delivered ON graphql_subscription_events(delivered, received_at);
CREATE INDEX IF NOT EXISTS idx_graphql_federation_sources_schema ON graphql_federation_sources(schema_id);
CREATE INDEX IF NOT EXISTS idx_graphql_type_mappings_schema ON graphql_type_mappings(schema_id);
