package ai

import (
	"testing"
)

func TestClassifier_Classify(t *testing.T) {
	classifier := NewClassifier()

	tests := []struct {
		name         string
		errorMsg     string
		httpStatus   *int
		responseBody string
		wantCategory ErrorCategory
		wantRetry    bool
	}{
		{
			name:         "connection refused",
			errorMsg:     "dial tcp: connection refused",
			wantCategory: CategoryNetwork,
			wantRetry:    true,
		},
		{
			name:         "timeout",
			errorMsg:     "context deadline exceeded",
			wantCategory: CategoryTimeout,
			wantRetry:    true,
		},
		{
			name:         "DNS error",
			errorMsg:     "no such host",
			wantCategory: CategoryDNS,
			wantRetry:    true,
		},
		{
			name:         "certificate error",
			errorMsg:     "x509: certificate has expired",
			wantCategory: CategoryCertificate,
			wantRetry:    false,
		},
		{
			name:         "unauthorized by status",
			errorMsg:     "",
			httpStatus:   intPtr(401),
			wantCategory: CategoryAuth,
			wantRetry:    false,
		},
		{
			name:         "rate limited",
			errorMsg:     "HTTP 429: too many requests",
			httpStatus:   intPtr(429),
			wantCategory: CategoryRateLimit,
			wantRetry:    true,
		},
		{
			name:         "server error",
			errorMsg:     "",
			httpStatus:   intPtr(500),
			wantCategory: CategoryServerError,
			wantRetry:    true,
		},
		{
			name:         "bad request",
			errorMsg:     "",
			httpStatus:   intPtr(400),
			wantCategory: CategoryClientError,
			wantRetry:    false,
		},
		{
			name:         "invalid json in response",
			errorMsg:     "",
			httpStatus:   intPtr(400),
			responseBody: "invalid json payload",
			wantCategory: CategoryPayload,
			wantRetry:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.errorMsg, tt.httpStatus, tt.responseBody)

			if result.Category != tt.wantCategory {
				t.Errorf("Classify() category = %v, want %v", result.Category, tt.wantCategory)
			}

			if result.IsRetryable != tt.wantRetry {
				t.Errorf("Classify() retryable = %v, want %v", result.IsRetryable, tt.wantRetry)
			}
		})
	}
}

func TestClassifier_GetSuggestions(t *testing.T) {
	classifier := NewClassifier()

	tests := []struct {
		name           string
		classification ErrorClassification
		wantMinCount   int
	}{
		{
			name:           "network error suggestions",
			classification: ErrorClassification{Category: CategoryNetwork, IsRetryable: true},
			wantMinCount:   1,
		},
		{
			name:           "auth error suggestions",
			classification: ErrorClassification{Category: CategoryAuth},
			wantMinCount:   1,
		},
		{
			name:           "rate limit suggestions",
			classification: ErrorClassification{Category: CategoryRateLimit},
			wantMinCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := classifier.GetSuggestions(tt.classification, &DeliveryContext{})

			if len(suggestions) < tt.wantMinCount {
				t.Errorf("GetSuggestions() count = %d, want at least %d", len(suggestions), tt.wantMinCount)
			}
		})
	}
}

func TestClassifier_NormalizeError(t *testing.T) {
	classifier := NewClassifier()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normalize UUID",
			input: "failed for id 123e4567-e89b-12d3-a456-426614174000",
			want:  "failed for id <UUID>",
		},
		{
			name:  "normalize IP",
			input: "connection to 192.168.1.100 failed",
			want:  "connection to <IP> failed",
		},
		{
			name:  "normalize numbers",
			input: "error code 12345",
			want:  "error code <N>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifier.normalizeError(tt.input)
			if got != tt.want {
				t.Errorf("normalizeError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
