package waf

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

var (
	ErrScanResultNotFound   = errors.New("scan result not found")
	ErrWAFRuleNotFound      = errors.New("WAF rule not found")
	ErrQuarantineNotFound   = errors.New("quarantined webhook not found")
	ErrIPReputationNotFound = errors.New("IP reputation not found")
	ErrAlertNotFound        = errors.New("security alert not found")
	ErrAlreadyReviewed      = errors.New("quarantined webhook already reviewed")
	ErrAlreadyAcknowledged  = errors.New("alert already acknowledged")
)

// Repository defines the interface for WAF storage
type Repository interface {
	// Scan Results
	CreateScanResult(ctx context.Context, result *ScanResult) error
	GetScanResult(ctx context.Context, tenantID, resultID string) (*ScanResult, error)
	ListScanResults(ctx context.Context, tenantID string, limit, offset int) ([]ScanResult, int, error)
	GetScanResultsByWebhook(ctx context.Context, tenantID, webhookID string, limit int) ([]ScanResult, error)

	// WAF Rules
	CreateWAFRule(ctx context.Context, rule *WAFRule) error
	GetWAFRule(ctx context.Context, tenantID, ruleID string) (*WAFRule, error)
	UpdateWAFRule(ctx context.Context, rule *WAFRule) error
	DeleteWAFRule(ctx context.Context, tenantID, ruleID string) error
	ListWAFRules(ctx context.Context, tenantID string) ([]WAFRule, error)
	GetEnabledWAFRules(ctx context.Context, tenantID string) ([]WAFRule, error)
	IncrementRuleHitCount(ctx context.Context, tenantID, ruleID string) error

	// Quarantine
	CreateQuarantine(ctx context.Context, quarantine *QuarantinedWebhook) error
	GetQuarantine(ctx context.Context, tenantID, quarantineID string) (*QuarantinedWebhook, error)
	ListQuarantined(ctx context.Context, tenantID string, limit, offset int) ([]QuarantinedWebhook, int, error)
	UpdateQuarantine(ctx context.Context, quarantine *QuarantinedWebhook) error

	// IP Reputation
	GetIPReputation(ctx context.Context, ip string) (*IPReputation, error)
	UpsertIPReputation(ctx context.Context, reputation *IPReputation) error
	ListBlockedIPs(ctx context.Context, limit, offset int) ([]IPReputation, int, error)

	// Security Alerts
	CreateAlert(ctx context.Context, alert *SecurityAlert) error
	GetAlert(ctx context.Context, tenantID, alertID string) (*SecurityAlert, error)
	ListAlerts(ctx context.Context, tenantID string, limit, offset int) ([]SecurityAlert, int, error)
	UpdateAlert(ctx context.Context, alert *SecurityAlert) error

	// Dashboard Stats
	GetTotalScans(ctx context.Context, tenantID string) (int64, error)
	GetThreatsDetected(ctx context.Context, tenantID string) (int64, error)
	GetThreatsBlocked(ctx context.Context, tenantID string) (int64, error)
	GetQuarantineCount(ctx context.Context, tenantID string) (int64, error)
	GetTopThreats(ctx context.Context, tenantID string, limit int) ([]ThreatSummary, error)
	GetRiskTrend(ctx context.Context, tenantID string, days int) ([]RiskDataPoint, error)

	// Security Thresholds
	GetSecurityThreshold(ctx context.Context, tenantID string) (*SecurityThreshold, error)
	UpsertSecurityThreshold(ctx context.Context, threshold *SecurityThreshold) error
}

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateScanResult(ctx context.Context, result *ScanResult) error {
	query := `INSERT INTO waf_scan_results (id, tenant_id, webhook_id, delivery_id, risk_score, action, scanned_at, duration_ms)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err := r.db.ExecContext(ctx, query, result.ID, result.TenantID, result.WebhookID, result.DeliveryID,
		result.RiskScore, result.Action, result.ScannedAt, result.DurationMs)
	return err
}

func (r *PostgresRepository) GetScanResult(ctx context.Context, tenantID, resultID string) (*ScanResult, error) {
	var s ScanResult
	err := r.db.GetContext(ctx, &s, `SELECT * FROM waf_scan_results WHERE id = $1 AND tenant_id = $2`, resultID, tenantID)
	if err != nil {
		return nil, ErrScanResultNotFound
	}
	return &s, nil
}

func (r *PostgresRepository) ListScanResults(ctx context.Context, tenantID string, limit, offset int) ([]ScanResult, int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM waf_scan_results WHERE tenant_id = $1`, tenantID); err != nil {
		return nil, 0, fmt.Errorf("count scan results: %w", err)
	}
	var results []ScanResult
	err := r.db.SelectContext(ctx, &results,
		`SELECT * FROM waf_scan_results WHERE tenant_id = $1 ORDER BY scanned_at DESC LIMIT $2 OFFSET $3`, tenantID, limit, offset)
	return results, total, err
}

