-- Rollback Playground Tables

DROP INDEX IF EXISTS idx_playground_snippets_type;
DROP INDEX IF EXISTS idx_playground_snippets_tenant;
DROP INDEX IF EXISTS idx_transformation_executions_session;
DROP INDEX IF EXISTS idx_request_captures_created;
DROP INDEX IF EXISTS idx_request_captures_endpoint;
DROP INDEX IF EXISTS idx_request_captures_session;
DROP INDEX IF EXISTS idx_request_captures_tenant;
DROP INDEX IF EXISTS idx_playground_sessions_expires;
DROP INDEX IF EXISTS idx_playground_sessions_tenant;

DROP TABLE IF EXISTS playground_snippets;
DROP TABLE IF EXISTS debug_breakpoints;
DROP TABLE IF EXISTS transformation_executions;
DROP TABLE IF EXISTS request_captures;
DROP TABLE IF EXISTS playground_sessions;
