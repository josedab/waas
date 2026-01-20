-- Rollback Cloud Platform Tables

DROP INDEX IF EXISTS idx_tenant_regions_tenant;
DROP INDEX IF EXISTS idx_org_api_tokens_prefix;
DROP INDEX IF EXISTS idx_team_members_email;
DROP INDEX IF EXISTS idx_onboarding_sessions_email;
DROP INDEX IF EXISTS idx_usage_events_recorded_at;
DROP INDEX IF EXISTS idx_usage_events_org_type;
DROP INDEX IF EXISTS idx_usage_meters_org_period;
DROP INDEX IF EXISTS idx_organizations_status;
DROP INDEX IF EXISTS idx_organizations_slug;

DROP TABLE IF EXISTS tenant_regions;
DROP TABLE IF EXISTS cloud_regions;
DROP TABLE IF EXISTS org_api_tokens;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS onboarding_sessions;
DROP TABLE IF EXISTS usage_events;
DROP TABLE IF EXISTS usage_meters;
DROP TABLE IF EXISTS subscription_plans;
DROP TABLE IF EXISTS organizations;
