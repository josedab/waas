-- Create quota_usage table
CREATE TABLE quota_usage (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    month TIMESTAMP NOT NULL,
    request_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    failure_count INTEGER NOT NULL DEFAULT 0,
    overage_count INTEGER NOT NULL DEFAULT 0,
    last_updated TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, month)
);

-- Create billing_records table
CREATE TABLE billing_records (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    billing_period TIMESTAMP NOT NULL,
    base_requests INTEGER NOT NULL DEFAULT 0,
    overage_requests INTEGER NOT NULL DEFAULT 0,
    base_amount INTEGER NOT NULL DEFAULT 0,
    overage_amount INTEGER NOT NULL DEFAULT 0,
    total_amount INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, billing_period)
);

-- Create quota_notifications table
CREATE TABLE quota_notifications (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    threshold INTEGER NOT NULL DEFAULT 0,
    usage_count INTEGER NOT NULL DEFAULT 0,
    quota_limit INTEGER NOT NULL DEFAULT 0,
    sent BOOLEAN NOT NULL DEFAULT FALSE,
    sent_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_quota_usage_tenant_month ON quota_usage(tenant_id, month);
CREATE INDEX idx_quota_usage_last_updated ON quota_usage(last_updated);

CREATE INDEX idx_billing_records_tenant_period ON billing_records(tenant_id, billing_period);
CREATE INDEX idx_billing_records_status ON billing_records(status);
CREATE INDEX idx_billing_records_created_at ON billing_records(created_at);

CREATE INDEX idx_quota_notifications_tenant_sent ON quota_notifications(tenant_id, sent);
CREATE INDEX idx_quota_notifications_type ON quota_notifications(type);
CREATE INDEX idx_quota_notifications_created_at ON quota_notifications(created_at);

-- Add check constraints
ALTER TABLE quota_usage ADD CONSTRAINT chk_quota_usage_counts 
    CHECK (request_count >= 0 AND success_count >= 0 AND failure_count >= 0 AND overage_count >= 0);

ALTER TABLE billing_records ADD CONSTRAINT chk_billing_amounts 
    CHECK (base_amount >= 0 AND overage_amount >= 0 AND total_amount >= 0);

ALTER TABLE billing_records ADD CONSTRAINT chk_billing_status 
    CHECK (status IN ('pending', 'processed', 'paid', 'failed'));

ALTER TABLE quota_notifications ADD CONSTRAINT chk_notification_threshold 
    CHECK (threshold >= 0 AND threshold <= 200);

ALTER TABLE quota_notifications ADD CONSTRAINT chk_notification_type 
    CHECK (type IN ('warning', 'approaching_limit', 'limit_reached', 'overage', 'subscription_change'));