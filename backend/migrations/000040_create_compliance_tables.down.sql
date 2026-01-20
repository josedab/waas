-- Rollback: Compliance Automation Suite tables

DROP INDEX IF EXISTS idx_dsr_status;
DROP INDEX IF EXISTS idx_dsr_tenant;
DROP INDEX IF EXISTS idx_compliance_findings_severity;
DROP INDEX IF EXISTS idx_compliance_findings_report;
DROP INDEX IF EXISTS idx_compliance_reports_status;
DROP INDEX IF EXISTS idx_compliance_reports_tenant;
DROP INDEX IF EXISTS idx_compliance_audit_action;
DROP INDEX IF EXISTS idx_compliance_audit_tenant_time;
DROP INDEX IF EXISTS idx_pii_detections_source;
DROP INDEX IF EXISTS idx_pii_detections_tenant;
DROP INDEX IF EXISTS idx_pii_patterns_tenant;
DROP INDEX IF EXISTS idx_data_retention_tenant;
DROP INDEX IF EXISTS idx_compliance_profiles_framework;
DROP INDEX IF EXISTS idx_compliance_profiles_tenant;

DROP TABLE IF EXISTS data_subject_requests;
DROP TABLE IF EXISTS compliance_findings;
DROP TABLE IF EXISTS compliance_reports;
DROP TABLE IF EXISTS compliance_audit_logs;
DROP TABLE IF EXISTS pii_detections;
DROP TABLE IF EXISTS pii_detection_patterns;
DROP TABLE IF EXISTS data_retention_policies;
DROP TABLE IF EXISTS compliance_profiles;
