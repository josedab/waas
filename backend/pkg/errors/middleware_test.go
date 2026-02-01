package errors

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/josedab/waas/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAlerter is a mock implementation of AlerterInterface
type MockAlerter struct {
	mock.Mock
}

func (m *MockAlerter) SendAlert(ctx context.Context, err *WebhookError) error {
	args := m.Called(ctx, err)
	return args.Error(0)
}

func TestNewErrorHandler(t *testing.T) {
	logger := utils.NewLogger("test")
	alerter := &MockAlerter{}
	
	handler := NewErrorHandler(logger, alerter)
	
	assert.NotNil(t, handler)
	assert.Equal(t, logger, handler.logger)
	assert.Equal(t, alerter, handler.alerter)
}

func TestErrorHandler_HandleError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
		shouldAlert    bool
	}{
		{
			name:           "webhook error",
			err:            ErrInvalidRequest,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_REQUEST",
			shouldAlert:    false,
		},
		{
			name:           "high severity error should alert",
			err:            ErrInternalServer,
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_SERVER_ERROR",
			shouldAlert:    true,
		},
		{
			name:           "generic error gets categorized",
			err:            errors.New("database connection failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "DATABASE_ERROR",
			shouldAlert:    true,
		},
		{
			name:           "nil error does nothing",
			err:            nil,
			expectedStatus: 0,
			expectedCode:   "",
			shouldAlert:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := utils.NewLogger("test")
			alerter := &MockAlerter{}
			
			if tt.shouldAlert {
				alerter.On("SendAlert", mock.Anything, mock.AnythingOfType("*errors.WebhookError")).Return(nil)
			}
			
			handler := NewErrorHandler(logger, alerter)
			
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)
			c.Set("request_id", "req_123")
			c.Set("trace_id", "trace_456")
			
			handler.HandleError(c, tt.err)
			
			if tt.err == nil {
				assert.Equal(t, 200, w.Code) // Default status when no error
				return
			}
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.True(t, c.IsAborted())
			
			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.Equal(t, tt.expectedCode, response.Error.Code)
			assert.Equal(t, "req_123", response.Error.RequestID)
			assert.Equal(t, "trace_456", response.Error.TraceID)
			
			if tt.shouldAlert {
				alerter.AssertExpectations(t)
			}
		})
	}
}

func TestErrorHandler_categorizeError(t *testing.T) {
	logger := utils.NewLogger("test")
	handler := NewErrorHandler(logger, nil)
	
	tests := []struct {
		name             string
		err              error
		expectedCategory ErrorCategory
		expectedCode     string
	}{
		{
			name:             "database error",
			err:              errors.New("database connection failed"),
			expectedCategory: CategoryDatabase,
			expectedCode:     "DATABASE_ERROR",
		},
		{
			name:             "sql error",
			err:              errors.New("SQL syntax error"),
			expectedCategory: CategoryDatabase,
			expectedCode:     "DATABASE_ERROR",
		},
		{
			name:             "connection error",
			err:              errors.New("connection refused"),
			expectedCategory: CategoryDatabase,
			expectedCode:     "DATABASE_ERROR",
		},
		{
			name:             "queue error",
			err:              errors.New("queue operation failed"),
			expectedCategory: CategoryQueue,
			expectedCode:     "QUEUE_ERROR",
		},
		{
			name:             "redis error",
			err:              errors.New("redis connection failed"),
			expectedCategory: CategoryQueue,
			expectedCode:     "QUEUE_ERROR",
		},
		{
			name:             "rabbitmq error",
			err:              errors.New("rabbitmq publish failed"),
			expectedCategory: CategoryQueue,
			expectedCode:     "QUEUE_ERROR",
		},
		{
			name:             "validation error",
			err:              errors.New("validation failed"),
			expectedCategory: CategoryValidation,
			expectedCode:     "VALIDATION_ERROR",
		},
		{
			name:             "invalid error",
			err:              errors.New("invalid input"),
			expectedCategory: CategoryValidation,
			expectedCode:     "VALIDATION_ERROR",
		},
		{
			name:             "required field error",
			err:              errors.New("required field missing"),
			expectedCategory: CategoryValidation,
			expectedCode:     "VALIDATION_ERROR",
		},
		{
			name:             "timeout error",
			err:              errors.New("operation timeout"),
			expectedCategory: CategoryTimeout,
			expectedCode:     "TIMEOUT",
		},
		{
			name:             "deadline exceeded error",
			err:              errors.New("context deadline exceeded"),
			expectedCategory: CategoryTimeout,
			expectedCode:     "TIMEOUT",
		},
		{
			name:             "network error",
			err:              errors.New("network unreachable"),
			expectedCategory: CategoryExternalAPI,
			expectedCode:     "NETWORK_ERROR",
		},
		{
			name:             "connection refused error",
			err:              errors.New("connection refused"),
			expectedCategory: CategoryExternalAPI,
			expectedCode:     "NETWORK_ERROR",
		},
		{
			name:             "no route to host error",
			err:              errors.New("no route to host"),
			expectedCategory: CategoryExternalAPI,
			expectedCode:     "NETWORK_ERROR",
		},
		{
			name:             "unknown error",
			err:              errors.New("unknown error"),
			expectedCategory: CategoryInternal,
			expectedCode:     "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhookErr := handler.categorizeError(tt.err)
			
			assert.Equal(t, tt.expectedCategory, webhookErr.Category)
			assert.Equal(t, tt.expectedCode, webhookErr.Code)
			assert.Equal(t, tt.err, webhookErr.Cause)
		})
	}
}

