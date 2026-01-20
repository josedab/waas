-- Drop prediction tables

DROP TRIGGER IF EXISTS trigger_prediction_alert_rules_updated ON prediction_alert_rules;

DROP TABLE IF EXISTS prediction_features;
DROP TABLE IF EXISTS prediction_models;
DROP TABLE IF EXISTS endpoint_health_scores;
DROP TABLE IF EXISTS prediction_alert_rules;
DROP TABLE IF EXISTS prediction_alerts;
DROP TABLE IF EXISTS failure_predictions;
