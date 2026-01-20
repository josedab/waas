-- Rollback Contract Testing Tables

DROP INDEX IF EXISTS idx_contract_test_results_suite;
DROP INDEX IF EXISTS idx_contract_subscriptions_consumer;
DROP INDEX IF EXISTS idx_contract_subscriptions_contract;
DROP INDEX IF EXISTS idx_breaking_changes_contract;
DROP INDEX IF EXISTS idx_contract_validations_contract;
DROP INDEX IF EXISTS idx_contract_versions_contract;
DROP INDEX IF EXISTS idx_webhook_contracts_status;
DROP INDEX IF EXISTS idx_webhook_contracts_event_type;
DROP INDEX IF EXISTS idx_webhook_contracts_tenant;

DROP TABLE IF EXISTS contract_test_results;
DROP TABLE IF EXISTS contract_test_suites;
DROP TABLE IF EXISTS contract_subscriptions;
DROP TABLE IF EXISTS breaking_change_detections;
DROP TABLE IF EXISTS contract_validations;
DROP TABLE IF EXISTS contract_versions;
DROP TABLE IF EXISTS webhook_contracts;