func (r *PostgresRepository) GetScanResultsByWebhook(ctx context.Context, tenantID, webhookID string, limit int) ([]ScanResult, error) {
	var results []ScanResult
	err := r.db.SelectContext(ctx, &results,
		`SELECT * FROM waf_scan_results WHERE tenant_id = $1 AND webhook_id = $2 ORDER BY scanned_at DESC LIMIT $3`, tenantID, webhookID, limit)
	return results, err
}

func (r *PostgresRepository) CreateWAFRule(ctx context.Context, rule *WAFRule) error {
	query := `INSERT INTO waf_rules (id, tenant_id, name, description, pattern, rule_type, action, priority, enabled, hit_count, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
	_, err := r.db.ExecContext(ctx, query, rule.ID, rule.TenantID, rule.Name, rule.Description,
		rule.Pattern, rule.RuleType, rule.Action, rule.Priority, rule.Enabled, rule.HitCount, rule.CreatedAt)
	return err
}

func (r *PostgresRepository) GetWAFRule(ctx context.Context, tenantID, ruleID string) (*WAFRule, error) {
	var rule WAFRule
	err := r.db.GetContext(ctx, &rule, `SELECT * FROM waf_rules WHERE id = $1 AND tenant_id = $2`, ruleID, tenantID)
	if err != nil {
		return nil, ErrWAFRuleNotFound
	}
	return &rule, nil
}

func (r *PostgresRepository) UpdateWAFRule(ctx context.Context, rule *WAFRule) error {
	query := `UPDATE waf_rules SET name=$1, description=$2, pattern=$3, rule_type=$4, action=$5, priority=$6, enabled=$7 WHERE id=$8 AND tenant_id=$9`
	_, err := r.db.ExecContext(ctx, query, rule.Name, rule.Description, rule.Pattern, rule.RuleType,
		rule.Action, rule.Priority, rule.Enabled, rule.ID, rule.TenantID)
	return err
}

func (r *PostgresRepository) DeleteWAFRule(ctx context.Context, tenantID, ruleID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM waf_rules WHERE id = $1 AND tenant_id = $2`, ruleID, tenantID)
	return err
}

func (r *PostgresRepository) ListWAFRules(ctx context.Context, tenantID string) ([]WAFRule, error) {
	var rules []WAFRule
	err := r.db.SelectContext(ctx, &rules, `SELECT * FROM waf_rules WHERE tenant_id = $1 ORDER BY priority ASC`, tenantID)
	return rules, err
}

func (r *PostgresRepository) GetEnabledWAFRules(ctx context.Context, tenantID string) ([]WAFRule, error) {
	var rules []WAFRule
	err := r.db.SelectContext(ctx, &rules,
		`SELECT * FROM waf_rules WHERE tenant_id = $1 AND enabled = true ORDER BY priority ASC`, tenantID)
	return rules, err
}

