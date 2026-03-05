package errors

import (
	"context"
	stderrors "errors"
	"github.com/josedab/waas/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// This file demonstrates how to integrate the error handling system
// into existing handlers and middleware

// ExampleHandler shows how to use the error handling system in a handler
type ExampleHandler struct {
	logger       *utils.Logger
	errorHandler *ErrorHandler
}

// NewExampleHandler creates a new example handler with error handling
func NewExampleHandler(logger *utils.Logger, errorHandler *ErrorHandler) *ExampleHandler {
	return &ExampleHandler{
		logger:       logger,
		errorHandler: errorHandler,
	}
}

// ExampleEndpoint demonstrates proper error handling in an API endpoint
func (h *ExampleHandler) ExampleEndpoint(c *gin.Context) {
	// Example 1: Validation error
	var request struct {
		Email string `json:"email" binding:"required,email"`
		Name  string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		// Use helper function for validation errors
		AbortWithValidationError(c, "request", err.Error())
		return
	}

	// Example 2: Database operation with error handling
	if err := h.performDatabaseOperation(request.Email); err != nil {
		// For this example, we'll use the error directly since it's already a WebhookError
		var webhookErr *WebhookError
		if stderrors.As(err, &webhookErr) {
			AbortWithError(c, webhookErr)
			return
		}
		// Convert repository error to structured error for other cases
		webhookErr = HandleRepositoryError(err)
		if webhookErr != nil {
			AbortWithError(c, webhookErr)
			return
		}
	}

	// Example 3: Business logic error
	if request.Email == "forbidden@example.com" {
		AbortWithForbidden(c)
		return
	}

	// Example 4: External API call with error handling
	if err := h.callExternalAPI(request.Email); err != nil {
		// For this example, we'll use the error directly since it's already a WebhookError
		var webhookErr *WebhookError
		if stderrors.As(err, &webhookErr) {
			AbortWithError(c, webhookErr)
			return
		}
		// Handle network/HTTP errors for other cases
		webhookErr = HandleNetworkError(err, "https://external-api.com")
		if webhookErr != nil {
			AbortWithError(c, webhookErr)
			return
		}
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{
		"message": "Operation completed successfully",
		"email":   request.Email,
	})
}

// performDatabaseOperation simulates a database operation that might fail
func (h *ExampleHandler) performDatabaseOperation(email string) error {
	// Simulate different types of database errors
	switch email {
	case "notfound@example.com":
		return NewWebhookError("USER_NOT_FOUND", "User not found", CategoryNotFound, http.StatusNotFound, SeverityLow)
	case "duplicate@example.com":
		return NewWebhookError("USER_EXISTS", "User already exists", CategoryValidation, http.StatusConflict, SeverityLow)
	case "dberror@example.com":
		return NewWebhookError("DB_CONNECTION_FAILED", "Database connection failed", CategoryDatabase, http.StatusInternalServerError, SeverityHigh)
	default:
		return nil
	}
}

// callExternalAPI simulates an external API call that might fail
func (h *ExampleHandler) callExternalAPI(email string) error {
	// Simulate different types of network errors
	switch email {
	case "timeout@example.com":
		return NewWebhookError("API_TIMEOUT", "External API timeout", CategoryTimeout, http.StatusGatewayTimeout, SeverityMedium)
	case "unreachable@example.com":
		return NewWebhookError("API_UNREACHABLE", "External API unreachable", CategoryExternalAPI, http.StatusBadGateway, SeverityMedium)
	default:
		return nil
	}
}

// SetupErrorHandlingMiddleware demonstrates how to set up the error handling middleware
func SetupErrorHandlingMiddleware(router *gin.Engine, logger *utils.Logger) *ErrorHandler {
	// Create alerter (in production, configure with real Slack/email settings)
	alerterConfig := &AlerterConfig{
		Enabled:            true,
		MaxAlertsPerMinute: 10,
		MaxAlertsPerHour:   100,
		SlackWebhookURL:    "",         // Configure in production
		EmailRecipients:    []string{}, // Configure in production
	}
	alerter := NewAlerter(context.Background(), alerterConfig, logger)

	// Create error handler
	errorHandler := NewErrorHandler(logger, alerter)

	// Add middleware in the correct order
	router.Use(RequestIDMiddleware())
	router.Use(TraceIDMiddleware())
	router.Use(errorHandler.Middleware()) // This should be one of the last middleware

	return errorHandler
}

// ExampleErrorHandlingInService demonstrates error handling in service layer
func ExampleErrorHandlingInService() {
	// Example of creating and using structured errors in service layer

	// Create a validation error with context
	validationErr := NewValidationError("email", "invalid format").
		WithRequestID("req_123").
		WithTraceID("trace_456").
		WithDebuggingHints(
			"Ensure email follows the format: user@domain.com",
			"Check for typos in the email address",
		)

	// Create a database error with cause
	dbErr := FromDatabaseError(NewWebhookError("CONNECTION_FAILED", "Database connection failed", CategoryDatabase, http.StatusInternalServerError, SeverityHigh)).
		WithDetails(map[string]interface{}{
			"database": "postgres",
			"host":     "db.example.com",
		})

	// Create a delivery error with full context
	deliveryErr := NewDeliveryError("endpoint_123", "delivery_456", 500, "Internal Server Error").
		WithRequestID("req_789").
		WithDebuggingHints(
			"Check if the webhook endpoint is responding correctly",
			"Verify the endpoint can handle the payload format",
		)

	// Log errors with context (in real code, you'd use these errors appropriately)
	_ = validationErr
	_ = dbErr
	_ = deliveryErr
}

// ExampleCustomErrorDefinition shows how to create custom error definitions
func ExampleCustomErrorDefinition() {
	// Define a custom error for a specific business case
	ErrCustomBusinessLogic := NewWebhookError(
		"CUSTOM_BUSINESS_ERROR",
		"Custom business logic validation failed",
		CategoryValidation,
		http.StatusBadRequest,
		SeverityLow,
	).WithDebuggingHints(
		"Check the business rules for this operation",
		"Ensure all prerequisites are met",
		"Contact support if the error persists",
	).WithDocumentation("https://docs.webhook-platform.com/business-rules")

	// Use the custom error
	_ = ErrCustomBusinessLogic
}

// ExampleErrorHandlingInMiddleware demonstrates error handling in custom middleware
func ExampleErrorHandlingInMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Example: Rate limiting middleware with structured errors

		// Simulate rate limit check
		if c.GetHeader("X-Test-Rate-Limit") == "exceeded" {
			AbortWithRateLimit(c, 60) // Retry after 60 seconds
			return
		}

		// Simulate quota check
		if c.GetHeader("X-Test-Quota") == "exceeded" {
			AbortWithQuotaExceeded(c, 1500, 1000) // Current usage: 1500, Limit: 1000
			return
		}

		// Continue to next handler
		c.Next()
	}
}

