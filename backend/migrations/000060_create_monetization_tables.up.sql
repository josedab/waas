-- Webhook Monetization Platform tables

-- Pricing plans
CREATE TABLE IF NOT EXISTS monetization_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    pricing_model VARCHAR(50) NOT NULL DEFAULT 'usage_based', -- usage_based, tiered, flat_rate, hybrid
    billing_period VARCHAR(20) NOT NULL DEFAULT 'monthly', -- monthly, annual, weekly
    base_price BIGINT NOT NULL DEFAULT 0, -- In cents
    price_per_webhook BIGINT NOT NULL DEFAULT 0, -- In cents
    included_webhooks BIGINT NOT NULL DEFAULT 0,
    tiers JSONB DEFAULT '[]',
    features JSONB NOT NULL DEFAULT '{}',
    limits JSONB NOT NULL DEFAULT '{}',
    trial_days INT NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT true,
    public BOOLEAN NOT NULL DEFAULT false,
    sort_order INT NOT NULL DEFAULT 100,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_monetization_plan_name UNIQUE(tenant_id, name)
);

CREATE INDEX idx_monetization_plans_tenant ON monetization_plans(tenant_id);
CREATE INDEX idx_monetization_plans_active ON monetization_plans(tenant_id, active);
CREATE INDEX idx_monetization_plans_public ON monetization_plans(tenant_id, public);

-- Customers (end customers of the tenant)
CREATE TABLE IF NOT EXISTS monetization_customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    external_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    company VARCHAR(255),
    stripe_id VARCHAR(100),
    billing_email VARCHAR(255),
    payment_method VARCHAR(50),
    currency VARCHAR(3) NOT NULL DEFAULT 'usd',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_monetization_customer UNIQUE(tenant_id, external_id)
);

CREATE INDEX idx_monetization_customers_tenant ON monetization_customers(tenant_id);
CREATE INDEX idx_monetization_customers_email ON monetization_customers(email);
CREATE INDEX idx_monetization_customers_stripe ON monetization_customers(stripe_id);

-- Customer subscriptions
CREATE TABLE IF NOT EXISTS monetization_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    customer_id UUID NOT NULL REFERENCES monetization_customers(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES monetization_plans(id),
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, paused, cancelled, trialing, past_due
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    cancel_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    trial_start TIMESTAMP WITH TIME ZONE,
    trial_end TIMESTAMP WITH TIME ZONE,
    stripe_sub_id VARCHAR(100),
    quantity INT NOT NULL DEFAULT 1,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_monetization_subscriptions_tenant ON monetization_subscriptions(tenant_id);
CREATE INDEX idx_monetization_subscriptions_customer ON monetization_subscriptions(customer_id);
CREATE INDEX idx_monetization_subscriptions_status ON monetization_subscriptions(tenant_id, status);
CREATE INDEX idx_monetization_subscriptions_period ON monetization_subscriptions(current_period_end);

-- Customer API keys
CREATE TABLE IF NOT EXISTS monetization_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    customer_id UUID NOT NULL REFERENCES monetization_customers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(20) NOT NULL DEFAULT 'whk_live_',
    key_hash VARCHAR(128) NOT NULL,
    last_chars VARCHAR(4) NOT NULL,
    scopes JSONB DEFAULT '[]',
    rate_limit INT NOT NULL DEFAULT 100,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_monetization_api_keys_tenant ON monetization_api_keys(tenant_id);
CREATE INDEX idx_monetization_api_keys_customer ON monetization_api_keys(customer_id);
CREATE INDEX idx_monetization_api_keys_hash ON monetization_api_keys(key_hash);
CREATE INDEX idx_monetization_api_keys_active ON monetization_api_keys(tenant_id, active);

-- Usage records
CREATE TABLE IF NOT EXISTS monetization_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    customer_id UUID NOT NULL REFERENCES monetization_customers(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES monetization_subscriptions(id) ON DELETE CASCADE,
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    webhooks_sent BIGINT NOT NULL DEFAULT 0,
    webhooks_success BIGINT NOT NULL DEFAULT 0,
    webhooks_failed BIGINT NOT NULL DEFAULT 0,
    bytes_transferred BIGINT NOT NULL DEFAULT 0,
    unique_endpoints INT NOT NULL DEFAULT 0,
    cost BIGINT NOT NULL DEFAULT 0, -- In cents
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_usage_period UNIQUE(subscription_id, period_start)
);

CREATE INDEX idx_monetization_usage_subscription ON monetization_usage(subscription_id);
CREATE INDEX idx_monetization_usage_customer ON monetization_usage(customer_id);
CREATE INDEX idx_monetization_usage_period ON monetization_usage(period_start, period_end);

-- Invoices
CREATE TABLE IF NOT EXISTS monetization_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    customer_id UUID NOT NULL REFERENCES monetization_customers(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES monetization_subscriptions(id),
    number VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft', -- draft, open, paid, void, uncollectible
    currency VARCHAR(3) NOT NULL DEFAULT 'usd',
    subtotal BIGINT NOT NULL DEFAULT 0,
    tax BIGINT NOT NULL DEFAULT 0,
    total BIGINT NOT NULL DEFAULT 0,
    amount_paid BIGINT NOT NULL DEFAULT 0,
    amount_due BIGINT NOT NULL DEFAULT 0,
    line_items JSONB NOT NULL DEFAULT '[]',
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    due_date TIMESTAMP WITH TIME ZONE NOT NULL,
    paid_at TIMESTAMP WITH TIME ZONE,
    stripe_invoice_id VARCHAR(100),
    pdf_url VARCHAR(500),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_invoice_number UNIQUE(tenant_id, number)
);

CREATE INDEX idx_monetization_invoices_tenant ON monetization_invoices(tenant_id);
CREATE INDEX idx_monetization_invoices_customer ON monetization_invoices(customer_id);
CREATE INDEX idx_monetization_invoices_status ON monetization_invoices(tenant_id, status);
CREATE INDEX idx_monetization_invoices_period ON monetization_invoices(period_start, period_end);

-- Invoice counter for generating sequential numbers
CREATE TABLE IF NOT EXISTS monetization_invoice_counters (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id),
    year INT NOT NULL,
    counter INT NOT NULL DEFAULT 0
);

-- Function to get next invoice number
CREATE OR REPLACE FUNCTION get_next_invoice_number(p_tenant_id UUID)
RETURNS VARCHAR(50) AS $$
DECLARE
    v_year INT;
    v_counter INT;
BEGIN
    v_year := EXTRACT(YEAR FROM NOW());
    
    INSERT INTO monetization_invoice_counters (tenant_id, year, counter)
    VALUES (p_tenant_id, v_year, 1)
    ON CONFLICT (tenant_id) DO UPDATE SET
        counter = CASE 
            WHEN monetization_invoice_counters.year = v_year 
            THEN monetization_invoice_counters.counter + 1
            ELSE 1
        END,
        year = v_year
    RETURNING counter INTO v_counter;
    
    RETURN 'INV-' || v_year || '-' || LPAD(v_counter::TEXT, 4, '0');
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
CREATE TRIGGER trigger_monetization_plans_updated
    BEFORE UPDATE ON monetization_plans
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();

CREATE TRIGGER trigger_monetization_customers_updated
    BEFORE UPDATE ON monetization_customers
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();

CREATE TRIGGER trigger_monetization_subscriptions_updated
    BEFORE UPDATE ON monetization_subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();

CREATE TRIGGER trigger_monetization_api_keys_updated
    BEFORE UPDATE ON monetization_api_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();

CREATE TRIGGER trigger_monetization_usage_updated
    BEFORE UPDATE ON monetization_usage
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();
