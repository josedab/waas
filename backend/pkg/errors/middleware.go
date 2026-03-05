package errors

import (
	"context"
	stderrors "errors"
	"fmt"
	"github.com/josedab/waas/pkg/utils"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorHandler provides centralized error handling for the application
type ErrorHandler struct {
	logger  *utils.Logger
	alerter AlerterInterface
}

// AlerterInterface defines the interface for sending alerts
type AlerterInterface interface {
	SendAlert(ctx context.Context, err *WebhookError) error
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger *utils.Logger, alerter AlerterInterface) *ErrorHandler {
	return &ErrorHandler{
		logger:  logger,
		alerter: alerter,
	}
}

// Middleware returns a Gin middleware that handles panics and errors
func (h *ErrorHandler) Middleware() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(gin.DefaultWriter, h.recoveryHandler)
}

// recoveryHandler handles panics and converts them to structured errors
func (h *ErrorHandler) recoveryHandler(c *gin.Context, recovered interface{}) {
	var err *WebhookError

	if recovered != nil {
		// Handle panic
		stack := debug.Stack()

		err = NewWebhookError(
			"PANIC_RECOVERED",
			"An unexpected error occurred",
			CategoryInternal,
			http.StatusInternalServerError,
			SeverityCritical,
		).WithDetails(map[string]interface{}{
			"panic_value": fmt.Sprintf("%v", recovered),
			"stack_trace": string(stack),
		}).WithRequestID(h.getRequestID(c)).WithTraceID(h.getTraceID(c))

		h.logger.Error("Panic recovered", map[string]interface{}{
			"error":      fmt.Sprintf("%v", recovered),
			"path":       c.Request.URL.Path,
			"method":     c.Request.Method,
			"request_id": err.RequestID,
			"trace_id":   err.TraceID,
			"stack":      string(stack),
		})

		// Send critical alert for panics
		if h.alerter != nil {
			reqCtx := c.Request.Context()
			go func() {
				ctx, cancel := context.WithTimeout(context.WithoutCancel(reqCtx), 5*time.Second)
				defer cancel()
				if alertErr := h.alerter.SendAlert(ctx, err); alertErr != nil {
					h.logger.Error("Failed to send panic alert", map[string]interface{}{
						"error": alertErr.Error(),
					})
				}
			}()
		}
	}

	h.HandleError(c, err)
}

// HandleError processes and responds to errors in a standardized way
func (h *ErrorHandler) HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	var webhookErr *WebhookError

	// Convert error to WebhookError if it isn't already
	if stderrors.As(err, &webhookErr) {
		// webhookErr is already set
	} else {
		// Try to categorize the error based on its message or type
		webhookErr = h.categorizeError(err)
	}

	// Add request context if not already present
	if webhookErr.RequestID == "" {
		webhookErr.RequestID = h.getRequestID(c)
	}
	if webhookErr.TraceID == "" {
		webhookErr.TraceID = h.getTraceID(c)
	}

	// Log the error
	h.logError(c, webhookErr)

	// Send alert if necessary
	if webhookErr.ShouldAlert() && h.alerter != nil {
		reqCtx := c.Request.Context()
		go func() {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(reqCtx), 5*time.Second)
			defer cancel()
			if alertErr := h.alerter.SendAlert(ctx, webhookErr); alertErr != nil {
				h.logger.Error("Failed to send error alert", map[string]interface{}{
					"error":      alertErr.Error(),
					"request_id": webhookErr.RequestID,
				})
			}
		}()
	}

	// Prepare response
	response := ErrorResponse{Error: webhookErr}

	// Remove sensitive information from response in production
	if gin.Mode() == gin.ReleaseMode {
		h.sanitizeErrorForProduction(webhookErr)
	}

	// Set appropriate headers
	h.setErrorHeaders(c, webhookErr)

	// Send JSON response
	c.JSON(webhookErr.GetHTTPStatus(), response)
	c.Abort()
}

