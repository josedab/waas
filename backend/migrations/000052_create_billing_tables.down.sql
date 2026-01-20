-- Drop billing tables

DROP TRIGGER IF EXISTS trigger_update_spend_tracker ON billing_usage;
DROP FUNCTION IF EXISTS update_spend_tracker();

DROP TABLE IF EXISTS billing_alert_configs;
DROP TABLE IF EXISTS billing_invoices;
DROP TABLE IF EXISTS billing_optimizations;
DROP TABLE IF EXISTS billing_alerts;
DROP TABLE IF EXISTS billing_budgets;
DROP TABLE IF EXISTS billing_spend_trackers;
DROP TABLE IF EXISTS billing_usage;
