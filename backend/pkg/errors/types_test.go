package errors

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWebhookError(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		message    string
		category   ErrorCategory
		httpStatus int
		severity   ErrorSeverity
	}{
		{
			name:       "validation error",
			code:       "INVALID_REQUEST",
			message:    "Invalid request format",
			category:   CategoryValidation,
			httpStatus: http.StatusBadRequest,
			severity:   SeverityLow,
		},
		{
			name:       "internal error",
			code:       "INTERNAL_ERROR",
			message:    "Internal server error",
			category:   CategoryInternal,
			httpStatus: http.StatusInternalServerError,
			severity:   SeverityHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewWebhookError(tt.code, tt.message, tt.category, tt.httpStatus, tt.severity)
			
			assert.Equal(t, tt.code, err.Code)
			assert.Equal(t, tt.message, err.Message)
			assert.Equal(t, tt.category, err.Category)
			assert.Equal(t, tt.httpStatus, err.HTTPStatus)
			assert.Equal(t, tt.severity, err.Severity)
			assert.WithinDuration(t, time.Now(), err.Timestamp, time.Second)
		})
	}
}

func TestWebhookError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *WebhookError
		cause    error
		expected string
	}{
		{
			name: "without cause",
			err: &WebhookError{
				Code:    "TEST_ERROR",
				Message: "Test error message",
			},
			expected: "TEST_ERROR: Test error message",
		},
		{
			name: "with cause",
			err: &WebhookError{
				Code:    "TEST_ERROR",
				Message: "Test error message",
			},
			cause:    errors.New("underlying error"),
			expected: "TEST_ERROR: Test error message (caused by: underlying error)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cause != nil {
				tt.err.Cause = tt.cause
			}
			
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestWebhookError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &WebhookError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
		Cause:   cause,
	}
	
	assert.Equal(t, cause, err.Unwrap())
}

func TestWebhookError_WithDetails(t *testing.T) {
	err := &WebhookError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
	}
	
	details := map[string]interface{}{
		"field": "test_field",
		"value": 123,
	}
	
	result := err.WithDetails(details)
	
	assert.Equal(t, err, result) // Should return same instance
	assert.Equal(t, details, err.Details)
	
	// Test adding more details
	moreDetails := map[string]interface{}{
		"another_field": "another_value",
	}
	
	err.WithDetails(moreDetails)
	
	assert.Equal(t, "test_field", err.Details["field"])
	assert.Equal(t, 123, err.Details["value"])
	assert.Equal(t, "another_value", err.Details["another_field"])
}

func TestWebhookError_WithRequestID(t *testing.T) {
	err := &WebhookError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
	}
	
	requestID := "req_123456"
	result := err.WithRequestID(requestID)
	
	assert.Equal(t, err, result)
	assert.Equal(t, requestID, err.RequestID)
}

func TestWebhookError_WithTraceID(t *testing.T) {
	err := &WebhookError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
	}
	
	traceID := "trace_123456"
	result := err.WithTraceID(traceID)
	
	assert.Equal(t, err, result)
	assert.Equal(t, traceID, err.TraceID)
}

func TestWebhookError_WithCause(t *testing.T) {
	err := &WebhookError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
	}
	
	cause := errors.New("underlying error")
	result := err.WithCause(cause)
	
	assert.Equal(t, err, result)
	assert.Equal(t, cause, err.Cause)
}

func TestWebhookError_WithDebuggingHints(t *testing.T) {
	err := &WebhookError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
	}
	
	hints := []string{"hint1", "hint2"}
	result := err.WithDebuggingHints(hints...)
	
	assert.Equal(t, err, result)
	assert.Equal(t, hints, err.DebuggingHints)
	
	// Test adding more hints
	err.WithDebuggingHints("hint3")
	
	expected := []string{"hint1", "hint2", "hint3"}
	assert.Equal(t, expected, err.DebuggingHints)
}

func TestWebhookError_WithDocumentation(t *testing.T) {
	err := &WebhookError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
	}
	
	docURL := "https://docs.example.com/errors"
	result := err.WithDocumentation(docURL)
	
	assert.Equal(t, err, result)
	assert.Equal(t, docURL, err.Documentation)
}

func TestWebhookError_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		category ErrorCategory
		expected bool
	}{
		{
			name:     "timeout error is retryable",
			category: CategoryTimeout,
			expected: true,
		},
		{
			name:     "unavailable error is retryable",
			category: CategoryUnavailable,
			expected: true,
		},
		{
			name:     "queue error is retryable",
			category: CategoryQueue,
			expected: true,
		},
		{
			name:     "external API error is retryable",
			category: CategoryExternalAPI,
			expected: true,
		},
		{
			name:     "internal error is retryable",
			category: CategoryInternal,
			expected: true,
		},
		{
			name:     "database error is retryable",
			category: CategoryDatabase,
			expected: true,
		},
		{
			name:     "validation error is not retryable",
			category: CategoryValidation,
			expected: false,
		},
		{
			name:     "authentication error is not retryable",
			category: CategoryAuthentication,
			expected: false,
		},
		{
			name:     "authorization error is not retryable",
			category: CategoryAuthorization,
			expected: false,
		},
		{
			name:     "not found error is not retryable",
			category: CategoryNotFound,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &WebhookError{
				Category: tt.category,
			}
			
			assert.Equal(t, tt.expected, err.IsRetryable())
		})
	}
}

