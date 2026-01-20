-- AI Auto-Remediation tables

-- Remediation actions
CREATE TABLE IF NOT EXISTS remediation_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID,
    delivery_id UUID,
    error_pattern_id UUID,
    action_type VARCHAR(50) NOT NULL, -- update_url, rotate_credentials, adjust_timeout, etc.
    description TEXT NOT NULL,
    parameters JSONB NOT NULL DEFAULT '{}',
    confidence_score DECIMAL(5,4) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, approved, executing, completed, failed, rejected
    auto_execute BOOLEAN NOT NULL DEFAULT false,
    requires_approval BOOLEAN NOT NULL DEFAULT true,
    approved_by VARCHAR(255),
    approved_at TIMESTAMP WITH TIME ZONE,
    executed_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    result JSONB,
    error_message TEXT,
    rollback_data JSONB,
    rolled_back BOOLEAN NOT NULL DEFAULT false,
    rolled_back_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_remediation_actions_tenant ON remediation_actions(tenant_id);
CREATE INDEX idx_remediation_actions_endpoint ON remediation_actions(endpoint_id);
CREATE INDEX idx_remediation_actions_status ON remediation_actions(tenant_id, status);
CREATE INDEX idx_remediation_actions_type ON remediation_actions(tenant_id, action_type);
CREATE INDEX idx_remediation_actions_created ON remediation_actions(created_at);

-- Remediation rules
CREATE TABLE IF NOT EXISTS remediation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    error_pattern TEXT NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    action_params JSONB NOT NULL DEFAULT '{}',
    min_confidence DECIMAL(5,4) NOT NULL DEFAULT 0.8,
    auto_execute BOOLEAN NOT NULL DEFAULT false,
    max_executions_per_hour INT NOT NULL DEFAULT 10,
    cooldown_minutes INT NOT NULL DEFAULT 30,
    enabled BOOLEAN NOT NULL DEFAULT true,
    priority INT NOT NULL DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_remediation_rule_name UNIQUE(tenant_id, name)
);

CREATE INDEX idx_remediation_rules_tenant ON remediation_rules(tenant_id);
CREATE INDEX idx_remediation_rules_enabled ON remediation_rules(tenant_id, enabled);
CREATE INDEX idx_remediation_rules_type ON remediation_rules(tenant_id, action_type);

-- Remediation approvals
CREATE TABLE IF NOT EXISTS remediation_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    action_id UUID NOT NULL REFERENCES remediation_actions(id) ON DELETE CASCADE,
    approver_id VARCHAR(255) NOT NULL,
    approver_email VARCHAR(255),
    decision VARCHAR(20) NOT NULL, -- approved, rejected
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_remediation_approvals_action ON remediation_approvals(action_id);
CREATE INDEX idx_remediation_approvals_approver ON remediation_approvals(approver_id);

-- Remediation history (audit trail)
CREATE TABLE IF NOT EXISTS remediation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    action_id UUID NOT NULL REFERENCES remediation_actions(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL, -- created, approved, rejected, executed, completed, failed, rolled_back
    event_data JSONB DEFAULT '{}',
    actor VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_remediation_history_action ON remediation_history(action_id);
CREATE INDEX idx_remediation_history_tenant ON remediation_history(tenant_id, created_at);

-- Remediation metrics
CREATE TABLE IF NOT EXISTS remediation_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    date DATE NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    total_actions INT NOT NULL DEFAULT 0,
    auto_executed INT NOT NULL DEFAULT 0,
    approved INT NOT NULL DEFAULT 0,
    rejected INT NOT NULL DEFAULT 0,
    successful INT NOT NULL DEFAULT 0,
    failed INT NOT NULL DEFAULT 0,
    rolled_back INT NOT NULL DEFAULT 0,
    avg_confidence DECIMAL(5,4),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_remediation_metrics UNIQUE(tenant_id, date, action_type)
);

CREATE INDEX idx_remediation_metrics_tenant_date ON remediation_metrics(tenant_id, date);

-- Trigger for updated_at
CREATE TRIGGER trigger_remediation_actions_updated
    BEFORE UPDATE ON remediation_actions
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();

CREATE TRIGGER trigger_remediation_rules_updated
    BEFORE UPDATE ON remediation_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();
