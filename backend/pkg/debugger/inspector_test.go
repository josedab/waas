package debugger

import (
	"testing"
	"time"
)

func TestGenerateCurlCommand(t *testing.T) {
	req := InspectionRequest{
		Method: "POST",
		URL:    "https://example.com/webhook",
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"X-Webhook-Signature": "sha256=abc123",
		},
		Body: `{"event":"test"}`,
	}

	curl := GenerateCurlCommand(req)
	if curl.Command == "" {
		t.Error("expected non-empty curl command")
	}
	if curl.Verbose == "" {
		t.Error("expected non-empty verbose command")
	}

	// Check key parts are present
	for _, expected := range []string{"curl", "-X", "POST", "Content-Type", "example.com/webhook"} {
		if !contains(curl.Command, expected) {
			t.Errorf("curl command missing '%s': %s", expected, curl.Command)
		}
	}
	if !contains(curl.Verbose, "--trace-time") {
		t.Error("verbose command missing --trace-time")
	}
}

func TestGenerateCurlCommandGET(t *testing.T) {
	req := InspectionRequest{
		Method: "GET",
		URL:    "https://example.com/status",
	}
	curl := GenerateCurlCommand(req)
	if contains(curl.Command, "-X") {
		t.Error("GET request should not include -X flag")
	}
}

func TestExtractTimingFromTrace(t *testing.T) {
	trace := &DeliveryTrace{
		DeliveryID: "del-123",
		TotalDurMs: 250,
		CreatedAt:  time.Now(),
		Stages: []TraceStage{
			{Name: "dns_lookup", DurationMs: 20},
			{Name: "tcp_connect", DurationMs: 30},
			{Name: "tls_handshake", DurationMs: 50},
			{Name: StageDelivery, DurationMs: 100},
			{Name: StageResponse, DurationMs: 50},
		},
	}

	timing := ExtractTimingFromTrace(trace)
	if timing.DNSLookupMs != 20 {
		t.Errorf("expected dns 20, got %f", timing.DNSLookupMs)
	}
	if timing.TLSHandshakeMs != 50 {
		t.Errorf("expected tls 50, got %f", timing.TLSHandshakeMs)
	}
	if timing.TotalMs != 250 {
		t.Errorf("expected total 250, got %f", timing.TotalMs)
	}
}

func TestMatchesFilter(t *testing.T) {
	now := time.Now()
	trace := &DeliveryTrace{
		EndpointID:  "ep-1",
		FinalStatus: "success",
		TotalDurMs:  150,
		CreatedAt:   now,
		Stages: []TraceStage{
			{Name: StageReceived, Input: `{"event":"order.created"}`},
		},
	}

	tests := []struct {
		name     string
		filter   SearchFilter
		expected bool
	}{
		{"match all", SearchFilter{}, true},
		{"match endpoint", SearchFilter{EndpointID: "ep-1"}, true},
		{"wrong endpoint", SearchFilter{EndpointID: "ep-999"}, false},
		{"match status", SearchFilter{Status: "success"}, true},
		{"wrong status", SearchFilter{Status: "failed"}, false},
		{"min duration match", SearchFilter{MinDuration: 100}, true},
		{"min duration fail", SearchFilter{MinDuration: 200}, false},
		{"max duration match", SearchFilter{MaxDuration: 200}, true},
		{"max duration fail", SearchFilter{MaxDuration: 100}, false},
		{"payload contains match", SearchFilter{PayloadContains: "order.created"}, true},
		{"payload contains fail", SearchFilter{PayloadContains: "nonexistent"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchesFilter(trace, tt.filter); got != tt.expected {
				t.Errorf("MatchesFilter() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuildInspection(t *testing.T) {
	trace := &DeliveryTrace{
		DeliveryID: "del-abc",
		TotalDurMs: 200,
		CreatedAt:  time.Now(),
		Stages: []TraceStage{
			{
				Name:  StageReceived,
				Input: `{"event":"test"}`,
				Metadata: map[string]string{
					"url":          "https://example.com/hook",
					"Content-Type": "application/json",
				},
			},
			{Name: StageDelivery, DurationMs: 150},
			{
				Name:   StageResponse,
				Output: `{"ok":true}`,
				Metadata: map[string]string{"status_code": "200"},
			},
			{
				Name:       StageRetry,
				Error:      "timeout",
				DurationMs: 5000,
				Metadata:   map[string]string{"attempt": "1", "status_code": "503"},
			},
		},
	}

	insp := BuildInspection(trace)
	if insp.DeliveryID != "del-abc" {
		t.Errorf("expected delivery ID 'del-abc', got '%s'", insp.DeliveryID)
	}
	if insp.Request.URL != "https://example.com/hook" {
		t.Errorf("expected URL from metadata, got '%s'", insp.Request.URL)
	}
	if insp.Response.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", insp.Response.StatusCode)
	}
	if len(insp.Retries) != 1 {
		t.Fatalf("expected 1 retry, got %d", len(insp.Retries))
	}
	if insp.Retries[0].Attempt != 1 {
		t.Errorf("expected retry attempt 1, got %d", insp.Retries[0].Attempt)
	}
	if insp.Curl.Command == "" {
		t.Error("expected curl command to be generated")
	}
}

func TestFormatAsJSON(t *testing.T) {
	result := FormatAsJSON(map[string]string{"key": "value"})
	if result == "" {
		t.Error("expected non-empty JSON output")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
