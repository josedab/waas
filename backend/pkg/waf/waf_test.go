package waf

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// mockRepository implements the Repository interface for testing
type mockRepository struct {
	scanResults   map[string]*ScanResult
	wafRules      map[string]*WAFRule
	enabledRules  []WAFRule
	quarantined   map[string]*QuarantinedWebhook
	ipReputations map[string]*IPReputation
	alerts        map[string]*SecurityAlert
	thresholds    map[string]*SecurityThreshold
	hitCounts     map[string]int
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		scanResults:   make(map[string]*ScanResult),
		wafRules:      make(map[string]*WAFRule),
		enabledRules:  []WAFRule{},
		quarantined:   make(map[string]*QuarantinedWebhook),
		ipReputations: make(map[string]*IPReputation),
		alerts:        make(map[string]*SecurityAlert),
		thresholds:    make(map[string]*SecurityThreshold),
		hitCounts:     make(map[string]int),
	}
}

func (m *mockRepository) CreateScanResult(_ context.Context, result *ScanResult) error {
	m.scanResults[result.ID] = result
	return nil
}

func (m *mockRepository) GetScanResult(_ context.Context, tenantID, resultID string) (*ScanResult, error) {
	if r, ok := m.scanResults[resultID]; ok && r.TenantID == tenantID {
		return r, nil
	}
	return nil, ErrScanResultNotFound
}

func (m *mockRepository) ListScanResults(_ context.Context, tenantID string, limit, offset int) ([]ScanResult, int, error) {
	var results []ScanResult
	for _, r := range m.scanResults {
		if r.TenantID == tenantID {
			results = append(results, *r)
		}
	}
	return results, len(results), nil
}

func (m *mockRepository) GetScanResultsByWebhook(_ context.Context, tenantID, webhookID string, limit int) ([]ScanResult, error) {
	var results []ScanResult
	for _, r := range m.scanResults {
		if r.TenantID == tenantID && r.WebhookID == webhookID {
			results = append(results, *r)
		}
	}
	return results, nil
}

func (m *mockRepository) CreateWAFRule(_ context.Context, rule *WAFRule) error {
	m.wafRules[rule.ID] = rule
	return nil
}

func (m *mockRepository) GetWAFRule(_ context.Context, tenantID, ruleID string) (*WAFRule, error) {
	if r, ok := m.wafRules[ruleID]; ok && r.TenantID == tenantID {
		return r, nil
	}
	return nil, ErrWAFRuleNotFound
}

func (m *mockRepository) UpdateWAFRule(_ context.Context, rule *WAFRule) error {
	m.wafRules[rule.ID] = rule
	return nil
}

func (m *mockRepository) DeleteWAFRule(_ context.Context, tenantID, ruleID string) error {
	delete(m.wafRules, ruleID)
	return nil
}

func (m *mockRepository) ListWAFRules(_ context.Context, tenantID string) ([]WAFRule, error) {
	var rules []WAFRule
	for _, r := range m.wafRules {
		if r.TenantID == tenantID {
			rules = append(rules, *r)
		}
	}
	return rules, nil
}

func (m *mockRepository) GetEnabledWAFRules(_ context.Context, tenantID string) ([]WAFRule, error) {
	var rules []WAFRule
	for _, r := range m.enabledRules {
		if r.TenantID == tenantID {
			rules = append(rules, r)
		}
	}
	return rules, nil
}

func (m *mockRepository) IncrementRuleHitCount(_ context.Context, tenantID, ruleID string) error {
	m.hitCounts[ruleID]++
	return nil
}

func (m *mockRepository) CreateQuarantine(_ context.Context, q *QuarantinedWebhook) error {
	m.quarantined[q.ID] = q
	return nil
}

func (m *mockRepository) GetQuarantine(_ context.Context, tenantID, quarantineID string) (*QuarantinedWebhook, error) {
	if q, ok := m.quarantined[quarantineID]; ok && q.TenantID == tenantID {
		return q, nil
	}
	return nil, ErrQuarantineNotFound
}

