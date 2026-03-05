package errors

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Helper functions for common error scenarios

// AbortWithError aborts the Gin context with a WebhookError
func AbortWithError(c *gin.Context, err *WebhookError) {
	if err.RequestID == "" {
		if requestID, exists := c.Get("request_id"); exists {
			if id, ok := requestID.(string); ok {
				err.RequestID = id
			}
		}
	}

	if err.TraceID == "" {
		if traceID, exists := c.Get("trace_id"); exists {
			if id, ok := traceID.(string); ok {
				err.TraceID = id
			}
		}
	}

	c.JSON(err.GetHTTPStatus(), ErrorResponse{Error: err})
	c.Abort()
}

// AbortWithValidationError creates and aborts with a validation error
func AbortWithValidationError(c *gin.Context, field, reason string) {
	err := NewValidationError(field, reason)
	AbortWithError(c, err)
}

// AbortWithUnauthorized aborts with an unauthorized error
func AbortWithUnauthorized(c *gin.Context) {
	AbortWithError(c, ErrUnauthorized)
}

// AbortWithForbidden aborts with a forbidden error
func AbortWithForbidden(c *gin.Context) {
	AbortWithError(c, ErrForbidden)
}

// AbortWithNotFound aborts with a not found error
func AbortWithNotFound(c *gin.Context, resource string) {
	err := NewWebhookError(
		"NOT_FOUND",
		fmt.Sprintf("%s not found", resource),
		CategoryNotFound,
		http.StatusNotFound,
		SeverityLow,
	)
	AbortWithError(c, err)
}

// AbortWithInternalError aborts with an internal server error
func AbortWithInternalError(c *gin.Context, cause error) {
	err := ErrInternalServer.WithCause(cause)
	AbortWithError(c, err)
}

// AbortWithDatabaseError aborts with a database error
func AbortWithDatabaseError(c *gin.Context, cause error) {
	err := FromDatabaseError(cause)
	AbortWithError(c, err)
}

// AbortWithQueueError aborts with a queue error
func AbortWithQueueError(c *gin.Context, cause error) {
	err := FromQueueError(cause)
	AbortWithError(c, err)
}

// AbortWithRateLimit aborts with a rate limit error
func AbortWithRateLimit(c *gin.Context, retryAfter int) {
	err := NewRateLimitError(retryAfter)
	AbortWithError(c, err)
}

// AbortWithQuotaExceeded aborts with a quota exceeded error
func AbortWithQuotaExceeded(c *gin.Context, currentUsage, limit int) {
	err := NewQuotaExceededError(currentUsage, limit)
	AbortWithError(c, err)
}

// AbortWithPayloadTooLarge aborts with a payload too large error
func AbortWithPayloadTooLarge(c *gin.Context, actualSize, maxSize int) {
	err := NewPayloadTooLargeError(actualSize, maxSize)
	AbortWithError(c, err)
}

// RespondWithError writes a structured error response without aborting the handler chain.
// Use this instead of AbortWithError when you want to return an error but continue middleware execution.
func RespondWithError(c *gin.Context, err *WebhookError) {
	if err.RequestID == "" {
		if requestID, exists := c.Get("request_id"); exists {
			if id, ok := requestID.(string); ok {
				err.RequestID = id
			}
		}
	}
	c.JSON(err.GetHTTPStatus(), ErrorResponse{Error: err})
}

// HandleBindError converts a Gin binding/validation error into a user-friendly WebhookError.
// Instead of exposing raw struct field names, it returns actionable messages.
func HandleBindError(c *gin.Context, err error) {
	webhookErr := NewWebhookError(
		"INVALID_REQUEST_BODY",
		"The request body could not be parsed or is missing required fields",
		CategoryValidation,
		http.StatusBadRequest,
		SeverityLow,
	).WithCause(err).WithDebuggingHints(
		"Ensure the request body is valid JSON",
		"Check that all required fields are present and correctly typed",
		fmt.Sprintf("Validation detail: %s", sanitizeValidationError(err.Error())),
	).WithDocumentation("https://docs.webhook-platform.com/api-reference")
	AbortWithError(c, webhookErr)
}

// sanitizeValidationError cleans up Go struct validation errors for end users
func sanitizeValidationError(msg string) string {
	// Replace Go struct field references with friendlier names
	replacer := strings.NewReplacer(
		"Key: '", "",
		"' Error:Field validation for '", ": field '",
		"' failed on the '", "' failed validation '",
		"' tag", "'",
	)
	return replacer.Replace(msg)
}

