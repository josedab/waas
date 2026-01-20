-- Usage records table
CREATE TABLE IF NOT EXISTS usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    endpoint_id UUID,
    webhook_id UUID,
    unit VARCHAR(50) NOT NULL,
    quantity BIGINT NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_usage_records_tenant_id ON usage_records(tenant_id);
CREATE INDEX idx_usage_records_endpoint_id ON usage_records(endpoint_id);
CREATE INDEX idx_usage_records_unit ON usage_records(unit);
CREATE INDEX idx_usage_records_recorded_at ON usage_records(recorded_at);
CREATE INDEX idx_usage_records_tenant_recorded ON usage_records(tenant_id, recorded_at);

-- Cost allocations table
CREATE TABLE IF NOT EXISTS cost_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    period VARCHAR(7) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    resource_name VARCHAR(255),
    usage JSONB NOT NULL DEFAULT '{}',
    cost JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, period, resource_type, resource_id)
);

CREATE INDEX idx_cost_allocations_tenant_id ON cost_allocations(tenant_id);
CREATE INDEX idx_cost_allocations_period ON cost_allocations(period);
CREATE INDEX idx_cost_allocations_resource_type ON cost_allocations(resource_type);

-- Budgets table
CREATE TABLE IF NOT EXISTS budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    period VARCHAR(20) NOT NULL,
    resource_type VARCHAR(50),
    resource_id VARCHAR(255),
    alerts JSONB DEFAULT '[]',
    current_spend DECIMAL(15,2) DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_budgets_tenant_id ON budgets(tenant_id);
CREATE INDEX idx_budgets_is_active ON budgets(is_active);
CREATE INDEX idx_budgets_period ON budgets(period);

-- Pricing tiers table (for future customization)
CREATE TABLE IF NOT EXISTS pricing_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rates JSONB NOT NULL DEFAULT '[]',
    limits JSONB NOT NULL DEFAULT '{}',
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default pricing tier
INSERT INTO pricing_tiers (name, description, rates, limits, is_default) VALUES
('Standard', 'Standard pricing tier', 
 '[{"unit":"delivery","price":0.0001,"currency":"USD"},{"unit":"byte","price":0.00000001,"currency":"USD"},{"unit":"retry","price":0.00005,"currency":"USD"},{"unit":"transform","price":0.00002,"currency":"USD"}]',
 '{"monthly_deliveries":1000000,"monthly_bytes":10737418240,"max_endpoints":100,"max_retries":5}',
 true
);
