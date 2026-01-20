-- Rollback Feature 10: Serverless Edge Functions

DROP TABLE IF EXISTS edge_function_tests;
DROP TABLE IF EXISTS edge_function_secrets;
DROP TABLE IF EXISTS edge_function_metrics;
DROP TABLE IF EXISTS edge_function_invocations;
DROP TABLE IF EXISTS edge_function_triggers;
DROP TABLE IF EXISTS edge_function_deployments;
DROP TABLE IF EXISTS edge_function_versions;
DROP TABLE IF EXISTS edge_functions;
DROP TABLE IF EXISTS edge_locations;
