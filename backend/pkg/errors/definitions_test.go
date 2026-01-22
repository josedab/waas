package errors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPredefinedErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      *WebhookError
		code     string
		category ErrorCategory
		severity ErrorSeverity
		status   int
	}{
		{
			name:     "ErrUnauthorized",
			err:      ErrUnauthorized,
			code:     "UNAUTHORIZED",
			category: CategoryAuthentication,
			severity: SeverityMedium,
			status:   http.StatusUnauthorized,
		},
		{
			name:     "ErrInvalidAPIKey",
			err:      ErrInvalidAPIKey,
			code:     "INVALID_API_KEY",
			category: CategoryAuthentication,
			severity: SeverityMedium,
			status:   http.StatusUnauthorized,
		},
		{
			name:     "ErrForbidden",
			err:      ErrForbidden,
			code:     "FORBIDDEN",
			category: CategoryAuthorization,
			severity: SeverityMedium,
			status:   http.StatusForbidden,
		},
		{
			name:     "ErrTenantNotFound",
			err:      ErrTenantNotFound,
			code:     "TENANT_NOT_FOUND",
			category: CategoryAuthentication,
			severity: SeverityHigh,
			status:   http.StatusUnauthorized,
		},
		{
			name:     "ErrInvalidRequest",
			err:      ErrInvalidRequest,
			code:     "INVALID_REQUEST",
			category: CategoryValidation,
			severity: SeverityLow,
			status:   http.StatusBadRequest,
		},
		{
			name:     "ErrInvalidURL",
			err:      ErrInvalidURL,
			code:     "INVALID_URL",
			category: CategoryValidation,
			severity: SeverityLow,
			status:   http.StatusBadRequest,
		},
		{
			name:     "ErrInvalidPayload",
			err:      ErrInvalidPayload,
			code:     "INVALID_PAYLOAD",
			category: CategoryValidation,
			severity: SeverityLow,
			status:   http.StatusBadRequest,
		},
		{
			name:     "ErrPayloadTooLarge",
			err:      ErrPayloadTooLarge,
			code:     "PAYLOAD_TOO_LARGE",
			category: CategoryPayloadTooLarge,
			severity: SeverityLow,
			status:   http.StatusBadRequest,
		},
		{
			name:     "ErrInvalidID",
			err:      ErrInvalidID,
			code:     "INVALID_ID",
			category: CategoryValidation,
			severity: SeverityLow,
			status:   http.StatusBadRequest,
		},
		{
			name:     "ErrEndpointNotFound",
			err:      ErrEndpointNotFound,
			code:     "ENDPOINT_NOT_FOUND",
			category: CategoryNotFound,
			severity: SeverityLow,
			status:   http.StatusNotFound,
		},
		{
			name:     "ErrDeliveryNotFound",
			err:      ErrDeliveryNotFound,
			code:     "DELIVERY_NOT_FOUND",
			category: CategoryNotFound,
			severity: SeverityLow,
			status:   http.StatusNotFound,
		},
		{
			name:     "ErrEndpointInactive",
			err:      ErrEndpointInactive,
			code:     "ENDPOINT_INACTIVE",
			category: CategoryEndpointInactive,
			severity: SeverityLow,
			status:   http.StatusBadRequest,
		},
		{
			name:     "ErrNoActiveEndpoints",
			err:      ErrNoActiveEndpoints,
			code:     "NO_ACTIVE_ENDPOINTS",
			category: CategoryNotFound,
			severity: SeverityLow,
			status:   http.StatusBadRequest,
		},
		{
			name:     "ErrRateLimitExceeded",
			err:      ErrRateLimitExceeded,
			code:     "RATE_LIMIT_EXCEEDED",
			category: CategoryRateLimit,
			severity: SeverityMedium,
			status:   http.StatusTooManyRequests,
		},
		{
			name:     "ErrQuotaExceeded",
			err:      ErrQuotaExceeded,
			code:     "QUOTA_EXCEEDED",
			category: CategoryQuotaExceeded,
			severity: SeverityHigh,
			status:   http.StatusPaymentRequired,
		},
		{
			name:     "ErrInternalServer",
			err:      ErrInternalServer,
			code:     "INTERNAL_SERVER_ERROR",
			category: CategoryInternal,
			severity: SeverityHigh,
			status:   http.StatusInternalServerError,
		},
		{
			name:     "ErrDatabaseError",
			err:      ErrDatabaseError,
			code:     "DATABASE_ERROR",
			category: CategoryDatabase,
			severity: SeverityHigh,
			status:   http.StatusInternalServerError,
		},
		{
			name:     "ErrQueueError",
			err:      ErrQueueError,
			code:     "QUEUE_ERROR",
			category: CategoryQueue,
			severity: SeverityHigh,
			status:   http.StatusInternalServerError,
		},
		{
			name:     "ErrExternalAPIError",
			err:      ErrExternalAPIError,
			code:     "EXTERNAL_API_ERROR",
			category: CategoryExternalAPI,
			severity: SeverityMedium,
			status:   http.StatusBadGateway,
		},
		{
			name:     "ErrTimeout",
			err:      ErrTimeout,
			code:     "TIMEOUT",
			category: CategoryTimeout,
			severity: SeverityMedium,
			status:   http.StatusGatewayTimeout,
		},
		{
			name:     "ErrServiceUnavailable",
			err:      ErrServiceUnavailable,
			code:     "SERVICE_UNAVAILABLE",
			category: CategoryUnavailable,
			severity: SeverityCritical,
			status:   http.StatusServiceUnavailable,
		},
		{
			name:     "ErrDeliveryFailed",
			err:      ErrDeliveryFailed,
			code:     "DELIVERY_FAILED",
			category: CategoryDeliveryFailed,
			severity: SeverityMedium,
			status:   http.StatusBadGateway,
		},
		{
			name:     "ErrSignatureVerificationFailed",
			err:      ErrSignatureVerificationFailed,
			code:     "SIGNATURE_VERIFICATION_FAILED",
			category: CategorySignatureInvalid,
			severity: SeverityMedium,
			status:   http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.Equal(t, tt.category, tt.err.Category)
			assert.Equal(t, tt.severity, tt.err.Severity)
			assert.Equal(t, tt.status, tt.err.GetHTTPStatus())
			assert.NotEmpty(t, tt.err.Message)
			assert.NotEmpty(t, tt.err.DebuggingHints)
		})
	}
}