func TestWebhookError_ShouldAlert(t *testing.T) {
	tests := []struct {
		name     string
		severity ErrorSeverity
		expected bool
	}{
		{
			name:     "low severity should not alert",
			severity: SeverityLow,
			expected: false,
		},
		{
			name:     "medium severity should not alert",
			severity: SeverityMedium,
			expected: false,
		},
		{
			name:     "high severity should alert",
			severity: SeverityHigh,
			expected: true,
		},
		{
			name:     "critical severity should alert",
			severity: SeverityCritical,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &WebhookError{
				Severity: tt.severity,
			}
			
			assert.Equal(t, tt.expected, err.ShouldAlert())
		})
	}
}

func TestWebhookError_GetHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		category   ErrorCategory
		httpStatus int
		expected   int
	}{
		{
			name:       "explicit HTTP status",
			category:   CategoryValidation,
			httpStatus: http.StatusTeapot,
			expected:   http.StatusTeapot,
		},
		{
			name:     "validation error default",
			category: CategoryValidation,
			expected: http.StatusBadRequest,
		},
		{
			name:     "bad request error default",
			category: CategoryBadRequest,
			expected: http.StatusBadRequest,
		},
		{
			name:     "payload too large error default",
			category: CategoryPayloadTooLarge,
			expected: http.StatusBadRequest,
		},
		{
			name:     "authentication error default",
			category: CategoryAuthentication,
			expected: http.StatusUnauthorized,
		},
		{
			name:     "authorization error default",
			category: CategoryAuthorization,
			expected: http.StatusForbidden,
		},
		{
			name:     "not found error default",
			category: CategoryNotFound,
			expected: http.StatusNotFound,
		},
		{
			name:     "rate limit error default",
			category: CategoryRateLimit,
			expected: http.StatusTooManyRequests,
		},
		{
			name:     "quota exceeded error default",
			category: CategoryQuotaExceeded,
			expected: http.StatusPaymentRequired,
		},
		{
			name:     "unavailable error default",
			category: CategoryUnavailable,
			expected: http.StatusServiceUnavailable,
		},
		{
			name:     "timeout error default",
			category: CategoryTimeout,
			expected: http.StatusGatewayTimeout,
		},
		{
			name:     "unknown category default",
			category: "UNKNOWN",
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &WebhookError{
				Category:   tt.category,
				HTTPStatus: tt.httpStatus,
			}
			
			assert.Equal(t, tt.expected, err.GetHTTPStatus())
		})
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()
	
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "req_")
	assert.Contains(t, id2, "req_")
}

func TestGenerateTraceID(t *testing.T) {
	id1 := GenerateTraceID()
	id2 := GenerateTraceID()
	
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "trace_")
	assert.Contains(t, id2, "trace_")
}

func TestErrorResponse(t *testing.T) {
	err := NewWebhookError(
		"TEST_ERROR",
		"Test error message",
		CategoryValidation,
		http.StatusBadRequest,
		SeverityLow,
	)
	
	response := ErrorResponse{Error: err}
	
	assert.Equal(t, err, response.Error)
}

func TestWebhookErrorChaining(t *testing.T) {
	// Test method chaining
	err := NewWebhookError(
		"TEST_ERROR",
		"Test error message",
		CategoryValidation,
		http.StatusBadRequest,
		SeverityLow,
	).WithRequestID("req_123").
		WithTraceID("trace_456").
		WithDetails(map[string]interface{}{"field": "value"}).
		WithCause(errors.New("underlying")).
		WithDebuggingHints("hint1", "hint2").
		WithDocumentation("https://docs.example.com")
	
	assert.Equal(t, "TEST_ERROR", err.Code)
	assert.Equal(t, "Test error message", err.Message)
	assert.Equal(t, CategoryValidation, err.Category)
	assert.Equal(t, http.StatusBadRequest, err.HTTPStatus)
	assert.Equal(t, SeverityLow, err.Severity)
	assert.Equal(t, "req_123", err.RequestID)
	assert.Equal(t, "trace_456", err.TraceID)
	assert.Equal(t, "value", err.Details["field"])
	assert.Equal(t, "underlying", err.Cause.Error())
	assert.Equal(t, []string{"hint1", "hint2"}, err.DebuggingHints)
	assert.Equal(t, "https://docs.example.com", err.Documentation)
}