func TestErrorHandler_sanitizeErrorForProduction(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(gin.TestMode)
	
	logger := utils.NewLogger("test")
	handler := NewErrorHandler(logger, nil)
	
	tests := []struct {
		name        string
		err         *WebhookError
		expectClean bool
	}{
		{
			name: "internal error gets sanitized",
			err: &WebhookError{
				Code:     "INTERNAL_ERROR",
				Message:  "Detailed internal error",
				Category: CategoryInternal,
				Details: map[string]interface{}{
					"stack_trace": "detailed stack trace",
					"query":       "SELECT * FROM secrets",
				},
			},
			expectClean: true,
		},
		{
			name: "panic error keeps code but removes details",
			err: &WebhookError{
				Code:     "PANIC_RECOVERED",
				Message:  "Panic occurred",
				Category: CategoryInternal,
				Details: map[string]interface{}{
					"stack_trace": "detailed stack trace",
					"panic_value": "panic details",
				},
			},
			expectClean: false, // PANIC_RECOVERED is excluded from sanitization
		},
		{
			name: "database error gets sanitized",
			err: &WebhookError{
				Code:     "DATABASE_ERROR",
				Message:  "Database error",
				Category: CategoryDatabase,
				Details: map[string]interface{}{
					"query":             "SELECT * FROM users",
					"connection_string": "postgres://user:pass@host/db",
					"other_detail":      "kept",
				},
			},
			expectClean: true,
		},
		{
			name: "validation error not sanitized",
			err: &WebhookError{
				Code:     "VALIDATION_ERROR",
				Message:  "Validation failed",
				Category: CategoryValidation,
				Details: map[string]interface{}{
					"field": "email",
					"value": "invalid-email",
				},
			},
			expectClean: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalMessage := tt.err.Message
			originalDetails := make(map[string]interface{})
			if tt.err.Details != nil {
				for k, v := range tt.err.Details {
					originalDetails[k] = v
				}
			}
			
			handler.sanitizeErrorForProduction(tt.err)
			
			if tt.expectClean {
				if tt.err.Category == CategoryInternal && tt.err.Code != "PANIC_RECOVERED" {
					assert.Equal(t, "An internal server error occurred", tt.err.Message)
					assert.Nil(t, tt.err.Details)
				} else if tt.err.Category == CategoryDatabase {
					assert.NotContains(t, tt.err.Details, "query")
					assert.NotContains(t, tt.err.Details, "connection_string")
				}
				
				// Stack traces should always be removed
				if tt.err.Details != nil {
					assert.NotContains(t, tt.err.Details, "stack_trace")
					assert.NotContains(t, tt.err.Details, "panic_value")
				}
			} else {
				if tt.err.Code != "PANIC_RECOVERED" {
					assert.Equal(t, originalMessage, tt.err.Message)
				}
			}
		})
	}
}