// ExampleErrorHandlingInRepository demonstrates error handling in repository layer
type ExampleRepository struct {
	logger *utils.Logger
}

func (r *ExampleRepository) GetUser(id string) (*User, error) {
	// Simulate different repository error scenarios
	switch id {
	case "not_found":
		// Return a structured error that can be handled by HandleRepositoryError
		return nil, NewWebhookError(
			"USER_NOT_FOUND",
			"User not found",
			CategoryNotFound,
			http.StatusNotFound,
			SeverityLow,
		).WithDetails(map[string]interface{}{
			"user_id": id,
		})

	case "db_error":
		// Return a database error that will be categorized correctly
		return nil, FromDatabaseError(NewWebhookError(
			"DB_QUERY_FAILED",
			"Database query failed",
			CategoryDatabase,
			http.StatusInternalServerError,
			SeverityHigh,
		))

	default:
		// Return success case
		return &User{ID: id, Email: "user@example.com"}, nil
	}
}

// User represents a user entity (example)
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// ExampleErrorHandlingInDeliveryEngine demonstrates error handling in delivery engine
func ExampleErrorHandlingInDeliveryEngine(endpointURL string, payload []byte) error {
	// Simulate HTTP client call
	statusCode := 500
	responseBody := "Internal Server Error"

	// Handle HTTP errors with structured error response
	if statusCode >= 400 {
		return HandleHTTPError(statusCode, responseBody, endpointURL)
	}

	return nil
}

// ExampleErrorAggregation demonstrates how to collect and report multiple errors
func ExampleErrorAggregation() []error {
	var errors []error

	// Collect multiple validation errors
	if true { // Some validation condition
		errors = append(errors, NewValidationError("email", "required"))
	}

	if true { // Another validation condition
		errors = append(errors, NewValidationError("name", "too short"))
	}

	// In a real handler, you might want to return all validation errors at once
	return errors
}

// ExampleErrorMetrics demonstrates how errors can be used for metrics collection
func ExampleErrorMetrics(err error) {
	var webhookErr *WebhookError
	if stderrors.As(err, &webhookErr) {
		// Collect metrics based on error properties
		// metrics.IncrementCounter("errors_total", map[string]string{
		//     "category": string(webhookErr.Category),
		//     "severity": string(webhookErr.Severity),
		//     "code":     webhookErr.Code,
		// })

		// Log for monitoring
		if webhookErr.ShouldAlert() {
			// This error should trigger monitoring alerts
		}

		// Track retryable vs non-retryable errors
		if webhookErr.IsRetryable() {
			// metrics.IncrementCounter("retryable_errors_total")
		} else {
			// metrics.IncrementCounter("non_retryable_errors_total")
		}
	}
}
