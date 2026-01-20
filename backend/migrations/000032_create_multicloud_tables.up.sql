-- Multi-Cloud Gateway Tables
-- Feature 8: AWS EventBridge, Azure Event Grid, GCP Pub/Sub connectors

-- Cloud Connectors
CREATE TABLE IF NOT EXISTS cloud_connectors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    name VARCHAR(255) NOT NULL,
    description TEXT,
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('aws_eventbridge', 'azure_eventgrid', 'gcp_pubsub', 'kafka', 'rabbitmq', 'custom')),
    
    -- Connection configuration (encrypted)
    config_encrypted JSONB NOT NULL,
    
    -- Status
    status VARCHAR(50) DEFAULT 'inactive' CHECK (status IN ('inactive', 'active', 'error', 'testing')),
    last_health_check TIMESTAMP WITH TIME ZONE,
    health_status VARCHAR(50),
    error_message TEXT,
    
    -- Metadata
    tags JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(tenant_id, name)
);

-- Connector Routes (maps webhooks to cloud destinations)
CREATE TABLE IF NOT EXISTS connector_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    connector_id UUID NOT NULL REFERENCES cloud_connectors(id) ON DELETE CASCADE,
    
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Source filter
    source_filter JSONB DEFAULT '{}', -- event_types, endpoint_ids, etc.
    
    -- Destination config
    destination_config JSONB NOT NULL, -- bus name, topic, queue, etc.
    
    -- Transformation
    transform_enabled BOOLEAN DEFAULT false,
    transform_script TEXT,
    
    -- Options
    is_active BOOLEAN DEFAULT true,
    batch_enabled BOOLEAN DEFAULT false,
    batch_size INTEGER DEFAULT 1,
    batch_window_seconds INTEGER DEFAULT 60,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Cloud Deliveries (audit log for cloud sends)
CREATE TABLE IF NOT EXISTS cloud_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    connector_id UUID NOT NULL,
    route_id UUID NOT NULL,
    
    -- Original event info
    original_delivery_id UUID,
    event_type VARCHAR(255),
    
    -- Cloud-specific identifiers
    cloud_message_id VARCHAR(255),
    cloud_request_id VARCHAR(255),
    
    -- Payload
    payload_hash VARCHAR(64),
    payload_size_bytes INTEGER,
    
    -- Status
    status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'sent', 'delivered', 'failed')),
    http_status_code INTEGER,
    error_code VARCHAR(100),
    error_message TEXT,
    
    -- Timing
    sent_at TIMESTAMP WITH TIME ZONE,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    latency_ms INTEGER,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Connector Credentials (secure storage for cloud credentials)
CREATE TABLE IF NOT EXISTS connector_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connector_id UUID NOT NULL REFERENCES cloud_connectors(id) ON DELETE CASCADE,
    
    credential_type VARCHAR(50) NOT NULL CHECK (credential_type IN ('api_key', 'access_key', 'service_account', 'oauth', 'connection_string')),
    credential_data_encrypted TEXT NOT NULL,
    
    -- Rotation tracking
    version INTEGER DEFAULT 1,
    expires_at TIMESTAMP WITH TIME ZONE,
    rotated_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Dead Letter Queue for failed cloud deliveries
CREATE TABLE IF NOT EXISTS cloud_dlq (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    connector_id UUID NOT NULL,
    route_id UUID NOT NULL,
    original_delivery_id UUID,
    
    payload JSONB NOT NULL,
    headers JSONB,
    
    failure_reason VARCHAR(255),
    failure_details TEXT,
    attempt_count INTEGER DEFAULT 1,
    
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'retrying', 'resolved', 'discarded')),
    resolved_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_cloud_connectors_tenant ON cloud_connectors(tenant_id);
CREATE INDEX idx_cloud_connectors_provider ON cloud_connectors(provider);
CREATE INDEX idx_connector_routes_connector ON connector_routes(connector_id);
CREATE INDEX idx_connector_routes_tenant ON connector_routes(tenant_id);
CREATE INDEX idx_cloud_deliveries_connector ON cloud_deliveries(connector_id, created_at);
CREATE INDEX idx_cloud_deliveries_tenant ON cloud_deliveries(tenant_id, created_at);
CREATE INDEX idx_cloud_dlq_status ON cloud_dlq(tenant_id, status);
