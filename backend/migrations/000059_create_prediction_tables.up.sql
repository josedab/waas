-- Predictive Failure Prevention tables

-- Predictions
CREATE TABLE IF NOT EXISTS failure_predictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID NOT NULL,
    prediction_type VARCHAR(50) NOT NULL, -- endpoint_failure, latency_spike, error_rate_increase, etc.
    probability DECIMAL(5,4) NOT NULL,
    confidence DECIMAL(5,4) NOT NULL,
    predicted_at TIMESTAMP WITH TIME ZONE NOT NULL,
    prediction_window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    prediction_window_end TIMESTAMP WITH TIME ZONE NOT NULL,
    factors JSONB NOT NULL DEFAULT '[]',
    recommended_actions JSONB DEFAULT '[]',
    model_version VARCHAR(50),
    features_snapshot JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, expired, confirmed, false_positive
    outcome VARCHAR(20), -- occurred, did_not_occur
    outcome_recorded_at TIMESTAMP WITH TIME ZONE,
    acknowledged BOOLEAN NOT NULL DEFAULT false,
    acknowledged_by VARCHAR(255),
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_failure_predictions_tenant ON failure_predictions(tenant_id);
CREATE INDEX idx_failure_predictions_endpoint ON failure_predictions(endpoint_id);
CREATE INDEX idx_failure_predictions_status ON failure_predictions(tenant_id, status);
CREATE INDEX idx_failure_predictions_type ON failure_predictions(tenant_id, prediction_type);
CREATE INDEX idx_failure_predictions_window ON failure_predictions(prediction_window_start, prediction_window_end);
CREATE INDEX idx_failure_predictions_created ON failure_predictions(created_at);

-- Prediction alerts
CREATE TABLE IF NOT EXISTS prediction_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    prediction_id UUID NOT NULL REFERENCES failure_predictions(id) ON DELETE CASCADE,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'warning', -- info, warning, critical
    title VARCHAR(500) NOT NULL,
    message TEXT NOT NULL,
    channels JSONB NOT NULL DEFAULT '["email"]',
    sent BOOLEAN NOT NULL DEFAULT false,
    sent_at TIMESTAMP WITH TIME ZONE,
    acknowledged BOOLEAN NOT NULL DEFAULT false,
    acknowledged_by VARCHAR(255),
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_prediction_alerts_prediction ON prediction_alerts(prediction_id);
CREATE INDEX idx_prediction_alerts_tenant ON prediction_alerts(tenant_id);
CREATE INDEX idx_prediction_alerts_sent ON prediction_alerts(tenant_id, sent);
CREATE INDEX idx_prediction_alerts_created ON prediction_alerts(created_at);

-- Alert rules
CREATE TABLE IF NOT EXISTS prediction_alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    prediction_type VARCHAR(50),
    min_probability DECIMAL(5,4) NOT NULL DEFAULT 0.7,
    min_confidence DECIMAL(5,4) NOT NULL DEFAULT 0.6,
    severity VARCHAR(20) NOT NULL DEFAULT 'warning',
    channels JSONB NOT NULL DEFAULT '["email"]',
    recipients JSONB DEFAULT '[]',
    cooldown_minutes INT NOT NULL DEFAULT 60,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_prediction_rule_name UNIQUE(tenant_id, name)
);

CREATE INDEX idx_prediction_alert_rules_tenant ON prediction_alert_rules(tenant_id);
CREATE INDEX idx_prediction_alert_rules_enabled ON prediction_alert_rules(tenant_id, enabled);

-- Endpoint health scores
CREATE TABLE IF NOT EXISTS endpoint_health_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID NOT NULL,
    overall_score DECIMAL(5,2) NOT NULL,
    availability_score DECIMAL(5,2) NOT NULL,
    latency_score DECIMAL(5,2) NOT NULL,
    error_rate_score DECIMAL(5,2) NOT NULL,
    trend VARCHAR(20) NOT NULL DEFAULT 'stable', -- improving, stable, degrading
    components JSONB NOT NULL DEFAULT '{}',
    calculated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_endpoint_health_score UNIQUE(endpoint_id, calculated_at)
);

CREATE INDEX idx_endpoint_health_scores_tenant ON endpoint_health_scores(tenant_id);
CREATE INDEX idx_endpoint_health_scores_endpoint ON endpoint_health_scores(endpoint_id);
CREATE INDEX idx_endpoint_health_scores_calculated ON endpoint_health_scores(calculated_at);

-- ML model metadata
CREATE TABLE IF NOT EXISTS prediction_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id), -- NULL for global models
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    model_type VARCHAR(50) NOT NULL, -- random_forest, gradient_boost, neural_network, etc.
    prediction_type VARCHAR(50) NOT NULL,
    features JSONB NOT NULL DEFAULT '[]',
    hyperparameters JSONB DEFAULT '{}',
    metrics JSONB DEFAULT '{}', -- accuracy, precision, recall, f1
    training_data_range JSONB,
    model_path VARCHAR(500),
    active BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_model_version UNIQUE(tenant_id, name, version)
);

CREATE INDEX idx_prediction_models_tenant ON prediction_models(tenant_id);
CREATE INDEX idx_prediction_models_active ON prediction_models(tenant_id, active);
CREATE INDEX idx_prediction_models_type ON prediction_models(prediction_type);

-- Feature store (time-series features for ML)
CREATE TABLE IF NOT EXISTS prediction_features (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    endpoint_id UUID NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    features JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_prediction_features_endpoint ON prediction_features(endpoint_id, timestamp DESC);
CREATE INDEX idx_prediction_features_tenant ON prediction_features(tenant_id, timestamp DESC);

-- Partition by timestamp for better performance
-- CREATE TABLE prediction_features_partitioned (LIKE prediction_features INCLUDING ALL)
-- PARTITION BY RANGE (timestamp);

-- Trigger for updated_at
CREATE TRIGGER trigger_prediction_alert_rules_updated
    BEFORE UPDATE ON prediction_alert_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_stream_config_timestamp();