func TestNewValidationError(t *testing.T) {
	t.Parallel()
	field := "email"
	reason := "invalid format"
	
	err := NewValidationError(field, reason)
	
	assert.Equal(t, "INVALID_REQUEST", err.Code)
	assert.Equal(t, CategoryValidation, err.Category)
	assert.Equal(t, SeverityLow, err.Severity)
	assert.Equal(t, field, err.Details["field"])
	assert.Equal(t, reason, err.Details["reason"])
}

func TestNewPayloadTooLargeError(t *testing.T) {
	t.Parallel()
	actualSize := 2048
	maxSize := 1024
	
	err := NewPayloadTooLargeError(actualSize, maxSize)
	
	assert.Equal(t, "PAYLOAD_TOO_LARGE", err.Code)
	assert.Equal(t, CategoryPayloadTooLarge, err.Category)
	assert.Equal(t, SeverityLow, err.Severity)
	assert.Equal(t, actualSize, err.Details["actual_size_bytes"])
	assert.Equal(t, maxSize, err.Details["max_size_bytes"])
}

func TestNewRateLimitError(t *testing.T) {
	t.Parallel()
	retryAfter := 60
	
	err := NewRateLimitError(retryAfter)
	
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", err.Code)
	assert.Equal(t, CategoryRateLimit, err.Category)
	assert.Equal(t, SeverityMedium, err.Severity)
	assert.Equal(t, retryAfter, err.Details["retry_after_seconds"])
}

func TestNewQuotaExceededError(t *testing.T) {
	t.Parallel()
	currentUsage := 1500
	limit := 1000
	
	err := NewQuotaExceededError(currentUsage, limit)
	
	assert.Equal(t, "QUOTA_EXCEEDED", err.Code)
	assert.Equal(t, CategoryQuotaExceeded, err.Category)
	assert.Equal(t, SeverityHigh, err.Severity)
	assert.Equal(t, currentUsage, err.Details["current_usage"])
	assert.Equal(t, limit, err.Details["monthly_limit"])
}

func TestNewDeliveryError(t *testing.T) {
	t.Parallel()
	endpointID := "endpoint_123"
	deliveryID := "delivery_456"
	httpStatus := 500
	responseBody := "Internal Server Error"
	
	err := NewDeliveryError(endpointID, deliveryID, httpStatus, responseBody)
	
	assert.Equal(t, "DELIVERY_FAILED", err.Code)
	assert.Equal(t, CategoryDeliveryFailed, err.Category)
	assert.Equal(t, SeverityMedium, err.Severity)
	assert.Equal(t, endpointID, err.Details["endpoint_id"])
	assert.Equal(t, deliveryID, err.Details["delivery_id"])
	assert.Equal(t, httpStatus, err.Details["http_status"])
	assert.Equal(t, responseBody, err.Details["response_body"])
}

