-- Drop remediation tables

DROP TRIGGER IF EXISTS trigger_remediation_rules_updated ON remediation_rules;
DROP TRIGGER IF EXISTS trigger_remediation_actions_updated ON remediation_actions;

DROP TABLE IF EXISTS remediation_metrics;
DROP TABLE IF EXISTS remediation_history;
DROP TABLE IF EXISTS remediation_approvals;
DROP TABLE IF EXISTS remediation_rules;
DROP TABLE IF EXISTS remediation_actions;
