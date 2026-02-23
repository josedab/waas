-- Interactive Onboarding & Guided Setup Wizard

CREATE TABLE IF NOT EXISTS onboarding_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       VARCHAR(255) NOT NULL,
    current_step    VARCHAR(100) NOT NULL DEFAULT 'create_tenant',
    completed_steps JSONB NOT NULL DEFAULT '[]',
    metadata        JSONB NOT NULL DEFAULT '{}',
    is_complete     BOOLEAN NOT NULL DEFAULT false,
    started_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at    TIMESTAMP WITH TIME ZONE,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_onboarding_tenant ON onboarding_sessions(tenant_id);
CREATE INDEX idx_onboarding_complete ON onboarding_sessions(is_complete);
