package security

import (
	"context"
	"testing"
)

func TestIPAllowlist(t *testing.T) {
	svc := NewIPAllowlistService()
	ctx := context.Background()

	// Add entry
	_, err := svc.AddEntry(ctx, "tenant1", "ep1", "192.168.1.0/24", "office")
	if err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	// Check allowed IP
	allowed, err := svc.CheckIP(ctx, "ep1", "192.168.1.50")
	if err != nil || !allowed {
		t.Error("expected IP to be allowed")
	}

	// Check denied IP
	allowed, err = svc.CheckIP(ctx, "ep1", "10.0.0.1")
	if err != nil || allowed {
		t.Error("expected IP to be denied")
	}

	// Check unconfigured endpoint
	allowed, err = svc.CheckIP(ctx, "ep_no_config", "10.0.0.1")
	if err != nil || !allowed {
		t.Error("expected IP to be allowed when no config")
	}
}

func TestPayloadEncryption(t *testing.T) {
	key := "01234567890123456789012345678901" // 32 bytes
	enc, err := NewPayloadEncryptor(key)
	if err != nil {
		t.Fatalf("NewPayloadEncryptor failed: %v", err)
	}

	plaintext := []byte(`{"event": "test", "data": {"key": "value"}}`)
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted doesn't match: got %s, want %s", decrypted, plaintext)
	}
}

func TestAnomalyDetection(t *testing.T) {
	detector := NewAnomalyDetector(DefaultAnomalyConfig())
	ctx := context.Background()

	// Build baseline with normal metrics
	for i := 0; i < 15; i++ {
		detector.Analyze(ctx, &DeliveryMetrics{
			TenantID:     "t1",
			EndpointID:   "ep1",
			TotalCount:   100,
			FailureCount: 2,
			AvgLatencyMs: 50,
		})
	}

	// Inject anomalous metrics
	anomalies := detector.Analyze(ctx, &DeliveryMetrics{
		TenantID:     "t1",
		EndpointID:   "ep1",
		TotalCount:   100,
		FailureCount: 50, // 50% failure rate
		AvgLatencyMs: 50,
	})

	if len(anomalies) == 0 {
		t.Error("expected anomaly to be detected for high failure rate")
	}

	found := false
	for _, a := range anomalies {
		if a.Type == AnomalyHighFailureRate {
			found = true
		}
	}
	if !found {
		t.Error("expected high_failure_rate anomaly")
	}
}
