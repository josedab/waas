-- Rollback: Self-Healing Endpoint Intelligence tables

DROP INDEX IF EXISTS idx_circuit_breakers_state;
DROP INDEX IF EXISTS idx_optimization_suggestions_status;
DROP INDEX IF EXISTS idx_optimization_suggestions_endpoint;
DROP INDEX IF EXISTS idx_remediation_actions_time;
DROP INDEX IF EXISTS idx_remediation_actions_endpoint;
DROP INDEX IF EXISTS idx_remediation_rules_tenant;
DROP INDEX IF EXISTS idx_behavior_patterns_endpoint;
DROP INDEX IF EXISTS idx_health_predictions_time;
DROP INDEX IF EXISTS idx_health_predictions_endpoint;

DROP TABLE IF EXISTS endpoint_circuit_breakers;
DROP TABLE IF EXISTS endpoint_optimization_suggestions;
DROP TABLE IF EXISTS remediation_actions;
DROP TABLE IF EXISTS auto_remediation_rules;
DROP TABLE IF EXISTS endpoint_behavior_patterns;
DROP TABLE IF EXISTS endpoint_health_predictions;