// HandleRepositoryError converts common repository errors to WebhookErrors
func HandleRepositoryError(err error) *WebhookError {
	if err == nil {
		return nil
	}

	errMsg := strings.ToLower(err.Error())

	// Not found errors
	if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "no rows") {
		return NewWebhookError(
			"RESOURCE_NOT_FOUND",
			"The requested resource was not found",
			CategoryNotFound,
			http.StatusNotFound,
			SeverityLow,
		).WithCause(err)
	}

	// Duplicate key errors
	if strings.Contains(errMsg, "duplicate") || strings.Contains(errMsg, "unique constraint") {
		return NewWebhookError(
			"RESOURCE_ALREADY_EXISTS",
			"A resource with these details already exists",
			CategoryValidation,
			http.StatusConflict,
			SeverityLow,
		).WithCause(err).WithDebuggingHints(
			"Check if a resource with the same identifier already exists",
			"Use a different identifier or update the existing resource",
		)
	}

	// Foreign key constraint errors
	if strings.Contains(errMsg, "foreign key") || strings.Contains(errMsg, "constraint") {
		return NewWebhookError(
			"INVALID_REFERENCE",
			"Referenced resource does not exist",
			CategoryValidation,
			http.StatusBadRequest,
			SeverityLow,
		).WithCause(err).WithDebuggingHints(
			"Ensure all referenced resources exist",
			"Check the IDs in your request",
		)
	}

	// Connection errors
	if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "timeout") {
		return FromDatabaseError(err)
	}

	// Default to database error
	return FromDatabaseError(err)
}

// HandleValidationError converts validation errors to WebhookErrors
func HandleValidationError(err error, field string) *WebhookError {
	if err == nil {
		return nil
	}

	return NewValidationError(field, err.Error()).WithCause(err)
}

// HandleHTTPError converts HTTP client errors to WebhookErrors
func HandleHTTPError(statusCode int, responseBody string, endpointURL string) *WebhookError {
	details := map[string]interface{}{
		"http_status":  statusCode,
		"endpoint_url": endpointURL,
	}

	if responseBody != "" {
		// Truncate long response bodies
		if len(responseBody) > 1000 {
			responseBody = responseBody[:1000] + "... (truncated)"
		}
		details["response_body"] = responseBody
	}

	var err *WebhookError

	switch {
	case statusCode >= 400 && statusCode < 500:
		// Client errors from webhook endpoint
		err = NewWebhookError(
			"WEBHOOK_CLIENT_ERROR",
			fmt.Sprintf("Webhook endpoint returned client error: %d", statusCode),
			CategoryDeliveryFailed,
			http.StatusBadGateway,
			SeverityMedium,
		).WithDebuggingHints(
			"Check your webhook endpoint implementation",
			"Verify the endpoint can handle the webhook payload format",
			"Ensure the endpoint returns a 2xx status code for successful processing",
		)

	case statusCode >= 500:
		// Server errors from webhook endpoint
		err = NewWebhookError(
			"WEBHOOK_SERVER_ERROR",
			fmt.Sprintf("Webhook endpoint returned server error: %d", statusCode),
			CategoryDeliveryFailed,
			http.StatusBadGateway,
			SeverityMedium,
		).WithDebuggingHints(
			"The webhook endpoint is experiencing server issues",
			"Check the endpoint's server logs for errors",
			"The delivery will be retried automatically",
		)

	default:
		// Unexpected status code
		err = NewWebhookError(
			"WEBHOOK_UNEXPECTED_RESPONSE",
			fmt.Sprintf("Webhook endpoint returned unexpected status: %d", statusCode),
			CategoryDeliveryFailed,
			http.StatusBadGateway,
			SeverityMedium,
		).WithDebuggingHints(
			"Check the webhook endpoint implementation",
			"Verify the endpoint returns appropriate status codes",
		)
	}

	return err.WithDetails(details)
}