func TestErrorHandler_setErrorHeaders(t *testing.T) {
	logger := utils.NewLogger("test")
	handler := NewErrorHandler(logger, nil)
	
	tests := []struct {
		name        string
		err         *WebhookError
		expectedHeaders map[string]string
	}{
		{
			name: "basic error headers",
			err: &WebhookError{
				RequestID: "req_123",
				TraceID:   "trace_456",
				Category:  CategoryValidation,
			},
			expectedHeaders: map[string]string{
				"Content-Type":    "application/json",
				"X-Request-ID":    "req_123",
				"X-Trace-ID":      "trace_456",
				"Cache-Control":   "no-cache, no-store, must-revalidate",
				"Pragma":          "no-cache",
				"Expires":         "0",
			},
		},
		{
			name: "rate limit error with retry-after",
			err: &WebhookError{
				Category: CategoryRateLimit,
				Details: map[string]interface{}{
					"retry_after_seconds": 120,
				},
			},
			expectedHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Retry-After":   "120",
				"Cache-Control": "no-cache, no-store, must-revalidate",
				"Pragma":        "no-cache",
				"Expires":       "0",
			},
		},
		{
			name: "rate limit error without retry-after gets default",
			err: &WebhookError{
				Category: CategoryRateLimit,
			},
			expectedHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Retry-After":   "60",
				"Cache-Control": "no-cache, no-store, must-revalidate",
				"Pragma":        "no-cache",
				"Expires":       "0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			handler.setErrorHeaders(c, tt.err)
			
			for key, expectedValue := range tt.expectedHeaders {
				assert.Equal(t, expectedValue, w.Header().Get(key))
			}
		})
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		existingHeader string
		expectGenerated bool
	}{
		{
			name:           "generates new request ID when none provided",
			existingHeader: "",
			expectGenerated: true,
		},
		{
			name:           "uses existing request ID from header",
			existingHeader: "req_existing_123",
			expectGenerated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.existingHeader != "" {
				req.Header.Set("X-Request-ID", tt.existingHeader)
			}
			c.Request = req
			
			middleware := RequestIDMiddleware()
			middleware(c)
			
			requestID, exists := c.Get("request_id")
			assert.True(t, exists)
			
			if tt.expectGenerated {
				assert.Contains(t, requestID.(string), "req_")
				assert.NotEqual(t, tt.existingHeader, requestID.(string))
			} else {
				assert.Equal(t, tt.existingHeader, requestID.(string))
			}
			
			assert.Equal(t, requestID.(string), w.Header().Get("X-Request-ID"))
		})
	}
}

func TestTraceIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		existingHeader string
		expectGenerated bool
	}{
		{
			name:           "generates new trace ID when none provided",
			existingHeader: "",
			expectGenerated: true,
		},
		{
			name:           "uses existing trace ID from header",
			existingHeader: "trace_existing_123",
			expectGenerated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.existingHeader != "" {
				req.Header.Set("X-Trace-ID", tt.existingHeader)
			}
			c.Request = req
			
			middleware := TraceIDMiddleware()
			middleware(c)
			
			traceID, exists := c.Get("trace_id")
			assert.True(t, exists)
			
			if tt.expectGenerated {
				assert.Contains(t, traceID.(string), "trace_")
				assert.NotEqual(t, tt.existingHeader, traceID.(string))
			} else {
				assert.Equal(t, tt.existingHeader, traceID.(string))
			}
			
			assert.Equal(t, traceID.(string), w.Header().Get("X-Trace-ID"))
		})
	}
}

