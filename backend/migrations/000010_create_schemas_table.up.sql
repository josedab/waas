-- Schemas table for webhook payload schemas
CREATE TABLE IF NOT EXISTS schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    description TEXT,
    json_schema JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- Schema versions table for version history
CREATE TABLE IF NOT EXISTS schema_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schema_id UUID NOT NULL REFERENCES schemas(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    json_schema JSONB NOT NULL,
    changelog TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    UNIQUE(schema_id, version)
);

-- Endpoint schema assignments
CREATE TABLE IF NOT EXISTS endpoint_schemas (
    endpoint_id UUID PRIMARY KEY REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    schema_id UUID NOT NULL REFERENCES schemas(id) ON DELETE CASCADE,
    schema_version VARCHAR(50),
    validation_mode VARCHAR(20) NOT NULL DEFAULT 'warn',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_schemas_tenant_id ON schemas(tenant_id);
CREATE INDEX idx_schemas_name ON schemas(tenant_id, name);
CREATE INDEX idx_schema_versions_schema_id ON schema_versions(schema_id);
CREATE INDEX idx_endpoint_schemas_schema_id ON endpoint_schemas(schema_id);
