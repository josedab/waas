CREATE TABLE IF NOT EXISTS terraform_managed_resources (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(36) NOT NULL,
    name VARCHAR(255),
    state VARCHAR(20) NOT NULL DEFAULT 'managed',
    attributes JSONB NOT NULL DEFAULT '{}',
    managed_by VARCHAR(20) NOT NULL DEFAULT 'terraform',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, resource_type, resource_id)
);

CREATE INDEX idx_tf_resources_tenant ON terraform_managed_resources(tenant_id);
CREATE INDEX idx_tf_resources_type ON terraform_managed_resources(tenant_id, resource_type);