func TestErrorHandler_Middleware_PanicRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	logger := utils.NewLogger("test")
	alerter := &MockAlerter{}
	
	// Expect alert to be sent for panic
	alerter.On("SendAlert", mock.Anything, mock.AnythingOfType("*errors.WebhookError")).Return(nil)
	
	handler := NewErrorHandler(logger, alerter)
	
	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	
	// Add the error handling middleware
	engine.Use(handler.Middleware())
	
	// Add a route that panics
	engine.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})
	
	req := httptest.NewRequest("GET", "/panic", nil)
	c.Request = req
	
	// This should not panic the test, but should be handled by the middleware
	engine.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "PANIC_RECOVERED", response.Error.Code)
	assert.Equal(t, CategoryInternal, response.Error.Category)
	assert.Equal(t, SeverityCritical, response.Error.Severity)
	assert.Contains(t, response.Error.Details, "panic_value")
	assert.Contains(t, response.Error.Details, "stack_trace")
	
	alerter.AssertExpectations(t)
}

func TestErrorHandler_getRequestID(t *testing.T) {
	logger := utils.NewLogger("test")
	handler := NewErrorHandler(logger, nil)
	
	tests := []struct {
		name           string
		contextValue   interface{}
		headerValue    string
		expectGenerated bool
	}{
		{
			name:           "gets from context",
			contextValue:   "req_context_123",
			headerValue:    "",
			expectGenerated: false,
		},
		{
			name:           "gets from header when no context",
			contextValue:   nil,
			headerValue:    "req_header_123",
			expectGenerated: false,
		},
		{
			name:           "generates when neither available",
			contextValue:   nil,
			headerValue:    "",
			expectGenerated: true,
		},
		{
			name:           "ignores invalid context type",
			contextValue:   123, // invalid type
			headerValue:    "req_header_123",
			expectGenerated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.headerValue != "" {
				req.Header.Set("X-Request-ID", tt.headerValue)
			}
			c.Request = req
			
			if tt.contextValue != nil {
				c.Set("request_id", tt.contextValue)
			}
			
			requestID := handler.getRequestID(c)
			
			if tt.expectGenerated {
				assert.Contains(t, requestID, "req_")
			} else if tt.contextValue != nil && tt.name == "gets from context" {
				assert.Equal(t, tt.contextValue.(string), requestID)
			} else if tt.headerValue != "" {
				assert.Equal(t, tt.headerValue, requestID)
			}
		})
	}
}

func TestErrorHandler_getTraceID(t *testing.T) {
	logger := utils.NewLogger("test")
	handler := NewErrorHandler(logger, nil)
	
	tests := []struct {
		name           string
		contextValue   interface{}
		headerValue    string
		expectGenerated bool
	}{
		{
			name:           "gets from context",
			contextValue:   "trace_context_123",
			headerValue:    "",
			expectGenerated: false,
		},
		{
			name:           "gets from header when no context",
			contextValue:   nil,
			headerValue:    "trace_header_123",
			expectGenerated: false,
		},
		{
			name:           "generates when neither available",
			contextValue:   nil,
			headerValue:    "",
			expectGenerated: true,
		},
		{
			name:           "ignores invalid context type",
			contextValue:   123, // invalid type
			headerValue:    "trace_header_123",
			expectGenerated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.headerValue != "" {
				req.Header.Set("X-Trace-ID", tt.headerValue)
			}
			c.Request = req
			
			if tt.contextValue != nil {
				c.Set("trace_id", tt.contextValue)
			}
			
			traceID := handler.getTraceID(c)
			
			if tt.expectGenerated {
				assert.Contains(t, traceID, "trace_")
			} else if tt.contextValue != nil && tt.name == "gets from context" {
				assert.Equal(t, tt.contextValue.(string), traceID)
			} else if tt.headerValue != "" {
				assert.Equal(t, tt.headerValue, traceID)
			}
		})
	}
}