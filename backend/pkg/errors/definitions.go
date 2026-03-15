package errors

import (
	"net/http"
)

// Predefined error definitions for common scenarios

// Authentication and Authorization Errors
var (
	ErrUnauthorized = NewWebhookError(
		"UNAUTHORIZED",
		"Authentication required",
		CategoryAuthentication,
		http.StatusUnauthorized,
		SeverityMedium,
	).WithDebuggingHints(
		"Ensure you have included a valid API key in the Authorization header",
		"Check that your API key has not expired or been revoked",
		"Verify the API key format: 'Bearer <your-api-key>'",
	).WithDocumentation("https://docs.webhook-platform.com/authentication")

	ErrInvalidAPIKey = NewWebhookError(
		"INVALID_API_KEY",
		"Invalid or expired API key",
		CategoryAuthentication,
		http.StatusUnauthorized,
		SeverityMedium,
	).WithDebuggingHints(
		"Verify your API key is correct and has not been regenerated",
		"Check that the API key belongs to the correct tenant",
		"Ensure the API key is not expired",
	).WithDocumentation("https://docs.webhook-platform.com/authentication")

	ErrForbidden = NewWebhookError(
		"FORBIDDEN",
		"Access denied to this resource",
		CategoryAuthorization,
		http.StatusForbidden,
		SeverityMedium,
	).WithDebuggingHints(
		"Verify you have permission to access this resource",
		"Check that the resource belongs to your tenant",
		"Ensure your subscription tier allows this operation",
	)

	ErrTenantNotFound = NewWebhookError(
		"TENANT_NOT_FOUND",
		"Tenant not found in context",
		CategoryAuthentication,
		http.StatusUnauthorized,
		SeverityHigh,
	).WithDebuggingHints(
		"This is likely an internal error - contact support if it persists",
		"Ensure you are using a valid API key",
	)
)

// Validation Errors
var (
	ErrInvalidRequest = NewWebhookError(
		"INVALID_REQUEST",
		"Invalid request format",
		CategoryValidation,
		http.StatusBadRequest,
		SeverityLow,
	).WithDebuggingHints(
		"Check that your request body is valid JSON",
		"Verify all required fields are present",
		"Ensure field types match the expected format",
	).WithDocumentation("https://docs.webhook-platform.com/api-reference")

	ErrInvalidURL = NewWebhookError(
		"INVALID_URL",
		"Invalid webhook URL",
		CategoryValidation,
		http.StatusBadRequest,
		SeverityLow,
	).WithDebuggingHints(
		"Webhook URLs must use HTTPS protocol",
		"Ensure the URL is properly formatted (e.g., https://example.com/webhook)",
		"Check that the domain is accessible from the internet",
	).WithDocumentation("https://docs.webhook-platform.com/webhook-endpoints")

	ErrInvalidPayload = NewWebhookError(
		"INVALID_PAYLOAD",
		"Invalid webhook payload",
		CategoryValidation,
		http.StatusBadRequest,
		SeverityLow,
	).WithDebuggingHints(
		"Webhook payload must be valid JSON",
		"Check for syntax errors in your JSON payload",
		"Ensure special characters are properly escaped",
	)

	ErrPayloadTooLarge = NewWebhookError(
		"PAYLOAD_TOO_LARGE",
		"Webhook payload exceeds maximum size limit",
		CategoryPayloadTooLarge,
		http.StatusBadRequest,
		SeverityLow,
	).WithDebuggingHints(
		"Reduce the size of your webhook payload",
		"Consider splitting large payloads into multiple smaller webhooks",
		"Remove unnecessary data from the payload",
	).WithDocumentation("https://docs.webhook-platform.com/limits")

	ErrInvalidID = NewWebhookError(
		"INVALID_ID",
		"Invalid ID format",
		CategoryValidation,
		http.StatusBadRequest,
		SeverityLow,
	).WithDebuggingHints(
		"IDs must be valid UUIDs",
		"Check the ID format in your request URL or body",
		"Ensure you're using the correct ID from previous API responses",
	)
)

