-- Billing tables for real-time spend tracking and alerts

-- Usage records
CREATE TABLE IF NOT EXISTS billing_usage (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    webhook_id UUID,
    resource_type VARCHAR(50) NOT NULL,
    quantity BIGINT NOT NULL DEFAULT 0,
    unit_cost DECIMAL(12, 6) NOT NULL DEFAULT 0,
    total_cost DECIMAL(12, 6) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    billing_period VARCHAR(7) NOT NULL, -- YYYY-MM
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_billing_usage_webhook FOREIGN KEY (webhook_id) 
        REFERENCES webhooks(id) ON DELETE SET NULL
);

CREATE INDEX idx_billing_usage_tenant_period ON billing_usage(tenant_id, billing_period);
CREATE INDEX idx_billing_usage_resource ON billing_usage(tenant_id, resource_type, billing_period);
CREATE INDEX idx_billing_usage_recorded ON billing_usage(recorded_at);

-- Spend trackers
CREATE TABLE IF NOT EXISTS billing_spend_trackers (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    budget_limit DECIMAL(12, 2) NOT NULL DEFAULT 0,
    current_spend DECIMAL(12, 2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    period VARCHAR(20) NOT NULL, -- daily, weekly, monthly
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    breakdown JSONB DEFAULT '{}',
    alerts JSONB DEFAULT '[]',
    status VARCHAR(20) NOT NULL DEFAULT 'normal',
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, period, period_start)
);

CREATE INDEX idx_billing_spend_tenant ON billing_spend_trackers(tenant_id);
CREATE INDEX idx_billing_spend_period ON billing_spend_trackers(tenant_id, period);

-- Budget configurations
CREATE TABLE IF NOT EXISTS billing_budgets (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    amount DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    period VARCHAR(20) NOT NULL, -- daily, weekly, monthly
    resource_type VARCHAR(50),
    webhook_id UUID,
    alerts JSONB DEFAULT '[]',
    auto_pause BOOLEAN NOT NULL DEFAULT false,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_billing_budgets_tenant ON billing_budgets(tenant_id);
CREATE INDEX idx_billing_budgets_enabled ON billing_budgets(tenant_id, enabled);

-- Billing alerts
CREATE TABLE IF NOT EXISTS billing_alerts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'info',
    title VARCHAR(500) NOT NULL,
    message TEXT NOT NULL,
    data JSONB DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    channels JSONB DEFAULT '[]',
    sent_at TIMESTAMP WITH TIME ZONE,
    acked_at TIMESTAMP WITH TIME ZONE,
    acked_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_billing_alerts_tenant ON billing_alerts(tenant_id);
CREATE INDEX idx_billing_alerts_status ON billing_alerts(tenant_id, status);
CREATE INDEX idx_billing_alerts_type ON billing_alerts(tenant_id, type);
CREATE INDEX idx_billing_alerts_created ON billing_alerts(created_at);

-- Cost optimizations
CREATE TABLE IF NOT EXISTS billing_optimizations (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    type VARCHAR(50) NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    estimated_savings DECIMAL(12, 2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    impact VARCHAR(20) NOT NULL DEFAULT 'medium',
    resource_id UUID,
    resource_type VARCHAR(50),
    actions JSONB DEFAULT '[]',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    implemented_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_billing_optimizations_tenant ON billing_optimizations(tenant_id);
CREATE INDEX idx_billing_optimizations_status ON billing_optimizations(tenant_id, status);

-- Invoices
CREATE TABLE IF NOT EXISTS billing_invoices (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    number VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    period VARCHAR(7) NOT NULL, -- YYYY-MM
    subtotal DECIMAL(12, 2) NOT NULL DEFAULT 0,
    discount DECIMAL(12, 2) NOT NULL DEFAULT 0,
    tax DECIMAL(12, 2) NOT NULL DEFAULT 0,
    total DECIMAL(12, 2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    line_items JSONB DEFAULT '[]',
    due_date TIMESTAMP WITH TIME ZONE NOT NULL,
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, number)
);

CREATE INDEX idx_billing_invoices_tenant ON billing_invoices(tenant_id);
CREATE INDEX idx_billing_invoices_status ON billing_invoices(tenant_id, status);
CREATE INDEX idx_billing_invoices_period ON billing_invoices(tenant_id, period);

-- Alert configurations
CREATE TABLE IF NOT EXISTS billing_alert_configs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) UNIQUE,
    enabled BOOLEAN NOT NULL DEFAULT true,
    channels JSONB DEFAULT '["email"]',
    recipients JSONB DEFAULT '[]',
    schedule JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Trigger for usage aggregation updates
CREATE OR REPLACE FUNCTION update_spend_tracker()
RETURNS TRIGGER AS $$
BEGIN
    -- Update or create spend tracker for the period
    INSERT INTO billing_spend_trackers (
        id, tenant_id, budget_limit, current_spend, currency, period,
        period_start, period_end, status, updated_at
    )
    VALUES (
        gen_random_uuid(),
        NEW.tenant_id,
        0, -- Will be set from budget
        NEW.total_cost,
        NEW.currency,
        'monthly',
        date_trunc('month', NEW.recorded_at),
        date_trunc('month', NEW.recorded_at) + interval '1 month' - interval '1 second',
        'normal',
        NOW()
    )
    ON CONFLICT (tenant_id, period, period_start) DO UPDATE SET
        current_spend = billing_spend_trackers.current_spend + NEW.total_cost,
        updated_at = NOW();
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_spend_tracker
    AFTER INSERT ON billing_usage
    FOR EACH ROW
    EXECUTE FUNCTION update_spend_tracker();
