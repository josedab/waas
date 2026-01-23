package errors

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbortWithError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("request_id", "req_123")
	c.Set("trace_id", "trace_456")
	
	err := ErrInvalidRequest
	AbortWithError(c, err)
	
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response ErrorResponse
	jsonErr := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, jsonErr)
	
	assert.Equal(t, "INVALID_REQUEST", response.Error.Code)
	assert.Equal(t, "req_123", response.Error.RequestID)
	assert.Equal(t, "trace_456", response.Error.TraceID)
}

func TestAbortWithValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	
	field := "email"
	reason := "invalid format"
	
	AbortWithValidationError(c, field, reason)
	
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "INVALID_REQUEST", response.Error.Code)
	assert.Equal(t, field, response.Error.Details["field"])
	assert.Equal(t, reason, response.Error.Details["reason"])
}

func TestAbortWithUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	
	AbortWithUnauthorized(c)
	
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "UNAUTHORIZED", response.Error.Code)
}

func TestAbortWithForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	
	AbortWithForbidden(c)
	
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "FORBIDDEN", response.Error.Code)
}

func TestAbortWithNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	
	resource := "webhook endpoint"
	
	AbortWithNotFound(c, resource)
	
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusNotFound, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "NOT_FOUND", response.Error.Code)
	assert.Contains(t, response.Error.Message, resource)
}

func TestAbortWithInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	
	cause := errors.New("underlying error")
	
	AbortWithInternalError(c, cause)
	
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "INTERNAL_SERVER_ERROR", response.Error.Code)
}

func TestAbortWithDatabaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	
	cause := errors.New("database connection failed")
	
	AbortWithDatabaseError(c, cause)
	
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "DATABASE_ERROR", response.Error.Code)
}

func TestAbortWithQueueError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	
	cause := errors.New("queue operation failed")
	
	AbortWithQueueError(c, cause)
	
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "QUEUE_ERROR", response.Error.Code)
}

func TestAbortWithRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	
	retryAfter := 120
	
	AbortWithRateLimit(c, retryAfter)
	
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", response.Error.Code)
	assert.Equal(t, float64(retryAfter), response.Error.Details["retry_after_seconds"])
}

func TestAbortWithQuotaExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	
	currentUsage := 1500
	limit := 1000
	
	AbortWithQuotaExceeded(c, currentUsage, limit)
	
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusPaymentRequired, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "QUOTA_EXCEEDED", response.Error.Code)
	assert.Equal(t, float64(currentUsage), response.Error.Details["current_usage"])
	assert.Equal(t, float64(limit), response.Error.Details["monthly_limit"])
}

func TestAbortWithPayloadTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	
	actualSize := 2048
	maxSize := 1024
	
	AbortWithPayloadTooLarge(c, actualSize, maxSize)
	
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "PAYLOAD_TOO_LARGE", response.Error.Code)
	assert.Equal(t, float64(actualSize), response.Error.Details["actual_size_bytes"])
	assert.Equal(t, float64(maxSize), response.Error.Details["max_size_bytes"])
}