func (m *mockRepository) ListQuarantined(_ context.Context, tenantID string, limit, offset int) ([]QuarantinedWebhook, int, error) {
	var items []QuarantinedWebhook
	for _, q := range m.quarantined {
		if q.TenantID == tenantID {
			items = append(items, *q)
		}
	}
	return items, len(items), nil
}

func (m *mockRepository) UpdateQuarantine(_ context.Context, q *QuarantinedWebhook) error {
	m.quarantined[q.ID] = q
	return nil
}

func (m *mockRepository) GetIPReputation(_ context.Context, ip string) (*IPReputation, error) {
	if r, ok := m.ipReputations[ip]; ok {
		return r, nil
	}
	return nil, ErrIPReputationNotFound
}

func (m *mockRepository) UpsertIPReputation(_ context.Context, rep *IPReputation) error {
	m.ipReputations[rep.IP] = rep
	return nil
}

func (m *mockRepository) ListBlockedIPs(_ context.Context, limit, offset int) ([]IPReputation, int, error) {
	var ips []IPReputation
	for _, r := range m.ipReputations {
		if r.Blocked {
			ips = append(ips, *r)
		}
	}
	return ips, len(ips), nil
}

func (m *mockRepository) CreateAlert(_ context.Context, alert *SecurityAlert) error {
	m.alerts[alert.ID] = alert
	return nil
}

func (m *mockRepository) GetAlert(_ context.Context, tenantID, alertID string) (*SecurityAlert, error) {
	if a, ok := m.alerts[alertID]; ok && a.TenantID == tenantID {
		return a, nil
	}
	return nil, ErrAlertNotFound
}

func (m *mockRepository) ListAlerts(_ context.Context, tenantID string, limit, offset int) ([]SecurityAlert, int, error) {
	var alerts []SecurityAlert
	for _, a := range m.alerts {
		if a.TenantID == tenantID {
			alerts = append(alerts, *a)
		}
	}
	return alerts, len(alerts), nil
}

func (m *mockRepository) UpdateAlert(_ context.Context, alert *SecurityAlert) error {
	m.alerts[alert.ID] = alert
	return nil
}

func (m *mockRepository) GetTotalScans(_ context.Context, tenantID string) (int64, error) {
	return int64(len(m.scanResults)), nil
}

func (m *mockRepository) GetThreatsDetected(_ context.Context, tenantID string) (int64, error) {
	return 0, nil
}

func (m *mockRepository) GetThreatsBlocked(_ context.Context, tenantID string) (int64, error) {
	return 0, nil
}

func (m *mockRepository) GetQuarantineCount(_ context.Context, tenantID string) (int64, error) {
	return 0, nil
}

func (m *mockRepository) GetTopThreats(_ context.Context, tenantID string, limit int) ([]ThreatSummary, error) {
	return []ThreatSummary{}, nil
}

func (m *mockRepository) GetRiskTrend(_ context.Context, tenantID string, days int) ([]RiskDataPoint, error) {
	return []RiskDataPoint{}, nil
}

func (m *mockRepository) GetSecurityThreshold(_ context.Context, tenantID string) (*SecurityThreshold, error) {
	if t, ok := m.thresholds[tenantID]; ok {
		return t, nil
	}
	return nil, ErrScanResultNotFound
}

func (m *mockRepository) UpsertSecurityThreshold(_ context.Context, threshold *SecurityThreshold) error {
	m.thresholds[threshold.TenantID] = threshold
	return nil
}

// Tests

