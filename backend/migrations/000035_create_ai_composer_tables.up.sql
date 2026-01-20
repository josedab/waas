-- AI Webhook Composer tables
-- Stores AI-generated webhook configurations and conversation history

CREATE TABLE ai_composer_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    context JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL DEFAULT (NOW() + INTERVAL '24 hours')
);

CREATE TABLE ai_composer_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES ai_composer_sessions(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE ai_composer_generated_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES ai_composer_sessions(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    config_type VARCHAR(50) NOT NULL,
    generated_config JSONB NOT NULL,
    transformation_code TEXT,
    validation_status VARCHAR(50) DEFAULT 'pending',
    validation_errors JSONB DEFAULT '[]',
    applied BOOLEAN DEFAULT false,
    applied_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE ai_composer_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    category VARCHAR(100) NOT NULL,
    prompt_template TEXT NOT NULL,
    example_input TEXT,
    example_output JSONB,
    is_active BOOLEAN DEFAULT true,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE ai_composer_feedback (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES ai_composer_sessions(id) ON DELETE CASCADE,
    config_id UUID REFERENCES ai_composer_generated_configs(id) ON DELETE SET NULL,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    feedback_text TEXT,
    worked_as_expected BOOLEAN,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_ai_composer_sessions_tenant ON ai_composer_sessions(tenant_id);
CREATE INDEX idx_ai_composer_sessions_status ON ai_composer_sessions(status);
CREATE INDEX idx_ai_composer_messages_session ON ai_composer_messages(session_id);
CREATE INDEX idx_ai_composer_configs_session ON ai_composer_generated_configs(session_id);
CREATE INDEX idx_ai_composer_configs_tenant ON ai_composer_generated_configs(tenant_id);
CREATE INDEX idx_ai_composer_templates_category ON ai_composer_templates(category);
CREATE INDEX idx_ai_composer_feedback_session ON ai_composer_feedback(session_id);

-- Insert default templates
INSERT INTO ai_composer_templates (name, description, category, prompt_template, example_input, example_output) VALUES
('slack_notification', 'Send webhook to Slack channel', 'notifications', 
 'Create a webhook configuration to send a {{event_type}} notification to Slack channel {{channel}}. The message should include: {{fields}}',
 'Send payment completed events to #payments channel with amount and customer name',
 '{"url": "https://hooks.slack.com/services/xxx", "transformation": "return {text: `Payment of ${payload.amount} received from ${payload.customer_name}`}"}'),
 
('email_trigger', 'Trigger email via webhook', 'notifications',
 'Create a webhook to trigger an email when {{condition}}. Email should contain: {{content}}',
 'Send email when order status changes to shipped',
 '{"url": "https://api.sendgrid.com/v3/mail/send", "headers": {"Authorization": "Bearer {{api_key}}"}}'),

('data_sync', 'Sync data between systems', 'integration',
 'Create a webhook to sync {{entity}} data from {{source}} to {{destination}} when {{trigger}}',
 'Sync customer data to Salesforce when customer is created',
 '{"url": "https://yourinstance.salesforce.com/services/data/v54.0/sobjects/Contact", "method": "POST"}'),

('conditional_routing', 'Route webhooks based on conditions', 'routing',
 'Create a webhook that routes to different endpoints based on {{condition}}. Routes: {{routes}}',
 'Route high-value orders (>$1000) to priority queue, others to standard',
 '{"transformation": "return payload.amount > 1000 ? {queue: \"priority\"} : {queue: \"standard\"}"}'),

('payload_filter', 'Filter and transform payload', 'transformation',
 'Transform the incoming webhook payload to include only {{fields}} and rename {{renames}}',
 'Keep only id, email, and created_at. Rename created_at to signup_date',
 '{"transformation": "return {id: payload.id, email: payload.email, signup_date: payload.created_at}"}');