// Resource Errors
var (
	ErrEndpointNotFound = NewWebhookError(
		"ENDPOINT_NOT_FOUND",
		"Webhook endpoint not found",
		CategoryNotFound,
		http.StatusNotFound,
		SeverityLow,
	).WithDebuggingHints(
		"Verify the endpoint ID is correct",
		"Check that the endpoint belongs to your tenant",
		"Ensure the endpoint has not been deleted",
	)

	ErrDeliveryNotFound = NewWebhookError(
		"DELIVERY_NOT_FOUND",
		"Webhook delivery not found",
		CategoryNotFound,
		http.StatusNotFound,
		SeverityLow,
	).WithDebuggingHints(
		"Verify the delivery ID is correct",
		"Check that the delivery belongs to your tenant",
		"Ensure the delivery exists and has not expired",
	)

	ErrEndpointInactive = NewWebhookError(
		"ENDPOINT_INACTIVE",
		"Webhook endpoint is not active",
		CategoryEndpointInactive,
		http.StatusBadRequest,
		SeverityLow,
	).WithDebuggingHints(
		"Activate the webhook endpoint before sending webhooks",
		"Check the endpoint status in your dashboard",
		"Endpoints may be automatically disabled after repeated failures",
	)

	ErrNoActiveEndpoints = NewWebhookError(
		"NO_ACTIVE_ENDPOINTS",
		"No active webhook endpoints found",
		CategoryNotFound,
		http.StatusBadRequest,
		SeverityLow,
	).WithDebuggingHints(
		"Create at least one webhook endpoint before sending webhooks",
		"Ensure your endpoints are active and not disabled",
		"Check that endpoints belong to the correct tenant",
	).WithDocumentation("https://docs.webhook-platform.com/webhook-endpoints")
)

// Rate Limiting and Quota Errors
var (
	ErrRateLimitExceeded = NewWebhookError(
		"RATE_LIMIT_EXCEEDED",
		"Rate limit exceeded",
		CategoryRateLimit,
		http.StatusTooManyRequests,
		SeverityMedium,
	).WithDebuggingHints(
		"Reduce the frequency of your API requests",
		"Implement exponential backoff in your retry logic",
		"Consider upgrading your subscription for higher rate limits",
		"Check the Retry-After header for when to retry",
	).WithDocumentation("https://docs.webhook-platform.com/rate-limits")

	ErrQuotaExceeded = NewWebhookError(
		"QUOTA_EXCEEDED",
		"Monthly quota exceeded",
		CategoryQuotaExceeded,
		http.StatusPaymentRequired,
		SeverityHigh,
	).WithDebuggingHints(
		"Upgrade your subscription plan for higher quotas",
		"Monitor your usage to avoid hitting limits",
		"Consider optimizing your webhook usage patterns",
		"Contact support if you need temporary quota increases",
	).WithDocumentation("https://docs.webhook-platform.com/quotas")
)

// Internal System Errors
var (
	ErrInternalServer = NewWebhookError(
		"INTERNAL_SERVER_ERROR",
		"An internal server error occurred",
		CategoryInternal,
		http.StatusInternalServerError,
		SeverityHigh,
	).WithDebuggingHints(
		"This is a temporary issue - please try again",
		"If the problem persists, contact support with the request ID",
		"Check our status page for any ongoing incidents",
	).WithDocumentation("https://status.webhook-platform.com")

	ErrDatabaseError = NewWebhookError(
		"DATABASE_ERROR",
		"Database operation failed",
		CategoryDatabase,
		http.StatusInternalServerError,
		SeverityHigh,
	).WithDebuggingHints(
		"This is a temporary issue - please try again",
		"If the problem persists, contact support",
		"Check our status page for database incidents",
	)

	ErrQueueError = NewWebhookError(
		"QUEUE_ERROR",
		"Message queue operation failed",
		CategoryQueue,
		http.StatusInternalServerError,
		SeverityHigh,
	).WithDebuggingHints(
		"Webhook delivery may be delayed",
		"The system will automatically retry failed operations",
		"Contact support if deliveries are consistently failing",
	)

	ErrExternalAPIError = NewWebhookError(
		"EXTERNAL_API_ERROR",
		"External API call failed",
		CategoryExternalAPI,
		http.StatusBadGateway,
		SeverityMedium,
	).WithDebuggingHints(
		"This may be a temporary issue with an external service",
		"The system will automatically retry the operation",
		"Check if the target webhook endpoint is accessible",
	)

	ErrTimeout = NewWebhookError(
		"TIMEOUT",
		"Operation timed out",
		CategoryTimeout,
		http.StatusGatewayTimeout,
		SeverityMedium,
	).WithDebuggingHints(
		"The operation took longer than expected",
		"Try again with a smaller request or payload",
		"Check if the target endpoint is responding slowly",
	)

	ErrServiceUnavailable = NewWebhookError(
		"SERVICE_UNAVAILABLE",
		"Service temporarily unavailable",
		CategoryUnavailable,
		http.StatusServiceUnavailable,
		SeverityCritical,
	).WithDebuggingHints(
		"The service is temporarily down for maintenance",
		"Check our status page for updates",
		"Try again after the maintenance window",
	).WithDocumentation("https://status.webhook-platform.com")
)

