package securityintel

import (
	"context"
	"testing"
	"time"
)

func TestNewService(t *testing.T) {
	svc := NewService(nil, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestInspectPayload_Clean(t *testing.T) {
	svc := NewService(nil, nil)
	result, err := svc.InspectPayload(context.Background(), "tenant-1", &InspectPayloadRequest{
		Payload: `{"event":"user.created","data":{"name":"Alice"}}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Safe {
		t.Error("expected clean payload to be safe")
	}
}

func TestInspectPayload_SQLInjection(t *testing.T) {
	svc := NewService(nil, nil)
	result, err := svc.InspectPayload(context.Background(), "tenant-1", &InspectPayloadRequest{
		Payload: `{"query":"SELECT * FROM users UNION SELECT password FROM admin; --"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Safe {
		t.Error("expected SQL injection to be flagged")
	}
	found := false
	for _, f := range result.Findings {
		if f.Type == ThreatSQLInjection {
			found = true
		}
	}
	if !found {
		t.Error("expected SQL injection finding")
	}
}

func TestInspectPayload_XSS(t *testing.T) {
	svc := NewService(nil, nil)
	result, err := svc.InspectPayload(context.Background(), "tenant-1", &InspectPayloadRequest{
		Payload: `{"html":"<script>alert('xss')</script>"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Safe {
		t.Error("expected XSS to be flagged")
	}
}

func TestInspectPayload_SSRF(t *testing.T) {
	svc := NewService(nil, nil)
	result, err := svc.InspectPayload(context.Background(), "tenant-1", &InspectPayloadRequest{
		Payload: `{"url":"http://169.254.169.254/latest/meta-data/"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Safe {
		t.Error("expected SSRF to be flagged")
	}
	if result.ThreatLevel != ThreatCritical {
		t.Errorf("expected critical threat, got %s", result.ThreatLevel)
	}
}

func TestInspectPayload_Empty(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.InspectPayload(context.Background(), "tenant-1", &InspectPayloadRequest{})
	if err == nil {
		t.Error("expected error for empty payload")
	}
}

func TestCreatePolicy(t *testing.T) {
	svc := NewService(nil, nil)
	policy, err := svc.CreatePolicy(context.Background(), "tenant-1", &CreatePolicyRequest{
		Name:  "Block suspicious IPs",
		Rules: []PolicyRule{{ID: "r1", Type: "block_ip", Action: "block"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !policy.Enabled {
		t.Error("expected policy to be enabled")
	}
}

func TestCreatePolicy_NoRules(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.CreatePolicy(context.Background(), "tenant-1", &CreatePolicyRequest{
		Name: "Empty policy",
	})
	if err == nil {
		t.Error("expected error for empty rules")
	}
}

func TestGetDashboard(t *testing.T) {
	svc := NewService(nil, nil)
	dashboard, err := svc.GetDashboard(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dashboard == nil {
		t.Error("expected dashboard")
	}
}

func TestDetectAnomalies(t *testing.T) {
	svc := NewService(nil, nil)
	anomalies, err := svc.DetectAnomalies(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(anomalies) == 0 {
		t.Error("expected anomaly reports")
	}
}

func TestGetIPReputation(t *testing.T) {
	svc := NewService(nil, nil)
	ctx := context.Background()

	rep, err := svc.GetIPReputation(ctx, "tenant-1", "1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.IP != "1.2.3.4" {
		t.Errorf("expected IP 1.2.3.4, got %s", rep.IP)
	}
	if rep.Category != "clean" {
		t.Errorf("expected clean category, got %s", rep.Category)
	}

	_, err = svc.GetIPReputation(ctx, "tenant-1", "")
	if err == nil {
		t.Error("expected error for empty IP")
	}
}

func TestCreateGeoFenceRule(t *testing.T) {
	svc := NewService(nil, nil)
	ctx := context.Background()

	rule, err := svc.CreateGeoFenceRule(ctx, "tenant-1", "Block high-risk", "block", []string{"CN", "RU"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.Action != "block" {
		t.Errorf("expected block action, got %s", rule.Action)
	}
	if len(rule.Countries) != 2 {
		t.Errorf("expected 2 countries, got %d", len(rule.Countries))
	}

	_, err = svc.CreateGeoFenceRule(ctx, "tenant-1", "Bad", "invalid", []string{"US"})
	if err == nil {
		t.Error("expected error for invalid action")
	}
}

func TestExportComplianceAudit(t *testing.T) {
	svc := NewService(nil, nil)
	ctx := context.Background()

	export, err := svc.ExportComplianceAudit(ctx, "tenant-1", time.Now().AddDate(0, -1, 0), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if export.TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", export.TenantID)
	}
	if export.GeneratedAt.IsZero() {
		t.Error("expected non-zero generated_at")
	}
}

func TestBlockIP(t *testing.T) {
	svc := NewService(nil, nil)
	ctx := context.Background()

	entry, err := svc.BlockIP(ctx, "tenant-1", "10.0.0.1", "manual block")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.IP != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %s", entry.IP)
	}
}