// HandleNetworkError converts network errors to WebhookErrors
func HandleNetworkError(err error, endpointURL string) *WebhookError {
	if err == nil {
		return nil
	}

	errMsg := strings.ToLower(err.Error())
	details := map[string]interface{}{
		"endpoint_url":  endpointURL,
		"network_error": err.Error(),
	}

	var webhookErr *WebhookError

	switch {
	case strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded"):
		webhookErr = NewWebhookError(
			"WEBHOOK_TIMEOUT",
			"Webhook delivery timed out",
			CategoryTimeout,
			http.StatusGatewayTimeout,
			SeverityMedium,
		).WithDebuggingHints(
			"The webhook endpoint took too long to respond",
			"Check if the endpoint is responding slowly",
			"Consider optimizing the endpoint's response time",
			"The delivery will be retried automatically",
		)

	case strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "no route to host"):
		webhookErr = NewWebhookError(
			"WEBHOOK_UNREACHABLE",
			"Webhook endpoint is unreachable",
			CategoryExternalAPI,
			http.StatusBadGateway,
			SeverityMedium,
		).WithDebuggingHints(
			"Check that the webhook endpoint URL is correct",
			"Verify the endpoint is running and accessible from the internet",
			"Check firewall settings and network connectivity",
			"Ensure the endpoint is listening on the correct port",
		)

	case strings.Contains(errMsg, "tls") || strings.Contains(errMsg, "certificate"):
		webhookErr = NewWebhookError(
			"WEBHOOK_TLS_ERROR",
			"TLS/SSL error connecting to webhook endpoint",
			CategoryExternalAPI,
			http.StatusBadGateway,
			SeverityMedium,
		).WithDebuggingHints(
			"Check that the webhook endpoint has a valid SSL certificate",
			"Verify the certificate is not expired",
			"Ensure the certificate matches the domain name",
		)

	default:
		webhookErr = NewWebhookError(
			"WEBHOOK_NETWORK_ERROR",
			"Network error connecting to webhook endpoint",
			CategoryExternalAPI,
			http.StatusBadGateway,
			SeverityMedium,
		).WithDebuggingHints(
			"Check network connectivity to the webhook endpoint",
			"Verify the endpoint URL is correct and accessible",
			"The delivery will be retried automatically",
		)
	}

	return webhookErr.WithDetails(details).WithCause(err)
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	var webhookErr *WebhookError
	if stderrors.As(err, &webhookErr) {
		return webhookErr.IsRetryable()
	}

	// Check common retryable error patterns
	errMsg := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"network",
		"temporary",
		"unavailable",
		"deadline exceeded",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// GetErrorSeverity extracts the severity from an error
func GetErrorSeverity(err error) ErrorSeverity {
	var webhookErr *WebhookError
	if stderrors.As(err, &webhookErr) {
		return webhookErr.Severity
	}
	return SeverityMedium // Default severity for unknown errors
}

// GetErrorCategory extracts the category from an error
func GetErrorCategory(err error) ErrorCategory {
	var webhookErr *WebhookError
	if stderrors.As(err, &webhookErr) {
		return webhookErr.Category
	}
	return CategoryInternal // Default category for unknown errors
}

// WithContext adds context information to an error
func WithContext(err *WebhookError, ctx context.Context) *WebhookError {
	if err == nil {
		return nil
	}

	// Add context values if available
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok && err.RequestID == "" {
			err.RequestID = id
		}
	}

	if traceID := ctx.Value("trace_id"); traceID != nil {
		if id, ok := traceID.(string); ok && err.TraceID == "" {
			err.TraceID = id
		}
	}

	return err
}

// LogError logs an error with appropriate context
func LogError(logger interface{}, err error, context map[string]interface{}) {
	// This is a helper function that can be used when the full error handler is not available
	// In a real implementation, you would use the actual logger interface
	if context == nil {
		context = make(map[string]interface{})
	}

	var webhookErr *WebhookError
	if stderrors.As(err, &webhookErr) {
		context["error_code"] = webhookErr.Code
		context["error_category"] = webhookErr.Category
		context["error_severity"] = webhookErr.Severity
		context["request_id"] = webhookErr.RequestID
		context["trace_id"] = webhookErr.TraceID
	}

	context["error_message"] = err.Error()

	// Log based on severity if available
	if webhookErr != nil {
		switch webhookErr.Severity {
		case SeverityLow:
			// logger.Info("Error occurred", context)
		case SeverityMedium:
			// logger.Warn("Error occurred", context)
		case SeverityHigh, SeverityCritical:
			// logger.Error("Error occurred", context)
		}
	}
}