// Delivery Errors
var (
	ErrDeliveryFailed = NewWebhookError(
		"DELIVERY_FAILED",
		"Webhook delivery failed",
		CategoryDeliveryFailed,
		http.StatusBadGateway,
		SeverityMedium,
	).WithDebuggingHints(
		"Check that your webhook endpoint is accessible",
		"Verify the endpoint URL is correct and responds to HTTP requests",
		"Ensure your endpoint returns a 2xx status code",
		"Check your endpoint logs for any errors",
	).WithDocumentation("https://docs.webhook-platform.com/delivery-troubleshooting")

	ErrSignatureVerificationFailed = NewWebhookError(
		"SIGNATURE_VERIFICATION_FAILED",
		"Webhook signature verification failed",
		CategorySignatureInvalid,
		http.StatusBadRequest,
		SeverityMedium,
	).WithDebuggingHints(
		"Verify you're using the correct webhook secret",
		"Check your signature verification implementation",
		"Ensure you're using the same hashing algorithm (SHA-256)",
		"Make sure the payload hasn't been modified",
	).WithDocumentation("https://docs.webhook-platform.com/signature-verification")
)

// Helper functions to create contextual errors

// NewValidationError creates a validation error with specific details
func NewValidationError(field, reason string) *WebhookError {
	return ErrInvalidRequest.Clone().WithDetails(map[string]interface{}{
		"field":  field,
		"reason": reason,
	})
}

// NewPayloadTooLargeError creates a payload size error with specific limits
func NewPayloadTooLargeError(actualSize, maxSize int) *WebhookError {
	return ErrPayloadTooLarge.Clone().WithDetails(map[string]interface{}{
		"actual_size_bytes": actualSize,
		"max_size_bytes":    maxSize,
	})
}

// NewRateLimitError creates a rate limit error with retry information
func NewRateLimitError(retryAfter int) *WebhookError {
	return ErrRateLimitExceeded.Clone().WithDetails(map[string]interface{}{
		"retry_after_seconds": retryAfter,
	})
}

// NewQuotaExceededError creates a quota error with usage information
func NewQuotaExceededError(currentUsage, limit int) *WebhookError {
	return ErrQuotaExceeded.Clone().WithDetails(map[string]interface{}{
		"current_usage": currentUsage,
		"monthly_limit": limit,
	})
}

// NewDeliveryError creates a delivery error with endpoint and attempt information
func NewDeliveryError(endpointID, deliveryID string, httpStatus int, responseBody string) *WebhookError {
	details := map[string]interface{}{
		"endpoint_id": endpointID,
		"delivery_id": deliveryID,
	}

	if httpStatus > 0 {
		details["http_status"] = httpStatus
	}

	if responseBody != "" {
		// Truncate response body if too long
		if len(responseBody) > 500 {
			responseBody = responseBody[:500] + "... (truncated)"
		}
		details["response_body"] = responseBody
	}

	return ErrDeliveryFailed.Clone().WithDetails(details)
}

// WrapError wraps a generic error into a WebhookError
func WrapError(err error, code, message string, category ErrorCategory, severity ErrorSeverity) *WebhookError {
	webhookErr := NewWebhookError(code, message, category, 0, severity)
	return webhookErr.WithCause(err)
}

// FromValidationError converts a validation error to a WebhookError
func FromValidationError(err error) *WebhookError {
	return WrapError(err, "VALIDATION_ERROR", "Validation failed", CategoryValidation, SeverityLow).
		WithDebuggingHints(
			"Check the request format and required fields",
			"Ensure all field types match the expected format",
		)
}

// FromDatabaseError converts a database error to a WebhookError
func FromDatabaseError(err error) *WebhookError {
	return WrapError(err, "DATABASE_ERROR", "Database operation failed", CategoryDatabase, SeverityHigh).
		WithDebuggingHints(
			"This is a temporary issue - please try again",
			"Contact support if the problem persists",
		)
}

// FromQueueError converts a queue error to a WebhookError
func FromQueueError(err error) *WebhookError {
	return WrapError(err, "QUEUE_ERROR", "Message queue operation failed", CategoryQueue, SeverityHigh).
		WithDebuggingHints(
			"Webhook delivery may be delayed",
			"The system will automatically retry failed operations",
		)
}