func TestHandleRepositoryError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		err            error
		expectedCode   string
		expectedStatus int
		expectedCategory ErrorCategory
	}{
		{
			name:           "nil error returns nil",
			err:            nil,
			expectedCode:   "",
			expectedStatus: 0,
			expectedCategory: "",
		},
		{
			name:           "not found error",
			err:            errors.New("record not found"),
			expectedCode:   "RESOURCE_NOT_FOUND",
			expectedStatus: http.StatusNotFound,
			expectedCategory: CategoryNotFound,
		},
		{
			name:           "no rows error",
			err:            errors.New("sql: no rows in result set"),
			expectedCode:   "RESOURCE_NOT_FOUND",
			expectedStatus: http.StatusNotFound,
			expectedCategory: CategoryNotFound,
		},
		{
			name:           "duplicate key error",
			err:            errors.New("duplicate key value violates unique constraint"),
			expectedCode:   "RESOURCE_ALREADY_EXISTS",
			expectedStatus: http.StatusConflict,
			expectedCategory: CategoryValidation,
		},
		{
			name:           "unique constraint error",
			err:            errors.New("unique constraint violation"),
			expectedCode:   "RESOURCE_ALREADY_EXISTS",
			expectedStatus: http.StatusConflict,
			expectedCategory: CategoryValidation,
		},
		{
			name:           "foreign key constraint error",
			err:            errors.New("foreign key constraint violation"),
			expectedCode:   "INVALID_REFERENCE",
			expectedStatus: http.StatusBadRequest,
			expectedCategory: CategoryValidation,
		},
		{
			name:           "connection error",
			err:            errors.New("connection refused"),
			expectedCode:   "DATABASE_ERROR",
			expectedStatus: http.StatusInternalServerError,
			expectedCategory: CategoryDatabase,
		},
		{
			name:           "timeout error",
			err:            errors.New("connection timeout"),
			expectedCode:   "DATABASE_ERROR",
			expectedStatus: http.StatusInternalServerError,
			expectedCategory: CategoryDatabase,
		},
		{
			name:           "generic database error",
			err:            errors.New("some database error"),
			expectedCode:   "DATABASE_ERROR",
			expectedStatus: http.StatusInternalServerError,
			expectedCategory: CategoryDatabase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HandleRepositoryError(tt.err)
			
			if tt.err == nil {
				assert.Nil(t, result)
				return
			}
			
			assert.Equal(t, tt.expectedCode, result.Code)
			assert.Equal(t, tt.expectedStatus, result.GetHTTPStatus())
			assert.Equal(t, tt.expectedCategory, result.Category)
			assert.Equal(t, tt.err, result.Cause)
		})
	}
}

func TestHandleValidationError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		err   error
		field string
	}{
		{
			name:  "nil error returns nil",
			err:   nil,
			field: "test_field",
		},
		{
			name:  "validation error with field",
			err:   errors.New("validation failed"),
			field: "email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HandleValidationError(tt.err, tt.field)
			
			if tt.err == nil {
				assert.Nil(t, result)
				return
			}
			
			assert.Equal(t, "INVALID_REQUEST", result.Code)
			assert.Equal(t, CategoryValidation, result.Category)
			assert.Equal(t, SeverityLow, result.Severity)
			assert.Equal(t, tt.field, result.Details["field"])
			assert.Equal(t, tt.err.Error(), result.Details["reason"])
			assert.Equal(t, tt.err, result.Cause)
		})
	}
}

func TestHandleHTTPError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		endpointURL    string
		expectedCode   string
		expectedStatus int
		expectedCategory ErrorCategory
	}{
		{
			name:           "client error 400",
			statusCode:     400,
			responseBody:   "Bad Request",
			endpointURL:    "https://example.com/webhook",
			expectedCode:   "WEBHOOK_CLIENT_ERROR",
			expectedStatus: http.StatusBadGateway,
			expectedCategory: CategoryDeliveryFailed,
		},
		{
			name:           "client error 404",
			statusCode:     404,
			responseBody:   "Not Found",
			endpointURL:    "https://example.com/webhook",
			expectedCode:   "WEBHOOK_CLIENT_ERROR",
			expectedStatus: http.StatusBadGateway,
			expectedCategory: CategoryDeliveryFailed,
		},
		{
			name:           "server error 500",
			statusCode:     500,
			responseBody:   "Internal Server Error",
			endpointURL:    "https://example.com/webhook",
			expectedCode:   "WEBHOOK_SERVER_ERROR",
			expectedStatus: http.StatusBadGateway,
			expectedCategory: CategoryDeliveryFailed,
		},
		{
			name:           "server error 503",
			statusCode:     503,
			responseBody:   "Service Unavailable",
			endpointURL:    "https://example.com/webhook",
			expectedCode:   "WEBHOOK_SERVER_ERROR",
			expectedStatus: http.StatusBadGateway,
			expectedCategory: CategoryDeliveryFailed,
		},
		{
			name:           "unexpected status code",
			statusCode:     200, // Success code, but treated as unexpected in this context
			responseBody:   "OK",
			endpointURL:    "https://example.com/webhook",
			expectedCode:   "WEBHOOK_UNEXPECTED_RESPONSE",
			expectedStatus: http.StatusBadGateway,
			expectedCategory: CategoryDeliveryFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HandleHTTPError(tt.statusCode, tt.responseBody, tt.endpointURL)
			
			assert.Equal(t, tt.expectedCode, result.Code)
			assert.Equal(t, tt.expectedStatus, result.GetHTTPStatus())
			assert.Equal(t, tt.expectedCategory, result.Category)
			assert.Equal(t, tt.statusCode, result.Details["http_status"])
			assert.Equal(t, tt.endpointURL, result.Details["endpoint_url"])
			assert.Equal(t, tt.responseBody, result.Details["response_body"])
			assert.NotEmpty(t, result.DebuggingHints)
		})
	}
}

