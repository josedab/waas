-- AI debugging analyses table
CREATE TABLE IF NOT EXISTS ai_debug_analyses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id TEXT NOT NULL UNIQUE,
    classification JSONB NOT NULL,
    root_cause TEXT,
    explanation TEXT,
    suggestions JSONB DEFAULT '[]',
    transform_fix JSONB,
    similar_issues JSONB DEFAULT '[]',
    confidence_score DECIMAL(3,2) DEFAULT 0,
    processing_time_ms INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_ai_analyses_delivery ON ai_debug_analyses(delivery_id);
CREATE INDEX idx_ai_analyses_created ON ai_debug_analyses(created_at DESC);

-- AI error patterns table for learning
CREATE TABLE IF NOT EXISTS ai_error_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    pattern TEXT NOT NULL,
    category VARCHAR(50) NOT NULL,
    frequency INTEGER DEFAULT 1,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolution TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, pattern)
);

CREATE INDEX idx_ai_patterns_tenant ON ai_error_patterns(tenant_id);
CREATE INDEX idx_ai_patterns_category ON ai_error_patterns(category);
CREATE INDEX idx_ai_patterns_frequency ON ai_error_patterns(frequency DESC);
