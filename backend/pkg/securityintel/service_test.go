package securityintel

import (
	"context"
	"testing"
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
