-- White-Label SDK Generator tables
-- SDK configurations, generated artifacts, and distribution tracking

CREATE TABLE sdk_configurations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    package_prefix VARCHAR(100) NOT NULL,
    organization_name VARCHAR(255) NOT NULL,
    branding JSONB NOT NULL DEFAULT '{}',
    languages JSONB NOT NULL DEFAULT '[]',
    api_base_url VARCHAR(500),
    features JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE TABLE sdk_generations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    config_id UUID NOT NULL REFERENCES sdk_configurations(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    language VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    openapi_spec_hash VARCHAR(64),
    artifact_url VARCHAR(500),
    artifact_size_bytes BIGINT,
    package_registry VARCHAR(100),
    package_name VARCHAR(255),
    generation_log TEXT,
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE sdk_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    language VARCHAR(50) NOT NULL,
    template_type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    variables JSONB DEFAULT '[]',
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(language, template_type, name)
);

CREATE TABLE sdk_downloads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    generation_id UUID NOT NULL REFERENCES sdk_generations(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    download_type VARCHAR(50) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE sdk_webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    config_id UUID NOT NULL REFERENCES sdk_configurations(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    webhook_url VARCHAR(500) NOT NULL,
    secret_hash VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_sdk_configs_tenant ON sdk_configurations(tenant_id);
CREATE INDEX idx_sdk_generations_config ON sdk_generations(config_id);
CREATE INDEX idx_sdk_generations_tenant ON sdk_generations(tenant_id);
CREATE INDEX idx_sdk_generations_status ON sdk_generations(status);
CREATE INDEX idx_sdk_templates_language ON sdk_templates(language);
CREATE INDEX idx_sdk_downloads_generation ON sdk_downloads(generation_id);

-- Insert default templates
INSERT INTO sdk_templates (language, template_type, name, content, variables, is_default) VALUES
('go', 'client', 'default', 'package {{.PackageName}}

import (
	"context"
	"net/http"
)

// Client is the {{.OrganizationName}} API client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New creates a new {{.OrganizationName}} client
func New(apiKey string) *Client {
	return &Client{
		baseURL:    "{{.BaseURL}}",
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}
', '["PackageName", "OrganizationName", "BaseURL"]', true),

('typescript', 'client', 'default', 'import axios, { AxiosInstance } from "axios";

export class {{OrganizationName}}Client {
  private client: AxiosInstance;

  constructor(apiKey: string, baseURL = "{{BaseURL}}") {
    this.client = axios.create({
      baseURL,
      headers: { "X-API-Key": apiKey }
    });
  }
}
', '["OrganizationName", "BaseURL"]', true),

('python', 'client', 'default', '"""{{OrganizationName}} SDK"""
import requests

class {{OrganizationName}}Client:
    """{{OrganizationName}} API client"""
    
    def __init__(self, api_key: str, base_url: str = "{{BaseURL}}"):
        self.api_key = api_key
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers["X-API-Key"] = api_key
', '["OrganizationName", "BaseURL"]', true);
