-- Create secrets table for secure secret storage and rotation
CREATE TABLE IF NOT EXISTS secret_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    secret_id VARCHAR(255) NOT NULL,
    version INTEGER NOT NULL,
    encrypted_value TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ensure unique version per tenant/secret combination
    UNIQUE(tenant_id, secret_id, version)
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_secret_versions_tenant_secret ON secret_versions(tenant_id, secret_id);
CREATE INDEX IF NOT EXISTS idx_secret_versions_active ON secret_versions(tenant_id, secret_id, is_active);
CREATE INDEX IF NOT EXISTS idx_secret_versions_expires ON secret_versions(expires_at) WHERE expires_at IS NOT NULL;

-- Add encrypted payload storage to delivery attempts
ALTER TABLE delivery_attempts 
ADD COLUMN IF NOT EXISTS encrypted_payload TEXT,
ADD COLUMN IF NOT EXISTS payload_encryption_key_id UUID;

-- Create index for payload encryption key lookups
CREATE INDEX IF NOT EXISTS idx_delivery_attempts_encryption_key ON delivery_attempts(payload_encryption_key_id);