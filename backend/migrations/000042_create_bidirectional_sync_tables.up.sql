-- Feature 8: Bi-Directional Webhook Sync
-- Two-way webhook synchronization with request-response patterns

-- Sync configurations for bi-directional communication
CREATE TABLE IF NOT EXISTS webhook_sync_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    outbound_endpoint_id UUID REFERENCES endpoints(id) ON DELETE SET NULL,
    inbound_event_type VARCHAR(255),
    sync_mode VARCHAR(50) NOT NULL DEFAULT 'request_response', -- request_response, event_acknowledgment, state_sync
    timeout_seconds INTEGER NOT NULL DEFAULT 30,
    retry_on_timeout BOOLEAN NOT NULL DEFAULT true,
    max_retries INTEGER NOT NULL DEFAULT 3,
    correlation_config JSONB DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Sync transactions tracking request-response pairs
CREATE TABLE IF NOT EXISTS sync_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    config_id UUID NOT NULL REFERENCES webhook_sync_configs(id) ON DELETE CASCADE,
    correlation_id VARCHAR(255) NOT NULL,
    outbound_event_id UUID,
    inbound_event_id UUID,
    state VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, awaiting_response, completed, timeout, failed
    request_payload JSONB,
    response_payload JSONB,
    request_sent_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    response_received_at TIMESTAMP WITH TIME ZONE,
    timeout_at TIMESTAMP WITH TIME ZONE,
    retry_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- State synchronization records for state_sync mode
CREATE TABLE IF NOT EXISTS sync_state_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    config_id UUID NOT NULL REFERENCES webhook_sync_configs(id) ON DELETE CASCADE,
    resource_type VARCHAR(255) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    local_state JSONB NOT NULL,
    remote_state JSONB,
    last_local_update TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_remote_update TIMESTAMP WITH TIME ZONE,
    sync_status VARCHAR(50) NOT NULL DEFAULT 'synced', -- synced, pending_push, pending_pull, conflict
    conflict_data JSONB,
    conflict_resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(config_id, resource_type, resource_id)
);

-- Acknowledgment tracking for event_acknowledgment mode
CREATE TABLE IF NOT EXISTS sync_acknowledgments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    config_id UUID NOT NULL REFERENCES webhook_sync_configs(id) ON DELETE CASCADE,
    event_id UUID NOT NULL,
    correlation_id VARCHAR(255) NOT NULL,
    ack_type VARCHAR(50) NOT NULL, -- received, processed, rejected
    ack_payload JSONB,
    sent_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    timeout_at TIMESTAMP WITH TIME ZONE,
    retry_count INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, acknowledged, timeout, failed
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Conflict resolution history
CREATE TABLE IF NOT EXISTS sync_conflict_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    state_record_id UUID NOT NULL REFERENCES sync_state_records(id) ON DELETE CASCADE,
    local_state JSONB NOT NULL,
    remote_state JSONB NOT NULL,
    resolution_strategy VARCHAR(100) NOT NULL, -- local_wins, remote_wins, merge, manual
    resolved_state JSONB,
    resolved_by UUID,
    resolved_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for query optimization
CREATE INDEX IF NOT EXISTS idx_sync_configs_tenant ON webhook_sync_configs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sync_configs_enabled ON webhook_sync_configs(enabled);
CREATE INDEX IF NOT EXISTS idx_sync_transactions_config ON sync_transactions(config_id);
CREATE INDEX IF NOT EXISTS idx_sync_transactions_correlation ON sync_transactions(correlation_id);
CREATE INDEX IF NOT EXISTS idx_sync_transactions_state ON sync_transactions(state);
CREATE INDEX IF NOT EXISTS idx_sync_transactions_timeout ON sync_transactions(timeout_at) WHERE state = 'awaiting_response';
CREATE INDEX IF NOT EXISTS idx_sync_state_records_config ON sync_state_records(config_id);
CREATE INDEX IF NOT EXISTS idx_sync_state_records_resource ON sync_state_records(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_sync_state_records_status ON sync_state_records(sync_status);
CREATE INDEX IF NOT EXISTS idx_sync_acknowledgments_config ON sync_acknowledgments(config_id);
CREATE INDEX IF NOT EXISTS idx_sync_acknowledgments_correlation ON sync_acknowledgments(correlation_id);
CREATE INDEX IF NOT EXISTS idx_sync_conflict_history_record ON sync_conflict_history(state_record_id);