func TestScanPayload_XSS(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	req := &ScanPayloadRequest{
		WebhookID: "wh-1",
		Payload:   json.RawMessage(`{"data": "<script>alert('xss')</script>"}`),
	}

	result, err := svc.ScanPayload(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, threat := range result.Threats {
		if threat.Type == ThreatTypeXSS {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected XSS threat to be detected")
	}
	if result.RiskScore <= 0 {
		t.Error("expected non-zero risk score for XSS payload")
	}
}

func TestScanPayload_SQLInjection(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	req := &ScanPayloadRequest{
		WebhookID: "wh-1",
		Payload:   json.RawMessage(`{"query": "SELECT * FROM users WHERE id = 1 OR 1=1"}`),
	}

	result, err := svc.ScanPayload(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, threat := range result.Threats {
		if threat.Type == ThreatTypeSQLInjection {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected SQL injection threat to be detected")
	}
}

func TestScanPayload_PathTraversal(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	req := &ScanPayloadRequest{
		WebhookID: "wh-1",
		Payload:   json.RawMessage(`{"file": "../../etc/passwd"}`),
	}

	result, err := svc.ScanPayload(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, threat := range result.Threats {
		if threat.Type == ThreatTypePathTraversal {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected path traversal threat to be detected")
	}
}

func TestScanPayload_CleanPayload(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	req := &ScanPayloadRequest{
		WebhookID: "wh-1",
		Payload:   json.RawMessage(`{"event": "push", "ref": "refs/heads/main"}`),
	}

	result, err := svc.ScanPayload(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Threats) != 0 {
		t.Errorf("expected no threats for clean payload, got %d", len(result.Threats))
	}
	if result.RiskScore != 0 {
		t.Errorf("expected risk score 0 for clean payload, got %f", result.RiskScore)
	}
	if result.Action != ScanActionAllow {
		t.Errorf("expected action 'allow', got '%s'", result.Action)
	}
}

func TestScanPayload_DeeplyNestedJSON(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	// Build a deeply nested JSON object (depth > 50)
	nested := `"leaf"`
	for i := 0; i < 55; i++ {
		nested = `{"a":` + nested + `}`
	}

	req := &ScanPayloadRequest{
		WebhookID: "wh-1",
		Payload:   json.RawMessage(nested),
	}

	result, err := svc.ScanPayload(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, threat := range result.Threats {
		if threat.Type == ThreatTypeMaliciousJSON {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected malicious JSON threat for deeply nested payload")
	}
}

func TestEvaluateWAFRules_Matching(t *testing.T) {
	repo := newMockRepository()
	repo.enabledRules = []WAFRule{
		{
			ID:       "rule-1",
			TenantID: "tenant-1",
			Name:     "Block secret keyword",
			Pattern:  "secret_token",
			RuleType: WAFRuleTypeKeyword,
			Action:   ScanActionBlock,
			Enabled:  true,
		},
		{
			ID:       "rule-2",
			TenantID: "tenant-1",
			Name:     "Block regex pattern",
			Pattern:  `password\s*=\s*\S+`,
			RuleType: WAFRuleTypeRegex,
			Action:   ScanActionBlock,
			Enabled:  true,
		},
	}
	svc := NewService(repo)
	ctx := context.Background()

	// Test keyword match
	threats, err := svc.EvaluateWAFRules(ctx, "tenant-1", `{"key": "my secret_token here"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(threats) == 0 {
		t.Error("expected at least one threat from keyword WAF rule match")
	}

	// Test regex match
	threats, err = svc.EvaluateWAFRules(ctx, "tenant-1", `password = hunter2`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(threats) == 0 {
		t.Error("expected at least one threat from regex WAF rule match")
	}

	// Test no match
	threats, err = svc.EvaluateWAFRules(ctx, "tenant-1", `{"event": "push"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(threats) != 0 {
		t.Errorf("expected no threats for non-matching payload, got %d", len(threats))
	}
}

func TestCreateWAFRule_And_GetWAFRule(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	req := &CreateWAFRuleRequest{
		Name:     "Test Rule",
		Pattern:  "malicious",
		RuleType: WAFRuleTypeKeyword,
		Action:   ScanActionBlock,
		Priority: 1,
		Enabled:  true,
	}

	rule, err := svc.CreateWAFRule(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("unexpected error creating rule: %v", err)
	}
	if rule.Name != "Test Rule" {
		t.Errorf("expected rule name 'Test Rule', got '%s'", rule.Name)
	}
	if rule.TenantID != "tenant-1" {
		t.Errorf("expected tenant ID 'tenant-1', got '%s'", rule.TenantID)
	}
	if rule.ID == "" {
		t.Error("expected non-empty rule ID")
	}

	// Retrieve the rule
	retrieved, err := svc.GetWAFRule(ctx, "tenant-1", rule.ID)
	if err != nil {
		t.Fatalf("unexpected error getting rule: %v", err)
	}
	if retrieved.Name != rule.Name {
		t.Errorf("expected name '%s', got '%s'", rule.Name, retrieved.Name)
	}
	if retrieved.Pattern != rule.Pattern {
		t.Errorf("expected pattern '%s', got '%s'", rule.Pattern, retrieved.Pattern)
	}
}

func TestScanEndpointSecurity_NonHTTPS(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	result, err := svc.ScanEndpointSecurity(ctx, "tenant-1", "http://example.invalid:9999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TLSInfo == nil {
		t.Fatal("expected TLSInfo to be non-nil")
	}
	if result.TLSInfo.IsHTTPS {
		t.Error("expected IsHTTPS to be false for HTTP URL")
	}

	// Should have "tls-missing" finding
	found := false
	for _, f := range result.Findings {
		if f.ID == "tls-missing" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'tls-missing' finding for non-HTTPS URL")
	}

	// Score should be degraded (< 100) due to missing TLS and unreachable headers
	if result.NumericScore >= 100 {
		t.Errorf("expected degraded score for non-HTTPS URL, got %d", result.NumericScore)
	}
}

func TestCheckSecurityHeaders_NonReachable(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	checks := svc.checkSecurityHeaders("http://unreachable.invalid:9999")

	if len(checks) == 0 {
		t.Fatal("expected security header checks even for non-reachable endpoint")
	}

	expectedHeaders := []string{
		"Strict-Transport-Security",
		"Content-Security-Policy",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"X-XSS-Protection",
		"Referrer-Policy",
		"Permissions-Policy",
	}

	for _, expected := range expectedHeaders {
		found := false
		for _, check := range checks {
			if check.Header == expected {
				found = true
				if check.Present {
					t.Errorf("expected header %s to be not present for unreachable endpoint", expected)
				}
				break
			}
		}
		if !found {
			t.Errorf("expected header check for %s", expected)
		}
	}
}

func TestExportSecurityReport_JSON(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	result := &SecurityScanResult{
		EndpointID:   "ep-1",
		URL:          "https://example.com",
		OverallScore: "A",
		NumericScore: 95,
		ScannedAt:    time.Now(),
		Findings: []SecurityFinding{
			{
				ID:             "finding-1",
				Title:          "Test Finding",
				Severity:       "high",
				Category:       "test",
				Description:    "A test finding",
				Recommendation: "Fix it",
			},
		},
	}

	data, err := svc.ExportSecurityReport(ctx, result, "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's valid JSON
	var parsed SecurityScanResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("exported JSON is not valid: %v", err)
	}
	if parsed.NumericScore != 95 {
		t.Errorf("expected numeric score 95, got %d", parsed.NumericScore)
	}
	if len(parsed.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(parsed.Findings))
	}
}

func TestExportSecurityReport_CSV(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	result := &SecurityScanResult{
		Findings: []SecurityFinding{
			{
				ID:             "f-1",
				Title:          "Missing Header",
				Severity:       "high",
				Category:       "headers",
				Description:    "X-Frame-Options missing",
				Recommendation: "Add header",
			},
			{
				ID:             "f-2",
				Title:          "TLS Issue",
				Severity:       "critical",
				Category:       "transport",
				Description:    "No HTTPS",
				Recommendation: "Enable TLS",
			},
		},
	}

	data, err := svc.ExportSecurityReport(ctx, result, "csv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	csvStr := string(data)
	if !strings.Contains(csvStr, "ID,Title,Severity,Category,Description,Recommendation") {
		t.Error("expected CSV header row")
	}
	if !strings.Contains(csvStr, "f-1") {
		t.Error("expected finding f-1 in CSV")
	}
	if !strings.Contains(csvStr, "f-2") {
		t.Error("expected finding f-2 in CSV")
	}
}

func TestExportSecurityReport_UnsupportedFormat(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	result := &SecurityScanResult{}
	_, err := svc.ExportSecurityReport(ctx, result, "xml")
	if err == nil {
		t.Error("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("expected 'unsupported format' error, got: %v", err)
	}
}

func TestDetectSecurityDrift_ScoreDegradation(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	previous := &SecurityScanResult{NumericScore: 90, Findings: []SecurityFinding{}}
	current := &SecurityScanResult{NumericScore: 70, Findings: []SecurityFinding{}}

	drifts := svc.DetectSecurityDrift(current, previous)

	found := false
	for _, d := range drifts {
		if d.ID == "drift-score" {
			found = true
			if d.Severity != "high" {
				t.Errorf("expected severity 'high', got '%s'", d.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected drift-score finding for score degradation")
	}
}

func TestDetectSecurityDrift_NewFindings(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	previous := &SecurityScanResult{
		NumericScore: 80,
		Findings: []SecurityFinding{
			{ID: "existing-1", Title: "Old Finding"},
		},
	}
	current := &SecurityScanResult{
		NumericScore: 80,
		Findings: []SecurityFinding{
			{ID: "existing-1", Title: "Old Finding"},
			{ID: "new-1", Title: "New Finding", Category: "headers"},
		},
	}

	drifts := svc.DetectSecurityDrift(current, previous)

	found := false
	for _, d := range drifts {
		if d.ID == "new-1" {
			found = true
			if d.Category != "drift" {
				t.Errorf("expected category 'drift', got '%s'", d.Category)
			}
			break
		}
	}
	if !found {
		t.Error("expected new finding to appear as drift")
	}
}

func TestDetectSecurityDrift_TLSDowngrade(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	previous := &SecurityScanResult{
		NumericScore: 90,
		TLSInfo:      &TLSInfo{IsHTTPS: true},
	}
	current := &SecurityScanResult{
		NumericScore: 90,
		TLSInfo:      &TLSInfo{IsHTTPS: false},
	}

	drifts := svc.DetectSecurityDrift(current, previous)

	found := false
	for _, d := range drifts {
		if d.ID == "drift-tls-removed" {
			found = true
			if d.Severity != "critical" {
				t.Errorf("expected severity 'critical', got '%s'", d.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected TLS downgrade drift finding")
	}
}

func TestDetectSecurityDrift_NoDrift(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	scan := &SecurityScanResult{
		NumericScore: 90,
		TLSInfo:      &TLSInfo{IsHTTPS: true},
		Findings: []SecurityFinding{
			{ID: "f-1"},
		},
	}

	drifts := svc.DetectSecurityDrift(scan, scan)
	if len(drifts) != 0 {
		t.Errorf("expected no drifts when comparing identical scans, got %d", len(drifts))
	}
}

func TestGetSecurityThreshold_Default(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	threshold, err := svc.GetSecurityThreshold(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if threshold.MinScore != 60 {
		t.Errorf("expected default min score 60, got %d", threshold.MinScore)
	}
	if threshold.AutoDisable {
		t.Error("expected default auto_disable to be false")
	}
	if !threshold.AlertOnDegrade {
		t.Error("expected default alert_on_degrade to be true")
	}
}

func TestUpdateSecurityThreshold(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	threshold := &SecurityThreshold{
		TenantID:       "tenant-1",
		MinScore:       75,
		AutoDisable:    true,
		AlertOnDegrade: false,
	}

	if err := svc.UpdateSecurityThreshold(ctx, threshold); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, err := svc.GetSecurityThreshold(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.MinScore != 75 {
		t.Errorf("expected min score 75, got %d", retrieved.MinScore)
	}
	if !retrieved.AutoDisable {
		t.Error("expected auto_disable to be true")
	}
}
