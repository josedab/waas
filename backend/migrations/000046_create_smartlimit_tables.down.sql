DROP INDEX IF EXISTS idx_rate_events_type;
DROP INDEX IF EXISTS idx_rate_events_tenant_endpoint;
DROP TABLE IF EXISTS rate_limit_events;

DROP INDEX IF EXISTS idx_prediction_models_tenant;
DROP TABLE IF EXISTS prediction_models;

DROP INDEX IF EXISTS idx_learning_data_tenant_endpoint;
DROP TABLE IF EXISTS rate_limit_learning_data;

DROP INDEX IF EXISTS idx_rate_states_tenant;
DROP TABLE IF EXISTS rate_limit_states;

DROP INDEX IF EXISTS idx_rate_configs_tenant;
DROP TABLE IF EXISTS adaptive_rate_configs;

DROP INDEX IF EXISTS idx_behaviors_window;
DROP INDEX IF EXISTS idx_behaviors_tenant_endpoint;
DROP TABLE IF EXISTS endpoint_behaviors;