func (r *PostgresRepository) IncrementRuleHitCount(ctx context.Context, tenantID, ruleID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE waf_rules SET hit_count = hit_count + 1 WHERE id = $1 AND tenant_id = $2`, ruleID, tenantID)
	return err
}

func (r *PostgresRepository) CreateQuarantine(ctx context.Context, q *QuarantinedWebhook) error {
	query := `INSERT INTO waf_quarantine (id, tenant_id, webhook_id, reason, quarantined_at)
		VALUES ($1,$2,$3,$4,$5)`
	_, err := r.db.ExecContext(ctx, query, q.ID, q.TenantID, q.WebhookID, q.Reason, q.QuarantinedAt)
	return err
}

func (r *PostgresRepository) GetQuarantine(ctx context.Context, tenantID, quarantineID string) (*QuarantinedWebhook, error) {
	var q QuarantinedWebhook
	err := r.db.GetContext(ctx, &q, `SELECT * FROM waf_quarantine WHERE id = $1 AND tenant_id = $2`, quarantineID, tenantID)
	if err != nil {
		return nil, ErrQuarantineNotFound
	}
	return &q, nil
}

func (r *PostgresRepository) ListQuarantined(ctx context.Context, tenantID string, limit, offset int) ([]QuarantinedWebhook, int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM waf_quarantine WHERE tenant_id = $1`, tenantID); err != nil {
		return nil, 0, fmt.Errorf("count quarantined: %w", err)
	}
	var quarantined []QuarantinedWebhook
	err := r.db.SelectContext(ctx, &quarantined,
		`SELECT * FROM waf_quarantine WHERE tenant_id = $1 ORDER BY quarantined_at DESC LIMIT $2 OFFSET $3`, tenantID, limit, offset)
	return quarantined, total, err
}

func (r *PostgresRepository) UpdateQuarantine(ctx context.Context, q *QuarantinedWebhook) error {
	query := `UPDATE waf_quarantine SET decision=$1, reviewed_at=$2 WHERE id=$3`
	_, err := r.db.ExecContext(ctx, query, q.Decision, q.ReviewedAt, q.ID)
	return err
}

func (r *PostgresRepository) GetIPReputation(ctx context.Context, ip string) (*IPReputation, error) {
	var rep IPReputation
	err := r.db.GetContext(ctx, &rep, `SELECT * FROM waf_ip_reputation WHERE ip = $1`, ip)
	if err != nil {
		return nil, ErrIPReputationNotFound
	}
	return &rep, nil
}

func (r *PostgresRepository) UpsertIPReputation(ctx context.Context, rep *IPReputation) error {
	query := `INSERT INTO waf_ip_reputation (ip, threat_score, last_seen, report_count, blocked)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (ip) DO UPDATE SET threat_score=$2, last_seen=$3, report_count=waf_ip_reputation.report_count+1, blocked=$5`
	_, err := r.db.ExecContext(ctx, query, rep.IP, rep.ThreatScore, rep.LastSeen, rep.ReportCount, rep.Blocked)
	return err
}

