-- Create transformations table
CREATE TABLE IF NOT EXISTS transformations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    script TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    version INTEGER NOT NULL DEFAULT 1,
    timeout_ms INTEGER NOT NULL DEFAULT 5000,
    max_memory_mb INTEGER NOT NULL DEFAULT 64,
    allow_http BOOLEAN NOT NULL DEFAULT false,
    enable_logging BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create endpoint_transformations junction table
CREATE TABLE IF NOT EXISTS endpoint_transformations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    transformation_id UUID NOT NULL REFERENCES transformations(id) ON DELETE CASCADE,
    priority INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(endpoint_id, transformation_id)
);

-- Create transformation_logs table
CREATE TABLE IF NOT EXISTS transformation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transformation_id UUID NOT NULL REFERENCES transformations(id) ON DELETE CASCADE,
    delivery_id UUID NOT NULL,
    input_payload TEXT NOT NULL,
    output_payload TEXT,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    execution_time_ms INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_transformations_tenant_id ON transformations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_transformations_enabled ON transformations(enabled);
CREATE INDEX IF NOT EXISTS idx_endpoint_transformations_endpoint_id ON endpoint_transformations(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_endpoint_transformations_transformation_id ON endpoint_transformations(transformation_id);
CREATE INDEX IF NOT EXISTS idx_transformation_logs_transformation_id ON transformation_logs(transformation_id);
CREATE INDEX IF NOT EXISTS idx_transformation_logs_delivery_id ON transformation_logs(delivery_id);
CREATE INDEX IF NOT EXISTS idx_transformation_logs_created_at ON transformation_logs(created_at);
