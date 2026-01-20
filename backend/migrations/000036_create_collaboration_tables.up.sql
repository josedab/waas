-- Developer Collaboration Hub tables
-- Team workspaces, change reviews, and activity feeds

CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE TABLE team_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    permissions JSONB NOT NULL DEFAULT '[]',
    invited_by UUID,
    joined_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(team_id, email)
);

CREATE TABLE shared_configurations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    config_type VARCHAR(50) NOT NULL,
    config_data JSONB NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    is_locked BOOLEAN DEFAULT false,
    locked_by UUID REFERENCES team_members(id),
    locked_at TIMESTAMP,
    created_by UUID NOT NULL REFERENCES team_members(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE config_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    config_id UUID NOT NULL REFERENCES shared_configurations(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    config_data JSONB NOT NULL,
    change_summary TEXT,
    changed_by UUID NOT NULL REFERENCES team_members(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(config_id, version)
);

CREATE TABLE change_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    config_id UUID NOT NULL REFERENCES shared_configurations(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    proposed_changes JSONB NOT NULL,
    base_version INTEGER NOT NULL,
    created_by UUID NOT NULL REFERENCES team_members(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE change_request_reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    change_request_id UUID NOT NULL REFERENCES change_requests(id) ON DELETE CASCADE,
    reviewer_id UUID NOT NULL REFERENCES team_members(id),
    status VARCHAR(50) NOT NULL,
    comments TEXT,
    reviewed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE change_request_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    change_request_id UUID NOT NULL REFERENCES change_requests(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES team_members(id),
    content TEXT NOT NULL,
    parent_id UUID REFERENCES change_request_comments(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE activity_feed (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES team_members(id),
    action_type VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id UUID,
    resource_name VARCHAR(255),
    details JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_member_id UUID NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
    channel VARCHAR(50) NOT NULL,
    event_types JSONB NOT NULL DEFAULT '[]',
    is_enabled BOOLEAN DEFAULT true,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(team_member_id, channel)
);

CREATE TABLE notification_integrations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    integration_type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    config JSONB NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE sent_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    integration_id UUID REFERENCES notification_integrations(id),
    recipient_id UUID REFERENCES team_members(id),
    channel VARCHAR(50) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    subject VARCHAR(500),
    content TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    sent_at TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_teams_tenant ON teams(tenant_id);
CREATE INDEX idx_team_members_team ON team_members(team_id);
CREATE INDEX idx_team_members_user ON team_members(user_id);
CREATE INDEX idx_team_members_email ON team_members(email);
CREATE INDEX idx_shared_configs_team ON shared_configurations(team_id);
CREATE INDEX idx_config_versions_config ON config_versions(config_id);
CREATE INDEX idx_change_requests_team ON change_requests(team_id);
CREATE INDEX idx_change_requests_status ON change_requests(status);
CREATE INDEX idx_change_request_reviews_cr ON change_request_reviews(change_request_id);
CREATE INDEX idx_activity_feed_team ON activity_feed(team_id);
CREATE INDEX idx_activity_feed_created ON activity_feed(created_at DESC);
CREATE INDEX idx_notifications_team ON sent_notifications(team_id);
CREATE INDEX idx_notifications_recipient ON sent_notifications(recipient_id);
