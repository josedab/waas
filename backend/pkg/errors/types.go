package errors

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors for use with errors.Is() instead of string matching.
var (
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrNoSuchHost        = errors.New("no such host")
	ErrConnectionRefused = errors.New("connection refused")
)

// ErrorCategory represents the category of error
type ErrorCategory string

const (
	// Client error categories (4xx)
	CategoryValidation      ErrorCategory = "VALIDATION"
	CategoryAuthentication  ErrorCategory = "AUTHENTICATION"
	CategoryAuthorization   ErrorCategory = "AUTHORIZATION"
	CategoryNotFound        ErrorCategory = "NOT_FOUND"
	CategoryRateLimit       ErrorCategory = "RATE_LIMIT"
	CategoryQuotaExceeded   ErrorCategory = "QUOTA_EXCEEDED"
	CategoryPayloadTooLarge ErrorCategory = "PAYLOAD_TOO_LARGE"
	CategoryBadRequest      ErrorCategory = "BAD_REQUEST"

	// Server error categories (5xx)
	CategoryInternal    ErrorCategory = "INTERNAL"
	CategoryDatabase    ErrorCategory = "DATABASE"
	CategoryQueue       ErrorCategory = "QUEUE"
	CategoryExternalAPI ErrorCategory = "EXTERNAL_API"
	CategoryTimeout     ErrorCategory = "TIMEOUT"
	CategoryUnavailable ErrorCategory = "UNAVAILABLE"

	// Delivery error categories
	CategoryDeliveryFailed   ErrorCategory = "DELIVERY_FAILED"
	CategoryEndpointInactive ErrorCategory = "ENDPOINT_INACTIVE"
	CategorySignatureInvalid ErrorCategory = "SIGNATURE_INVALID"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "LOW"
	SeverityMedium   ErrorSeverity = "MEDIUM"
	SeverityHigh     ErrorSeverity = "HIGH"
	SeverityCritical ErrorSeverity = "CRITICAL"
)

// WebhookError represents a structured error in the webhook platform
type WebhookError struct {
	// Core error information
	Code     string        `json:"code"`
	Message  string        `json:"message"`
	Category ErrorCategory `json:"category"`

	// Additional context
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`

	// Internal fields (not exposed in JSON)
	HTTPStatus int           `json:"-"`
	Severity   ErrorSeverity `json:"-"`
	Cause      error         `json:"-"`

	// Debugging information
	DebuggingHints []string `json:"debugging_hints,omitempty"`
	Documentation  string   `json:"documentation,omitempty"`
}

// Error implements the error interface
func (e *WebhookError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error
func (e *WebhookError) Unwrap() error {
	return e.Cause
}

// WithDetails adds additional details to the error
func (e *WebhookError) WithDetails(details map[string]interface{}) *WebhookError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// WithRequestID adds a request ID to the error
func (e *WebhookError) WithRequestID(requestID string) *WebhookError {
	e.RequestID = requestID
	return e
}

// WithTraceID adds a trace ID to the error
func (e *WebhookError) WithTraceID(traceID string) *WebhookError {
	e.TraceID = traceID
	return e
}

// WithCause adds the underlying cause error
func (e *WebhookError) WithCause(cause error) *WebhookError {
	e.Cause = cause
	return e
}

// WithDebuggingHints adds debugging hints to help users resolve the error
func (e *WebhookError) WithDebuggingHints(hints ...string) *WebhookError {
	e.DebuggingHints = append(e.DebuggingHints, hints...)
	return e
}

// WithDocumentation adds a link to relevant documentation
func (e *WebhookError) WithDocumentation(docURL string) *WebhookError {
	e.Documentation = docURL
	return e
}

// ErrorResponse represents the standardized error response format
type ErrorResponse struct {
	Error *WebhookError `json:"error"`
}

// NewWebhookError creates a new WebhookError with the specified parameters
func NewWebhookError(code, message string, category ErrorCategory, httpStatus int, severity ErrorSeverity) *WebhookError {
	return &WebhookError{
		Code:       code,
		Message:    message,
		Category:   category,
		HTTPStatus: httpStatus,
		Severity:   severity,
		Timestamp:  time.Now(),
	}
}

// IsRetryable returns true if the error indicates a retryable condition
func (e *WebhookError) IsRetryable() bool {
	switch e.Category {
	case CategoryTimeout, CategoryUnavailable, CategoryQueue, CategoryExternalAPI:
		return true
	case CategoryInternal, CategoryDatabase:
		return true // Some internal errors might be retryable
	default:
		return false
	}
}

// ShouldAlert returns true if the error should trigger an alert
func (e *WebhookError) ShouldAlert() bool {
	return e.Severity == SeverityHigh || e.Severity == SeverityCritical
}

// GetHTTPStatus returns the appropriate HTTP status code for the error
func (e *WebhookError) GetHTTPStatus() int {
	if e.HTTPStatus != 0 {
		return e.HTTPStatus
	}

	// Default status codes based on category
	switch e.Category {
	case CategoryValidation, CategoryBadRequest, CategoryPayloadTooLarge:
		return http.StatusBadRequest
	case CategoryAuthentication:
		return http.StatusUnauthorized
	case CategoryAuthorization:
		return http.StatusForbidden
	case CategoryNotFound:
		return http.StatusNotFound
	case CategoryRateLimit:
		return http.StatusTooManyRequests
	case CategoryQuotaExceeded:
		return http.StatusPaymentRequired
	case CategoryUnavailable:
		return http.StatusServiceUnavailable
	case CategoryTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// GenerateRequestID generates a new request ID
func GenerateRequestID() string {
	return "req_" + uuid.New().String()
}

// GenerateTraceID generates a new trace ID
func GenerateTraceID() string {
	return "trace_" + uuid.New().String()
}
