-- Intelligent Auto-Retry Tables
-- Feature 5: ML-based retry optimization

-- Delivery Features (for ML model training)
CREATE TABLE IF NOT EXISTS delivery_features (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id UUID NOT NULL,
    endpoint_id UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    
    -- Endpoint features
    endpoint_success_rate_1h FLOAT,
    endpoint_success_rate_24h FLOAT,
    endpoint_avg_response_time_ms INTEGER,
    endpoint_error_rate_1h FLOAT,
    endpoint_last_success_minutes INTEGER,
    
    -- Time features
    hour_of_day INTEGER,
    day_of_week INTEGER,
    is_weekend BOOLEAN,
    is_business_hours BOOLEAN,
    
    -- Payload features
    payload_size_bytes INTEGER,
    has_large_payload BOOLEAN,
    
    -- Retry features
    attempt_number INTEGER,
    time_since_first_attempt_seconds INTEGER,
    previous_error_code VARCHAR(50),
    consecutive_failures INTEGER,
    
    -- Outcome (for training)
    was_successful BOOLEAN,
    response_time_ms INTEGER,
    http_status_code INTEGER,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Retry Predictions
CREATE TABLE IF NOT EXISTS retry_predictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id UUID NOT NULL,
    endpoint_id UUID NOT NULL,
    
    predicted_success_probability FLOAT NOT NULL,
    recommended_delay_seconds INTEGER NOT NULL,
    confidence_score FLOAT,
    
    model_version VARCHAR(50),
    feature_vector JSONB,
    
    -- Actual outcome (for model evaluation)
    actual_success BOOLEAN,
    actual_delay_used_seconds INTEGER,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    evaluated_at TIMESTAMP WITH TIME ZONE
);

-- Model Performance Metrics
CREATE TABLE IF NOT EXISTS model_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_version VARCHAR(50) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    metric_value FLOAT NOT NULL,
    sample_size INTEGER,
    period_start TIMESTAMP WITH TIME ZONE,
    period_end TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- A/B Test Experiments
CREATE TABLE IF NOT EXISTS retry_experiments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'draft' CHECK (status IN ('draft', 'running', 'paused', 'completed')),
    
    control_strategy JSONB NOT NULL,
    treatment_strategy JSONB NOT NULL,
    traffic_split FLOAT DEFAULT 0.5,
    
    start_date TIMESTAMP WITH TIME ZONE,
    end_date TIMESTAMP WITH TIME ZONE,
    
    control_sample_size INTEGER DEFAULT 0,
    treatment_sample_size INTEGER DEFAULT 0,
    control_success_rate FLOAT,
    treatment_success_rate FLOAT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Experiment Assignments
CREATE TABLE IF NOT EXISTS experiment_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id UUID NOT NULL REFERENCES retry_experiments(id) ON DELETE CASCADE,
    delivery_id UUID NOT NULL,
    variant VARCHAR(20) NOT NULL CHECK (variant IN ('control', 'treatment')),
    was_successful BOOLEAN,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_delivery_features_endpoint ON delivery_features(endpoint_id);
CREATE INDEX idx_delivery_features_created ON delivery_features(created_at);
CREATE INDEX idx_retry_predictions_delivery ON retry_predictions(delivery_id);
CREATE INDEX idx_retry_predictions_endpoint ON retry_predictions(endpoint_id);
CREATE INDEX idx_model_metrics_version ON model_metrics(model_version, metric_name);
CREATE INDEX idx_experiment_assignments_exp ON experiment_assignments(experiment_id);
