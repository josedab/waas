ALTER TABLE tenants ADD COLUMN IF NOT EXISTS api_key_lookup_hash VARCHAR(64);
CREATE INDEX IF NOT EXISTS idx_tenants_api_key_lookup ON tenants(api_key_lookup_hash);