func (r *PostgresRepository) ListBlockedIPs(ctx context.Context, limit, offset int) ([]IPReputation, int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM waf_ip_reputation WHERE blocked = true`); err != nil {
		return nil, 0, fmt.Errorf("count blocked IPs: %w", err)
	}
	var ips []IPReputation
	err := r.db.SelectContext(ctx, &ips,
		`SELECT * FROM waf_ip_reputation WHERE blocked = true ORDER BY threat_score DESC LIMIT $1 OFFSET $2`, limit, offset)
	return ips, total, err
}

func (r *PostgresRepository) CreateAlert(ctx context.Context, alert *SecurityAlert) error {
	query := `INSERT INTO waf_alerts (id, tenant_id, alert_type, severity, title, description, acknowledged, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err := r.db.ExecContext(ctx, query, alert.ID, alert.TenantID, alert.AlertType, alert.Severity,
		alert.Title, alert.Description, false, alert.CreatedAt)
	return err
}

func (r *PostgresRepository) GetAlert(ctx context.Context, tenantID, alertID string) (*SecurityAlert, error) {
	var alert SecurityAlert
	err := r.db.GetContext(ctx, &alert, `SELECT * FROM waf_alerts WHERE id = $1 AND tenant_id = $2`, alertID, tenantID)
	if err != nil {
		return nil, ErrAlertNotFound
	}
	return &alert, nil
}

func (r *PostgresRepository) ListAlerts(ctx context.Context, tenantID string, limit, offset int) ([]SecurityAlert, int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM waf_alerts WHERE tenant_id = $1`, tenantID); err != nil {
		return nil, 0, fmt.Errorf("count alerts: %w", err)
	}
	var alerts []SecurityAlert
	err := r.db.SelectContext(ctx, &alerts,
		`SELECT * FROM waf_alerts WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, tenantID, limit, offset)
	return alerts, total, err
}

func (r *PostgresRepository) UpdateAlert(ctx context.Context, alert *SecurityAlert) error {
	query := `UPDATE waf_alerts SET acknowledged=$1 WHERE id=$2`
	_, err := r.db.ExecContext(ctx, query, alert.Acknowledged, alert.ID)
	return err
}

func (r *PostgresRepository) GetTotalScans(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM waf_scan_results WHERE tenant_id = $1`, tenantID)
	return count, err
}

func (r *PostgresRepository) GetThreatsDetected(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM waf_scan_results WHERE tenant_id = $1 AND risk_score > 0`, tenantID)
	return count, err
}

func (r *PostgresRepository) GetThreatsBlocked(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM waf_scan_results WHERE tenant_id = $1 AND action = 'block'`, tenantID)
	return count, err
}

func (r *PostgresRepository) GetQuarantineCount(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM waf_quarantine WHERE tenant_id = $1 AND decision IS NULL`, tenantID)
	return count, err
}

func (r *PostgresRepository) GetTopThreats(ctx context.Context, tenantID string, limit int) ([]ThreatSummary, error) {
	// Fallback with zero-count entries for common threat types.
	// In production this would aggregate from scan results.
	return []ThreatSummary{
		{Type: ThreatTypeXSS, Count: 0},
		{Type: ThreatTypeSQLInjection, Count: 0},
		{Type: ThreatTypePathTraversal, Count: 0},
	}, nil
}

func (r *PostgresRepository) GetRiskTrend(ctx context.Context, tenantID string, days int) ([]RiskDataPoint, error) {
	// Computed fallback data for the last 7 days.
	// In production this would use time-series aggregation.
	now := time.Now()
	var points []RiskDataPoint
	for i := 6; i >= 0; i-- {
		points = append(points, RiskDataPoint{
			Timestamp: now.AddDate(0, 0, -i),
			Score:     85.0 + float64(i),
		})
	}
	return points, nil
}

func (r *PostgresRepository) GetSecurityThreshold(ctx context.Context, tenantID string) (*SecurityThreshold, error) {
	var t SecurityThreshold
	err := r.db.GetContext(ctx, &t, `SELECT tenant_id, min_score, auto_disable, alert_on_degrade FROM waf_security_thresholds WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("security threshold not found: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) UpsertSecurityThreshold(ctx context.Context, threshold *SecurityThreshold) error {
	query := `INSERT INTO waf_security_thresholds (tenant_id, min_score, auto_disable, alert_on_degrade)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (tenant_id) DO UPDATE SET min_score=$2, auto_disable=$3, alert_on_degrade=$4`
	_, err := r.db.ExecContext(ctx, query, threshold.TenantID, threshold.MinScore, threshold.AutoDisable, threshold.AlertOnDegrade)
	return err
}
