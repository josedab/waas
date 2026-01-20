-- Mock endpoints table
CREATE TABLE IF NOT EXISTS mock_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    url TEXT NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    template JSONB,
    schedule JSONB,
    settings JSONB NOT NULL DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_mock_endpoints_tenant_id ON mock_endpoints(tenant_id);
CREATE INDEX idx_mock_endpoints_event_type ON mock_endpoints(event_type);
CREATE INDEX idx_mock_endpoints_is_active ON mock_endpoints(is_active);

-- Mock deliveries table
CREATE TABLE IF NOT EXISTS mock_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL REFERENCES mock_endpoints(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    headers JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    status_code INTEGER,
    response_body TEXT,
    error TEXT,
    latency_ms INTEGER DEFAULT 0,
    scheduled_at TIMESTAMP WITH TIME ZONE,
    sent_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_mock_deliveries_endpoint_id ON mock_deliveries(endpoint_id);
CREATE INDEX idx_mock_deliveries_tenant_id ON mock_deliveries(tenant_id);
CREATE INDEX idx_mock_deliveries_status ON mock_deliveries(status);
CREATE INDEX idx_mock_deliveries_created_at ON mock_deliveries(created_at);

-- Mock templates table
CREATE TABLE IF NOT EXISTS mock_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    event_type VARCHAR(255) NOT NULL,
    category VARCHAR(100),
    template JSONB NOT NULL DEFAULT '{}',
    examples JSONB DEFAULT '[]',
    is_public BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_mock_templates_tenant_id ON mock_templates(tenant_id);
CREATE INDEX idx_mock_templates_event_type ON mock_templates(event_type);
CREATE INDEX idx_mock_templates_category ON mock_templates(category);
CREATE INDEX idx_mock_templates_is_public ON mock_templates(is_public);