func TestHandleHTTPError_TruncatesLongResponseBody(t *testing.T) {
	t.Parallel()
	// Create a response body longer than 1000 characters
	longResponseBody := ""
	for i := 0; i < 1200; i++ {
		longResponseBody += "a"
	}
	
	result := HandleHTTPError(500, longResponseBody, "https://example.com/webhook")
	
	responseBody := result.Details["response_body"].(string)
	assert.True(t, len(responseBody) <= 1015) // 1000 + "... (truncated)"
	assert.Contains(t, responseBody, "... (truncated)")
}

func TestHandleNetworkError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		err            error
		endpointURL    string
		expectedCode   string
		expectedStatus int
		expectedCategory ErrorCategory
	}{
		{
			name:           "nil error returns nil",
			err:            nil,
			endpointURL:    "https://example.com/webhook",
			expectedCode:   "",
			expectedStatus: 0,
			expectedCategory: "",
		},
		{
			name:           "timeout error",
			err:            errors.New("context deadline exceeded"),
			endpointURL:    "https://example.com/webhook",
			expectedCode:   "WEBHOOK_TIMEOUT",
			expectedStatus: http.StatusGatewayTimeout,
			expectedCategory: CategoryTimeout,
		},
		{
			name:           "connection refused error",
			err:            errors.New("connection refused"),
			endpointURL:    "https://example.com/webhook",
			expectedCode:   "WEBHOOK_UNREACHABLE",
			expectedStatus: http.StatusBadGateway,
			expectedCategory: CategoryExternalAPI,
		},
		{
			name:           "no route to host error",
			err:            errors.New("no route to host"),
			endpointURL:    "https://example.com/webhook",
			expectedCode:   "WEBHOOK_UNREACHABLE",
			expectedStatus: http.StatusBadGateway,
			expectedCategory: CategoryExternalAPI,
		},
		{
			name:           "TLS error",
			err:            errors.New("tls: certificate verification failed"),
			endpointURL:    "https://example.com/webhook",
			expectedCode:   "WEBHOOK_TLS_ERROR",
			expectedStatus: http.StatusBadGateway,
			expectedCategory: CategoryExternalAPI,
		},
		{
			name:           "certificate error",
			err:            errors.New("x509: certificate has expired"),
			endpointURL:    "https://example.com/webhook",
			expectedCode:   "WEBHOOK_TLS_ERROR",
			expectedStatus: http.StatusBadGateway,
			expectedCategory: CategoryExternalAPI,
		},
		{
			name:           "generic network error",
			err:            errors.New("network is unreachable"),
			endpointURL:    "https://example.com/webhook",
			expectedCode:   "WEBHOOK_NETWORK_ERROR",
			expectedStatus: http.StatusBadGateway,
			expectedCategory: CategoryExternalAPI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HandleNetworkError(tt.err, tt.endpointURL)
			
			if tt.err == nil {
				assert.Nil(t, result)
				return
			}
			
			assert.Equal(t, tt.expectedCode, result.Code)
			assert.Equal(t, tt.expectedStatus, result.GetHTTPStatus())
			assert.Equal(t, tt.expectedCategory, result.Category)
			assert.Equal(t, tt.endpointURL, result.Details["endpoint_url"])
			assert.Equal(t, tt.err.Error(), result.Details["network_error"])
			assert.Equal(t, tt.err, result.Cause)
			assert.NotEmpty(t, result.DebuggingHints)
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "webhook error - retryable",
			err:      ErrTimeout,
			expected: true,
		},
		{
			name:     "webhook error - not retryable",
			err:      ErrInvalidRequest,
			expected: false,
		},
		{
			name:     "generic timeout error",
			err:      errors.New("operation timeout"),
			expected: true,
		},
		{
			name:     "generic connection error",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "generic network error",
			err:      errors.New("network unreachable"),
			expected: true,
		},
		{
			name:     "temporary error",
			err:      errors.New("temporary failure"),
			expected: true,
		},
		{
			name:     "unavailable error",
			err:      errors.New("service unavailable"),
			expected: true,
		},
		{
			name:     "deadline exceeded error",
			err:      errors.New("context deadline exceeded"),
			expected: true,
		},
		{
			name:     "non-retryable error",
			err:      errors.New("validation failed"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetErrorSeverity(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      error
		expected ErrorSeverity
	}{
		{
			name:     "webhook error",
			err:      ErrInternalServer,
			expected: SeverityHigh,
		},
		{
			name:     "generic error",
			err:      errors.New("generic error"),
			expected: SeverityMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetErrorSeverity(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetErrorCategory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      error
		expected ErrorCategory
	}{
		{
			name:     "webhook error",
			err:      ErrInvalidRequest,
			expected: CategoryValidation,
		},
		{
			name:     "generic error",
			err:      errors.New("generic error"),
			expected: CategoryInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetErrorCategory(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		err         *WebhookError
		ctx         context.Context
		expectedReq string
		expectedTrace string
	}{
		{
			name: "nil error returns nil",
			err:  nil,
			ctx:  context.Background(),
		},
		{
			name: "adds request ID from context",
			err: &WebhookError{
				Code: "TEST_ERROR",
			},
			ctx:         context.WithValue(context.Background(), "request_id", "req_123"),
			expectedReq: "req_123",
		},
		{
			name: "adds trace ID from context",
			err: &WebhookError{
				Code: "TEST_ERROR",
			},
			ctx:           context.WithValue(context.Background(), "trace_id", "trace_456"),
			expectedTrace: "trace_456",
		},
		{
			name: "doesn't override existing request ID",
			err: &WebhookError{
				Code:      "TEST_ERROR",
				RequestID: "existing_req",
			},
			ctx:         context.WithValue(context.Background(), "request_id", "req_123"),
			expectedReq: "existing_req",
		},
		{
			name: "doesn't override existing trace ID",
			err: &WebhookError{
				Code:    "TEST_ERROR",
				TraceID: "existing_trace",
			},
			ctx:           context.WithValue(context.Background(), "trace_id", "trace_456"),
			expectedTrace: "existing_trace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithContext(tt.err, tt.ctx)
			
			if tt.err == nil {
				assert.Nil(t, result)
				return
			}
			
			assert.Equal(t, tt.err, result) // Should return same instance
			
			if tt.expectedReq != "" {
				assert.Equal(t, tt.expectedReq, result.RequestID)
			}
			
			if tt.expectedTrace != "" {
				assert.Equal(t, tt.expectedTrace, result.TraceID)
			}
		})
	}
}

func TestRespondWithError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("request_id", "req_respond")

	err := ErrEndpointNotFound
	RespondWithError(c, err)

	assert.False(t, c.IsAborted(), "RespondWithError should not abort")
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, "ENDPOINT_NOT_FOUND", response.Error.Code)
	assert.Equal(t, "req_respond", response.Error.RequestID)
}

func TestHandleBindError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)

	bindErr := errors.New("Key: 'CreateEndpointRequest.URL' Error:Field validation for 'URL' failed on the 'required' tag")
	HandleBindError(c, bindErr)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, "INVALID_REQUEST_BODY", response.Error.Code)
	assert.NotEmpty(t, response.Error.DebuggingHints)
}

func TestSanitizeValidationError(t *testing.T) {
	raw := "Key: 'CreateEndpointRequest.URL' Error:Field validation for 'URL' failed on the 'required' tag"
	sanitized := sanitizeValidationError(raw)
	assert.NotContains(t, sanitized, "Key: '")
	assert.NotContains(t, sanitized, "' tag")
}