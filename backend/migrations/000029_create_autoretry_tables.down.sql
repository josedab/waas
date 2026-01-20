-- Rollback Auto-Retry Tables

DROP INDEX IF EXISTS idx_experiment_assignments_exp;
DROP INDEX IF EXISTS idx_model_metrics_version;
DROP INDEX IF EXISTS idx_retry_predictions_endpoint;
DROP INDEX IF EXISTS idx_retry_predictions_delivery;
DROP INDEX IF EXISTS idx_delivery_features_created;
DROP INDEX IF EXISTS idx_delivery_features_endpoint;

DROP TABLE IF EXISTS experiment_assignments;
DROP TABLE IF EXISTS retry_experiments;
DROP TABLE IF EXISTS model_metrics;
DROP TABLE IF EXISTS retry_predictions;
DROP TABLE IF EXISTS delivery_features;
