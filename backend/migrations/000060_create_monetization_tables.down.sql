-- Drop monetization tables

DROP TRIGGER IF EXISTS trigger_monetization_usage_updated ON monetization_usage;
DROP TRIGGER IF EXISTS trigger_monetization_api_keys_updated ON monetization_api_keys;
DROP TRIGGER IF EXISTS trigger_monetization_subscriptions_updated ON monetization_subscriptions;
DROP TRIGGER IF EXISTS trigger_monetization_customers_updated ON monetization_customers;
DROP TRIGGER IF EXISTS trigger_monetization_plans_updated ON monetization_plans;

DROP FUNCTION IF EXISTS get_next_invoice_number(UUID);
DROP TABLE IF EXISTS monetization_invoice_counters;
DROP TABLE IF EXISTS monetization_invoices;
DROP TABLE IF EXISTS monetization_usage;
DROP TABLE IF EXISTS monetization_api_keys;
DROP TABLE IF EXISTS monetization_subscriptions;
DROP TABLE IF EXISTS monetization_customers;
DROP TABLE IF EXISTS monetization_plans;
