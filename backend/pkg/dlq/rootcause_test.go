package dlq

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeRootCause(t *testing.T) {
	svc := NewService()

	status429 := 429
	entry := DLQEntry{
		ID:              "entry-1",
		TenantID:        "tenant-1",
		EndpointID:      "ep-1",
		ErrorType:       "rate_limited",
		FinalHTTPStatus: &status429,
		Payload:         json.RawMessage(`{}`),
		Headers:         json.RawMessage(`{}`),
		RetryCount:      3,
		MaxRetries:      5,
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(24 * time.Hour),
		AllAttempts: []AttemptDetail{
			{AttemptNumber: 1, HTTPStatus: &status429, AttemptedAt: time.Now(), DurationMs: 100},
		},
	}

	svc.mu.Lock()
	svc.entries["entry-1"] = &entry
	svc.mu.Unlock()

	analysis, err := svc.AnalyzeRootCause(context.Background(), "entry-1")
	require.NoError(t, err)
	assert.Equal(t, RootCauseRateLimit, analysis.Category)
	assert.Equal(t, SeverityMedium, analysis.Severity)
	assert.NotEmpty(t, analysis.Summary)
	assert.NotEmpty(t, analysis.Suggestions)
	assert.True(t, analysis.Confidence > 0)
}

func TestAnalyzeRootCauseNotFound(t *testing.T) {
	svc := NewService()
	_, err := svc.AnalyzeRootCause(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestClassifyRootCause(t *testing.T) {
	tests := []struct {
		name     string
		entry    DLQEntry
		expected string
	}{
		{"timeout", DLQEntry{ErrorType: "timeout"}, RootCauseTimeout},
		{"dns", DLQEntry{ErrorType: "dns_error"}, RootCauseDNSFailure},
		{"tls", DLQEntry{ErrorType: "tls_handshake_failed"}, RootCauseTLSError},
		{"connection refused", DLQEntry{ErrorType: "connection_refused"}, RootCauseEndpointDown},
		{"401", DLQEntry{FinalHTTPStatus: intPtr(401)}, RootCauseAuthFailure},
		{"429", DLQEntry{FinalHTTPStatus: intPtr(429)}, RootCauseRateLimit},
		{"400", DLQEntry{FinalHTTPStatus: intPtr(400)}, RootCausePayloadRejected},
		{"500", DLQEntry{FinalHTTPStatus: intPtr(500)}, RootCauseServerError},
		{"unknown", DLQEntry{ErrorType: "something_weird"}, RootCauseUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category, _ := classifyRootCause(&tt.entry)
			assert.Equal(t, tt.expected, category)
		})
	}
}

func TestSmartRetryRecommendation(t *testing.T) {
	svc := NewService()

	status429 := 429
	svc.mu.Lock()
	svc.entries["entry-rl"] = &DLQEntry{
		ID: "entry-rl", ErrorType: "rate_limit", FinalHTTPStatus: &status429,
		Payload: json.RawMessage(`{}`), Headers: json.RawMessage(`{}`),
		RetryCount: 2, MaxRetries: 5, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour),
	}
	status401 := 401
	svc.entries["entry-auth"] = &DLQEntry{
		ID: "entry-auth", ErrorType: "auth_failure", FinalHTTPStatus: &status401,
		Payload: json.RawMessage(`{}`), Headers: json.RawMessage(`{}`),
		RetryCount: 1, MaxRetries: 5, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour),
	}
	svc.mu.Unlock()

	rec, err := svc.GetSmartRetryRecommendation(context.Background(), "entry-rl")
	require.NoError(t, err)
	assert.True(t, rec.ShouldRetry)
	assert.True(t, rec.SuccessProbability > 0)

	rec, err = svc.GetSmartRetryRecommendation(context.Background(), "entry-auth")
	require.NoError(t, err)
	assert.False(t, rec.ShouldRetry)
}
