-- Feature 6: Compliance Automation Suite
-- Automated compliance reporting, audit trails, and PII detection

-- Compliance profiles for different frameworks (SOC2, HIPAA, GDPR)
CREATE TABLE IF NOT EXISTS compliance_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    framework VARCHAR(100) NOT NULL, -- soc2, hipaa, gdpr, pci_dss, ccpa
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, framework)
);

-- Data retention policies
CREATE TABLE IF NOT EXISTS data_retention_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    profile_id UUID REFERENCES compliance_profiles(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    data_category VARCHAR(100) NOT NULL, -- events, logs, audit_trails, pii
    retention_days INTEGER NOT NULL,
    archive_enabled BOOLEAN NOT NULL DEFAULT false,
    archive_location VARCHAR(500),
    deletion_method VARCHAR(50) NOT NULL DEFAULT 'soft', -- soft, hard, crypto_shred
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_execution TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- PII detection patterns
CREATE TABLE IF NOT EXISTS pii_detection_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE, -- NULL for system-wide patterns
    name VARCHAR(255) NOT NULL,
    pattern_type VARCHAR(100) NOT NULL, -- regex, keyword, ml_model
    pattern_value TEXT NOT NULL,
    pii_category VARCHAR(100) NOT NULL, -- email, phone, ssn, credit_card, name, address
    sensitivity_level VARCHAR(50) NOT NULL DEFAULT 'medium', -- low, medium, high, critical
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- PII detection results
CREATE TABLE IF NOT EXISTS pii_detections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    pattern_id UUID REFERENCES pii_detection_patterns(id) ON DELETE SET NULL,
    source_type VARCHAR(100) NOT NULL, -- event_payload, endpoint_url, transformation_code
    source_id UUID NOT NULL,
    field_path VARCHAR(500) NOT NULL,
    pii_category VARCHAR(100) NOT NULL,
    sensitivity_level VARCHAR(50) NOT NULL,
    redaction_applied BOOLEAN NOT NULL DEFAULT false,
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Audit trail for all compliance-relevant actions
CREATE TABLE IF NOT EXISTS compliance_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_id UUID, -- user or system
    actor_type VARCHAR(50) NOT NULL, -- user, system, api
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id UUID,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    retention_until TIMESTAMP WITH TIME ZONE
);

-- Compliance reports
CREATE TABLE IF NOT EXISTS compliance_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    profile_id UUID REFERENCES compliance_profiles(id) ON DELETE SET NULL,
    report_type VARCHAR(100) NOT NULL, -- soc2_audit, hipaa_audit, gdpr_dpia, data_inventory
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, generating, completed, failed
    period_start TIMESTAMP WITH TIME ZONE,
    period_end TIMESTAMP WITH TIME ZONE,
    report_data JSONB DEFAULT '{}',
    artifact_url VARCHAR(500),
    generated_by UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Compliance findings from reports
CREATE TABLE IF NOT EXISTS compliance_findings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id UUID NOT NULL REFERENCES compliance_reports(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    severity VARCHAR(50) NOT NULL, -- info, low, medium, high, critical
    category VARCHAR(100) NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    recommendation TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'open', -- open, acknowledged, remediated, accepted
    remediation_deadline TIMESTAMP WITH TIME ZONE,
    remediated_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Data subject requests (GDPR)
CREATE TABLE IF NOT EXISTS data_subject_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    request_type VARCHAR(100) NOT NULL, -- access, rectification, erasure, portability, restriction
    data_subject_id VARCHAR(255) NOT NULL,
    data_subject_email VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, processing, completed, rejected
    request_details JSONB DEFAULT '{}',
    response_data JSONB,
    deadline TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for query optimization
CREATE INDEX IF NOT EXISTS idx_compliance_profiles_tenant ON compliance_profiles(tenant_id);
CREATE INDEX IF NOT EXISTS idx_compliance_profiles_framework ON compliance_profiles(framework);
CREATE INDEX IF NOT EXISTS idx_data_retention_tenant ON data_retention_policies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_pii_patterns_tenant ON pii_detection_patterns(tenant_id);
CREATE INDEX IF NOT EXISTS idx_pii_detections_tenant ON pii_detections(tenant_id);
CREATE INDEX IF NOT EXISTS idx_pii_detections_source ON pii_detections(source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_compliance_audit_tenant_time ON compliance_audit_logs(tenant_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_compliance_audit_action ON compliance_audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_compliance_reports_tenant ON compliance_reports(tenant_id);
CREATE INDEX IF NOT EXISTS idx_compliance_reports_status ON compliance_reports(status);
CREATE INDEX IF NOT EXISTS idx_compliance_findings_report ON compliance_findings(report_id);
CREATE INDEX IF NOT EXISTS idx_compliance_findings_severity ON compliance_findings(severity);
CREATE INDEX IF NOT EXISTS idx_dsr_tenant ON data_subject_requests(tenant_id);
CREATE INDEX IF NOT EXISTS idx_dsr_status ON data_subject_requests(status);

-- Insert default PII detection patterns
INSERT INTO pii_detection_patterns (name, pattern_type, pattern_value, pii_category, sensitivity_level) VALUES
('Email Address', 'regex', '[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}', 'email', 'medium'),
('Phone Number (US)', 'regex', '(\+1[-.\s]?)?(\(?\d{3}\)?[-.\s]?)?\d{3}[-.\s]?\d{4}', 'phone', 'medium'),
('Social Security Number', 'regex', '\d{3}[-\s]?\d{2}[-\s]?\d{4}', 'ssn', 'critical'),
('Credit Card (Visa/MC)', 'regex', '4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}', 'credit_card', 'critical'),
('Credit Card (Amex)', 'regex', '3[47][0-9]{13}', 'credit_card', 'critical'),
('IP Address', 'regex', '\b(?:\d{1,3}\.){3}\d{1,3}\b', 'ip_address', 'low'),
('Date of Birth', 'regex', '\b\d{1,2}[/-]\d{1,2}[/-]\d{2,4}\b', 'dob', 'high')
ON CONFLICT DO NOTHING;