// categorizeError attempts to categorize a generic error
func (h *ErrorHandler) categorizeError(err error) *WebhookError {
	errMsg := err.Error()
	errMsgLower := strings.ToLower(errMsg)

	// Database errors (check specific database terms first)
	if strings.Contains(errMsgLower, "database") ||
		strings.Contains(errMsgLower, "sql") {
		return FromDatabaseError(err)
	}

	// Queue errors
	if strings.Contains(errMsgLower, "queue") ||
		strings.Contains(errMsgLower, "redis") ||
		strings.Contains(errMsgLower, "rabbitmq") {
		return FromQueueError(err)
	}

	// Validation errors
	if strings.Contains(errMsgLower, "validation") ||
		strings.Contains(errMsgLower, "invalid") ||
		strings.Contains(errMsgLower, "required") {
		return FromValidationError(err)
	}

	// Timeout errors
	if strings.Contains(errMsgLower, "timeout") ||
		strings.Contains(errMsgLower, "deadline exceeded") {
		return WrapError(err, "TIMEOUT", "Operation timed out", CategoryTimeout, SeverityMedium)
	}

	// Network errors (check before general connection errors)
	if strings.Contains(errMsgLower, "network") ||
		strings.Contains(errMsgLower, "connection refused") ||
		strings.Contains(errMsgLower, "no route to host") {
		return WrapError(err, "NETWORK_ERROR", "Network error occurred", CategoryExternalAPI, SeverityMedium)
	}

	// General connection errors (could be database)
	if strings.Contains(errMsgLower, "connection") {
		return FromDatabaseError(err)
	}

	// Default to internal server error
	return WrapError(err, "INTERNAL_ERROR", "An internal error occurred", CategoryInternal, SeverityHigh)
}

// logError logs the error with appropriate level and context
func (h *ErrorHandler) logError(c *gin.Context, err *WebhookError) {
	logData := map[string]interface{}{
		"error_code":    err.Code,
		"error_message": err.Message,
		"category":      err.Category,
		"severity":      err.Severity,
		"http_status":   err.GetHTTPStatus(),
		"request_id":    err.RequestID,
		"trace_id":      err.TraceID,
		"path":          c.Request.URL.Path,
		"method":        c.Request.Method,
		"user_agent":    c.Request.UserAgent(),
		"remote_addr":   c.ClientIP(),
	}

	// Add error details if present
	if err.Details != nil {
		logData["details"] = err.Details
	}

	// Add cause if present
	if err.Cause != nil {
		logData["cause"] = err.Cause.Error()
	}

	// Add tenant information if available
	if tenantID, exists := c.Get("tenant_id"); exists {
		logData["tenant_id"] = tenantID
	}

	// Log with appropriate level based on severity
	switch err.Severity {
	case SeverityLow:
		h.logger.Info("Request error", logData)
	case SeverityMedium:
		h.logger.Warn("Request error", logData)
	case SeverityHigh, SeverityCritical:
		h.logger.Error("Request error", logData)
	default:
		h.logger.Error("Request error", logData)
	}
}

// sanitizeErrorForProduction removes sensitive information from errors in production
func (h *ErrorHandler) sanitizeErrorForProduction(err *WebhookError) {
	// Remove stack traces and internal details in production
	if err.Details != nil {
		delete(err.Details, "stack_trace")
		delete(err.Details, "panic_value")

		// Sanitize database error details
		if err.Category == CategoryDatabase {
			delete(err.Details, "query")
			delete(err.Details, "connection_string")
		}
	}

	// Generic message for internal errors in production
	if err.Category == CategoryInternal && err.Code != "PANIC_RECOVERED" {
		err.Message = "An internal server error occurred"
		err.Details = nil
	}
}

// setErrorHeaders sets appropriate HTTP headers for the error response
func (h *ErrorHandler) setErrorHeaders(c *gin.Context, err *WebhookError) {
	// Set content type
	c.Header("Content-Type", "application/json")

	// Set request ID header for tracking
	if err.RequestID != "" {
		c.Header("X-Request-ID", err.RequestID)
	}

	// Set trace ID header for distributed tracing
	if err.TraceID != "" {
		c.Header("X-Trace-ID", err.TraceID)
	}

	// Set retry-after header for rate limit errors
	if err.Category == CategoryRateLimit {
		if retryAfter, ok := err.Details["retry_after_seconds"].(int); ok {
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
		} else {
			c.Header("Retry-After", "60") // Default 1 minute
		}
	}

	// Set cache control for error responses
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
}

// getRequestID gets or generates a request ID
func (h *ErrorHandler) getRequestID(c *gin.Context) string {
	// Try to get existing request ID from context
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}

	// Try to get from header
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
		return requestID
	}

	// Generate new request ID
	return GenerateRequestID()
}

// getTraceID gets or generates a trace ID
func (h *ErrorHandler) getTraceID(c *gin.Context) string {
	// Try to get existing trace ID from context
	if traceID, exists := c.Get("trace_id"); exists {
		if id, ok := traceID.(string); ok {
			return id
		}
	}

	// Try to get from header
	if traceID := c.GetHeader("X-Trace-ID"); traceID != "" {
		return traceID
	}

	// Generate new trace ID
	return GenerateTraceID()
}

// RequestIDMiddleware adds request ID to the context
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = GenerateRequestID()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// TraceIDMiddleware adds trace ID to the context
func TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = GenerateTraceID()
		}

		c.Set("trace_id", traceID)
		c.Header("X-Trace-ID", traceID)
		c.Next()
	}
}
