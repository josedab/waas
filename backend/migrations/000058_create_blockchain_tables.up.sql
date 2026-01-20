-- Blockchain / Smart Contract Triggers tables

-- Contract monitors
CREATE TABLE IF NOT EXISTS contract_monitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    chain VARCHAR(50) NOT NULL, -- ethereum, polygon, arbitrum, optimism, base, bsc, avalanche, solana
    chain_id BIGINT NOT NULL,
    contract_address VARCHAR(100) NOT NULL,
    abi JSONB NOT NULL DEFAULT '[]',
    events_to_monitor JSONB NOT NULL DEFAULT '[]',
    start_block BIGINT,
    current_block BIGINT,
    confirmation_blocks INT NOT NULL DEFAULT 12,
    status VARCHAR(20) NOT NULL DEFAULT 'inactive', -- active, inactive, paused, error
    error_message TEXT,
    webhook_url TEXT,
    webhook_secret VARCHAR(255),
    filter_config JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    last_poll_at TIMESTAMP WITH TIME ZONE,
    events_processed BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_contract_monitor UNIQUE(tenant_id, chain, contract_address)
);

CREATE INDEX idx_contract_monitors_tenant ON contract_monitors(tenant_id);
CREATE INDEX idx_contract_monitors_chain ON contract_monitors(chain);
CREATE INDEX idx_contract_monitors_status ON contract_monitors(tenant_id, status);
CREATE INDEX idx_contract_monitors_address ON contract_monitors(contract_address);

-- Blockchain events
CREATE TABLE IF NOT EXISTS blockchain_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    monitor_id UUID NOT NULL REFERENCES contract_monitors(id) ON DELETE CASCADE,
    chain VARCHAR(50) NOT NULL,
    chain_id BIGINT NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash VARCHAR(100) NOT NULL,
    transaction_hash VARCHAR(100) NOT NULL,
    transaction_index INT NOT NULL,
    log_index INT NOT NULL,
    contract_address VARCHAR(100) NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    event_signature VARCHAR(100),
    topics JSONB NOT NULL DEFAULT '[]',
    data TEXT,
    decoded_data JSONB,
    timestamp TIMESTAMP WITH TIME ZONE,
    confirmed BOOLEAN NOT NULL DEFAULT false,
    confirmation_count INT NOT NULL DEFAULT 0,
    reorged BOOLEAN NOT NULL DEFAULT false,
    webhook_sent BOOLEAN NOT NULL DEFAULT false,
    webhook_sent_at TIMESTAMP WITH TIME ZONE,
    webhook_status VARCHAR(20),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_blockchain_events_monitor ON blockchain_events(monitor_id);
CREATE INDEX idx_blockchain_events_block ON blockchain_events(chain, block_number);
CREATE INDEX idx_blockchain_events_tx ON blockchain_events(transaction_hash);
CREATE INDEX idx_blockchain_events_contract ON blockchain_events(contract_address, event_name);
CREATE INDEX idx_blockchain_events_confirmed ON blockchain_events(tenant_id, confirmed);
CREATE INDEX idx_blockchain_events_created ON blockchain_events(created_at);

-- Chain configurations
CREATE TABLE IF NOT EXISTS chain_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    chain VARCHAR(50) NOT NULL,
    chain_id BIGINT NOT NULL,
    rpc_urls JSONB NOT NULL DEFAULT '[]',
    ws_urls JSONB DEFAULT '[]',
    block_time_ms INT NOT NULL DEFAULT 12000,
    confirmation_blocks INT NOT NULL DEFAULT 12,
    max_blocks_per_query INT NOT NULL DEFAULT 1000,
    rate_limit_rps INT NOT NULL DEFAULT 10,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_chain_config UNIQUE(tenant_id, chain)
);

CREATE INDEX idx_chain_configs_tenant ON chain_configs(tenant_id);

-- Block tracking
CREATE TABLE IF NOT EXISTS blockchain_blocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chain VARCHAR(50) NOT NULL,
    chain_id BIGINT NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash VARCHAR(100) NOT NULL,
    parent_hash VARCHAR(100),
    timestamp TIMESTAMP WITH TIME ZONE,
    processed BOOLEAN NOT NULL DEFAULT false,
    reorged BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_block UNIQUE(chain, block_number)
);

CREATE INDEX idx_blockchain_blocks_chain ON blockchain_blocks(chain, block_number DESC);
CREATE INDEX idx_blockchain_blocks_hash ON blockchain_blocks(block_hash);

-- Trigger for updated_at
CREATE TRIGGER trigger_contract_monitors_updated
    BEFORE UPDATE ON contract_monitors
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();

CREATE TRIGGER trigger_chain_configs_updated
    BEFORE UPDATE ON chain_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();
