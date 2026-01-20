-- Cloud Platform Tables for Hosted SaaS Offering
-- Feature 1: Multi-tenant infrastructure, usage metering, self-service onboarding

-- Organizations (parent of tenants for enterprise accounts)
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    billing_email VARCHAR(255) NOT NULL,
    billing_address JSONB,
    stripe_customer_id VARCHAR(255),
    plan_id UUID,
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'cancelled', 'trial')),
    trial_ends_at TIMESTAMP WITH TIME ZONE,
    settings JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Subscription Plans
CREATE TABLE IF NOT EXISTS subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    price_monthly_cents INTEGER NOT NULL DEFAULT 0,
    price_yearly_cents INTEGER NOT NULL DEFAULT 0,
    stripe_price_id_monthly VARCHAR(255),
    stripe_price_id_yearly VARCHAR(255),
    limits JSONB NOT NULL DEFAULT '{
        "max_endpoints": 10,
        "max_deliveries_per_month": 10000,
        "max_payload_size_kb": 256,
        "max_retention_days": 7,
        "max_team_members": 3,
        "features": []
    }',
    is_public BOOLEAN DEFAULT true,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Usage Metering
CREATE TABLE IF NOT EXISTS usage_meters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    meter_type VARCHAR(50) NOT NULL CHECK (meter_type IN (
        'deliveries', 'bandwidth_bytes', 'endpoints', 'transformations', 
        'api_calls', 'storage_bytes', 'team_members'
    )),
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    usage_count BIGINT DEFAULT 0,
    usage_limit BIGINT,
    overage_count BIGINT DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(organization_id, tenant_id, meter_type, period_start)
);

-- Usage Events (granular tracking)
CREATE TABLE IF NOT EXISTS usage_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    quantity BIGINT DEFAULT 1,
    properties JSONB DEFAULT '{}',
    idempotency_key VARCHAR(255),
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(idempotency_key)
);

-- Onboarding State Machine
CREATE TABLE IF NOT EXISTS onboarding_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    verification_token VARCHAR(255),
    verification_expires_at TIMESTAMP WITH TIME ZONE,
    current_step VARCHAR(50) DEFAULT 'email_verification' CHECK (current_step IN (
        'email_verification', 'organization_setup', 'plan_selection', 
        'payment_setup', 'first_endpoint', 'completed'
    )),
    completed_steps JSONB DEFAULT '[]',
    form_data JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    referral_source VARCHAR(255),
    utm_params JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Team Members
CREATE TABLE IF NOT EXISTS team_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(50) DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    password_hash VARCHAR(255),
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'suspended')),
    invite_token VARCHAR(255),
    invite_expires_at TIMESTAMP WITH TIME ZONE,
    last_login_at TIMESTAMP WITH TIME ZONE,
    mfa_enabled BOOLEAN DEFAULT false,
    mfa_secret VARCHAR(255),
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(organization_id, email)
);

-- API Tokens (organization-level)
CREATE TABLE IF NOT EXISTS org_api_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    token_prefix VARCHAR(20) NOT NULL,
    scopes JSONB DEFAULT '["*"]',
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_by UUID REFERENCES team_members(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE
);

-- Regional Deployments
CREATE TABLE IF NOT EXISTS cloud_regions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('aws', 'gcp', 'azure')),
    location VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    is_default BOOLEAN DEFAULT false,
    latency_ms INTEGER,
    features JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tenant Region Assignments
CREATE TABLE IF NOT EXISTS tenant_regions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    region_id UUID NOT NULL REFERENCES cloud_regions(id) ON DELETE CASCADE,
    is_primary BOOLEAN DEFAULT false,
    data_residency_required BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, region_id)
);

-- Indexes
CREATE INDEX idx_organizations_slug ON organizations(slug);
CREATE INDEX idx_organizations_status ON organizations(status);
CREATE INDEX idx_usage_meters_org_period ON usage_meters(organization_id, period_start, period_end);
CREATE INDEX idx_usage_events_org_type ON usage_events(organization_id, event_type, recorded_at);
CREATE INDEX idx_usage_events_recorded_at ON usage_events(recorded_at);
CREATE INDEX idx_onboarding_sessions_email ON onboarding_sessions(email);
CREATE INDEX idx_team_members_email ON team_members(email);
CREATE INDEX idx_org_api_tokens_prefix ON org_api_tokens(token_prefix);
CREATE INDEX idx_tenant_regions_tenant ON tenant_regions(tenant_id);

-- Insert default plans
INSERT INTO subscription_plans (name, slug, description, price_monthly_cents, price_yearly_cents, limits, sort_order) VALUES
('Free', 'free', 'Perfect for getting started', 0, 0, 
 '{"max_endpoints": 3, "max_deliveries_per_month": 1000, "max_payload_size_kb": 64, "max_retention_days": 1, "max_team_members": 1, "features": ["basic_retries"]}', 1),
('Starter', 'starter', 'For small projects and teams', 2900, 29000, 
 '{"max_endpoints": 10, "max_deliveries_per_month": 25000, "max_payload_size_kb": 256, "max_retention_days": 7, "max_team_members": 3, "features": ["basic_retries", "transformations", "custom_headers"]}', 2),
('Pro', 'pro', 'For growing businesses', 9900, 99000, 
 '{"max_endpoints": 50, "max_deliveries_per_month": 250000, "max_payload_size_kb": 1024, "max_retention_days": 30, "max_team_members": 10, "features": ["basic_retries", "transformations", "custom_headers", "schema_validation", "analytics", "webhooks_for_webhooks"]}', 3),
('Enterprise', 'enterprise', 'For large scale deployments', 49900, 499000, 
 '{"max_endpoints": -1, "max_deliveries_per_month": -1, "max_payload_size_kb": 5120, "max_retention_days": 90, "max_team_members": -1, "features": ["basic_retries", "transformations", "custom_headers", "schema_validation", "analytics", "webhooks_for_webhooks", "geo_routing", "sla", "dedicated_support", "custom_domain"]}', 4);

-- Insert default regions
INSERT INTO cloud_regions (code, name, provider, location, is_default) VALUES
('us-east-1', 'US East (N. Virginia)', 'aws', 'Virginia, USA', true),
('us-west-2', 'US West (Oregon)', 'aws', 'Oregon, USA', false),
('eu-west-1', 'Europe (Ireland)', 'aws', 'Dublin, Ireland', false),
('eu-central-1', 'Europe (Frankfurt)', 'aws', 'Frankfurt, Germany', false),
('ap-southeast-1', 'Asia Pacific (Singapore)', 'aws', 'Singapore', false),
('ap-northeast-1', 'Asia Pacific (Tokyo)', 'aws', 'Tokyo, Japan', false);
