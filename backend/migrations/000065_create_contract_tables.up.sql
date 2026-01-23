CREATE TABLE IF NOT EXISTS webhook_contracts (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    endpoint_id VARCHAR(36),
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    schema JSONB NOT NULL,
    strictness VARCHAR(20) NOT NULL DEFAULT 'standard',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS contract_test_results (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) NOT NULL,
    contract_id VARCHAR(36) NOT NULL REFERENCES webhook_contracts(id) ON DELETE CASCADE,
    endpoint_id VARCHAR(36),
    passed BOOLEAN NOT NULL,
    violations JSONB NOT NULL DEFAULT '[]',
    tested_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    duration_ms INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_contracts_tenant ON webhook_contracts(tenant_id);
CREATE INDEX idx_contracts_event_type ON webhook_contracts(tenant_id, event_type);
CREATE INDEX idx_contract_results_contract ON contract_test_results(contract_id);
CREATE INDEX idx_contract_results_tenant ON contract_test_results(tenant_id);