func TestNewDeliveryError_TruncatesLongResponseBody(t *testing.T) {
	t.Parallel()
	endpointID := "endpoint_123"
	deliveryID := "delivery_456"
	httpStatus := 500
	
	// Create a response body longer than 500 characters
	longResponseBody := ""
	for i := 0; i < 600; i++ {
		longResponseBody += "a"
	}
	
	err := NewDeliveryError(endpointID, deliveryID, httpStatus, longResponseBody)
	
	responseBody := err.Details["response_body"].(string)
	assert.True(t, len(responseBody) <= 515) // 500 + "... (truncated)"
	assert.Contains(t, responseBody, "... (truncated)")
}

func TestWrapError(t *testing.T) {
	t.Parallel()
	originalErr := errors.New("original error")
	code := "WRAPPED_ERROR"
	message := "Wrapped error message"
	category := CategoryInternal
	severity := SeverityHigh
	
	err := WrapError(originalErr, code, message, category, severity)
	
	assert.Equal(t, code, err.Code)
	assert.Equal(t, message, err.Message)
	assert.Equal(t, category, err.Category)
	assert.Equal(t, severity, err.Severity)
	assert.Equal(t, originalErr, err.Cause)
}

func TestFromValidationError(t *testing.T) {
	t.Parallel()
	originalErr := errors.New("validation failed")
	
	err := FromValidationError(originalErr)
	
	assert.Equal(t, "VALIDATION_ERROR", err.Code)
	assert.Equal(t, "Validation failed", err.Message)
	assert.Equal(t, CategoryValidation, err.Category)
	assert.Equal(t, SeverityLow, err.Severity)
	assert.Equal(t, originalErr, err.Cause)
	assert.NotEmpty(t, err.DebuggingHints)
}

func TestFromDatabaseError(t *testing.T) {
	t.Parallel()
	originalErr := errors.New("database connection failed")
	
	err := FromDatabaseError(originalErr)
	
	assert.Equal(t, "DATABASE_ERROR", err.Code)
	assert.Equal(t, "Database operation failed", err.Message)
	assert.Equal(t, CategoryDatabase, err.Category)
	assert.Equal(t, SeverityHigh, err.Severity)
	assert.Equal(t, originalErr, err.Cause)
	assert.NotEmpty(t, err.DebuggingHints)
}

func TestFromQueueError(t *testing.T) {
	t.Parallel()
	originalErr := errors.New("queue operation failed")
	
	err := FromQueueError(originalErr)
	
	assert.Equal(t, "QUEUE_ERROR", err.Code)
	assert.Equal(t, "Message queue operation failed", err.Message)
	assert.Equal(t, CategoryQueue, err.Category)
	assert.Equal(t, SeverityHigh, err.Severity)
	assert.Equal(t, originalErr, err.Cause)
	assert.NotEmpty(t, err.DebuggingHints)
}

func TestPredefinedErrorsHaveDocumentation(t *testing.T) {
	t.Parallel()
	errorsWithDocs := []*WebhookError{
		ErrUnauthorized,
		ErrInvalidAPIKey,
		ErrInvalidRequest,
		ErrInvalidURL,
		ErrPayloadTooLarge,
		ErrNoActiveEndpoints,
		ErrRateLimitExceeded,
		ErrQuotaExceeded,
		ErrInternalServer,
		ErrServiceUnavailable,
		ErrDeliveryFailed,
		ErrSignatureVerificationFailed,
	}
	
	for _, err := range errorsWithDocs {
		t.Run(err.Code, func(t *testing.T) {
			assert.NotEmpty(t, err.Documentation, "Error %s should have documentation", err.Code)
		})
	}
}

func TestPredefinedErrorsHaveDebuggingHints(t *testing.T) {
	t.Parallel()
	allPredefinedErrors := []*WebhookError{
		ErrUnauthorized,
		ErrInvalidAPIKey,
		ErrForbidden,
		ErrTenantNotFound,
		ErrInvalidRequest,
		ErrInvalidURL,
		ErrInvalidPayload,
		ErrPayloadTooLarge,
		ErrInvalidID,
		ErrEndpointNotFound,
		ErrDeliveryNotFound,
		ErrEndpointInactive,
		ErrNoActiveEndpoints,
		ErrRateLimitExceeded,
		ErrQuotaExceeded,
		ErrInternalServer,
		ErrDatabaseError,
		ErrQueueError,
		ErrExternalAPIError,
		ErrTimeout,
		ErrServiceUnavailable,
		ErrDeliveryFailed,
		ErrSignatureVerificationFailed,
	}
	
	for _, err := range allPredefinedErrors {
		t.Run(err.Code, func(t *testing.T) {
			assert.NotEmpty(t, err.DebuggingHints, "Error %s should have debugging hints", err.Code)
		})
	}
}