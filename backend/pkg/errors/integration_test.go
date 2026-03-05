package errors

import (
	"bytes"
	"encoding/json"
	stderrors "errors"
	"github.com/josedab/waas/pkg/utils"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorHandlingSystemIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up the complete error handling system
	logger := utils.NewLogger("test")
	alerter := &NoOpAlerter{} // Use no-op alerter for testing
	errorHandler := NewErrorHandler(logger, alerter)

	// Create router with error handling middleware
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.Use(TraceIDMiddleware())
	router.Use(errorHandler.Middleware())

	// Create example handler
	exampleHandler := NewExampleHandler(logger, errorHandler)

	// Set up routes
	router.POST("/example", exampleHandler.ExampleEndpoint)
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})
	router.GET("/validation-error", func(c *gin.Context) {
		AbortWithValidationError(c, "test_field", "test validation error")
	})
	router.GET("/rate-limit", func(c *gin.Context) {
		AbortWithRateLimit(c, 60)
	})
	router.GET("/quota-exceeded", func(c *gin.Context) {
		AbortWithQuotaExceeded(c, 1500, 1000)
	})

	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
		expectedCode   string
		checkHeaders   bool
	}{
		{
			name:           "successful request",
			method:         "POST",
			path:           "/example",
			body:           map[string]string{"email": "test@example.com", "name": "Test User"},
			expectedStatus: http.StatusOK,
			expectedCode:   "",
		},
		{
			name:           "validation error - missing fields",
			method:         "POST",
			path:           "/example",
			body:           map[string]string{},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_REQUEST",
		},
		{
			name:           "validation error - invalid email",
			method:         "POST",
			path:           "/example",
			body:           map[string]string{"email": "invalid-email", "name": "Test User"},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_REQUEST",
		},
		{
			name:           "database error - user not found",
			method:         "POST",
			path:           "/example",
			body:           map[string]string{"email": "notfound@example.com", "name": "Test User"},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "USER_NOT_FOUND",
		},
		{
			name:           "database error - user exists",
			method:         "POST",
			path:           "/example",
			body:           map[string]string{"email": "duplicate@example.com", "name": "Test User"},
			expectedStatus: http.StatusConflict,
			expectedCode:   "USER_EXISTS",
		},
		{
			name:           "database error - connection failed",
			method:         "POST",
			path:           "/example",
			body:           map[string]string{"email": "dberror@example.com", "name": "Test User"},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "DATABASE_ERROR",
		},
		{
			name:           "forbidden error",
			method:         "POST",
			path:           "/example",
			body:           map[string]string{"email": "forbidden@example.com", "name": "Test User"},
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
		{
			name:           "external API timeout",
			method:         "POST",
			path:           "/example",
			body:           map[string]string{"email": "timeout@example.com", "name": "Test User"},
			expectedStatus: http.StatusGatewayTimeout,
			expectedCode:   "API_TIMEOUT",
		},
		{
			name:           "external API unreachable",
			method:         "POST",
			path:           "/example",
			body:           map[string]string{"email": "unreachable@example.com", "name": "Test User"},
			expectedStatus: http.StatusBadGateway,
			expectedCode:   "API_UNREACHABLE",
		},
		{
			name:           "panic recovery",
			method:         "GET",
			path:           "/panic",
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "PANIC_RECOVERED",
		},
		{
			name:           "validation error helper",
			method:         "GET",
			path:           "/validation-error",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_REQUEST",
		},
		{
			name:           "rate limit error",
			method:         "GET",
			path:           "/rate-limit",
			expectedStatus: http.StatusTooManyRequests,
			expectedCode:   "RATE_LIMIT_EXCEEDED",
			checkHeaders:   true,
		},
		{
			name:           "quota exceeded error",
			method:         "GET",
			path:           "/quota-exceeded",
			expectedStatus: http.StatusPaymentRequired,
			expectedCode:   "QUOTA_EXCEEDED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request

			if tt.body != nil {
				bodyBytes, err := json.Marshal(tt.body)
				require.NoError(t, err)
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check that request ID and trace ID headers are set
			assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
			assert.NotEmpty(t, w.Header().Get("X-Trace-ID"))

			if tt.expectedCode != "" {
				// Parse error response
				var response ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tt.expectedCode, response.Error.Code)
				assert.NotEmpty(t, response.Error.Message)
				assert.NotEmpty(t, response.Error.Category)
				assert.NotEmpty(t, response.Error.Severity)
				assert.NotEmpty(t, response.Error.RequestID)
				assert.NotEmpty(t, response.Error.TraceID)
				assert.NotZero(t, response.Error.Timestamp)

				// Check debugging hints are present
				if len(response.Error.DebuggingHints) > 0 {
					assert.NotEmpty(t, response.Error.DebuggingHints[0])
				}

				// Check specific error details
				switch tt.expectedCode {
				case "INVALID_REQUEST":
					if response.Error.Details != nil {
						// Should have field and reason for validation errors
						if field, ok := response.Error.Details["field"]; ok {
							assert.NotEmpty(t, field)
						}
					}
				case "RATE_LIMIT_EXCEEDED":
					assert.NotNil(t, response.Error.Details)
					assert.Contains(t, response.Error.Details, "retry_after_seconds")
					if tt.checkHeaders {
						assert.Equal(t, "60", w.Header().Get("Retry-After"))
					}
				case "QUOTA_EXCEEDED":
					assert.NotNil(t, response.Error.Details)
					assert.Contains(t, response.Error.Details, "current_usage")
					assert.Contains(t, response.Error.Details, "monthly_limit")
				case "PANIC_RECOVERED":
					assert.NotNil(t, response.Error.Details)
					assert.Contains(t, response.Error.Details, "panic_value")
				}
			}
		})
	}
}

func TestErrorHandlingMiddlewareOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := utils.NewLogger("test")
	errorHandler := NewErrorHandler(logger, &NoOpAlerter{})

	router := gin.New()

	// Test that middleware sets up context correctly
	router.Use(RequestIDMiddleware())
	router.Use(TraceIDMiddleware())
	router.Use(func(c *gin.Context) {
		// Verify that request ID and trace ID are set
		requestID, exists := c.Get("request_id")
		assert.True(t, exists)
		assert.NotEmpty(t, requestID)

		traceID, exists := c.Get("trace_id")
		assert.True(t, exists)
		assert.NotEmpty(t, traceID)

		c.Next()
	})
	router.Use(errorHandler.Middleware())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	assert.NotEmpty(t, w.Header().Get("X-Trace-ID"))
}

func TestErrorHandlingWithCustomHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := utils.NewLogger("test")
	errorHandler := NewErrorHandler(logger, &NoOpAlerter{})

	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.Use(TraceIDMiddleware())
	router.Use(errorHandler.Middleware())

	router.GET("/test", func(c *gin.Context) {
		AbortWithValidationError(c, "test_field", "test error")
	})

	// Test with custom request ID and trace ID headers
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "custom_req_123")
	req.Header.Set("X-Trace-ID", "custom_trace_456")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "custom_req_123", w.Header().Get("X-Request-ID"))
	assert.Equal(t, "custom_trace_456", w.Header().Get("X-Trace-ID"))

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "custom_req_123", response.Error.RequestID)
	assert.Equal(t, "custom_trace_456", response.Error.TraceID)
}

func TestErrorHandlingProductionMode(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(gin.TestMode)

	logger := utils.NewLogger("test")
	errorHandler := NewErrorHandler(logger, &NoOpAlerter{})

	router := gin.New()
	router.Use(errorHandler.Middleware())

	router.GET("/internal-error", func(c *gin.Context) {
		err := &WebhookError{
			Code:     "INTERNAL_ERROR",
			Message:  "Detailed internal error with sensitive info",
			Category: CategoryInternal,
			Severity: SeverityHigh,
			Details: map[string]interface{}{
				"stack_trace":       "sensitive stack trace",
				"database_password": "secret123",
			},
		}
		AbortWithError(c, err)
	})

	req := httptest.NewRequest("GET", "/internal-error", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// In production mode, internal errors should be sanitized
	assert.Equal(t, "An internal server error occurred", response.Error.Message)
	assert.Nil(t, response.Error.Details) // Sensitive details should be removed
}

func TestErrorHandlingHelperFunctions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupHandler   func() gin.HandlerFunc
		expectedStatus int
		expectedCode   string
	}{
		{
			name: "AbortWithUnauthorized",
			setupHandler: func() gin.HandlerFunc {
				return func(c *gin.Context) {
					AbortWithUnauthorized(c)
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
		},
		{
			name: "AbortWithForbidden",
			setupHandler: func() gin.HandlerFunc {
				return func(c *gin.Context) {
					AbortWithForbidden(c)
				}
			},
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
		{
			name: "AbortWithNotFound",
			setupHandler: func() gin.HandlerFunc {
				return func(c *gin.Context) {
					AbortWithNotFound(c, "test resource")
				}
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name: "AbortWithInternalError",
			setupHandler: func() gin.HandlerFunc {
				return func(c *gin.Context) {
					AbortWithInternalError(c, assert.AnError)
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_SERVER_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", tt.setupHandler())

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedCode, response.Error.Code)
		})
	}
}

func TestErrorHandlingRepositoryIntegration(t *testing.T) {
	logger := utils.NewLogger("test")
	repo := &ExampleRepository{logger: logger}

	tests := []struct {
		name           string
		userID         string
		expectError    bool
		expectedCode   string
		expectedStatus int
	}{
		{
			name:        "successful get user",
			userID:      "valid_user",
			expectError: false,
		},
		{
			name:           "user not found",
			userID:         "not_found",
			expectError:    true,
			expectedCode:   "USER_NOT_FOUND",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "database error",
			userID:         "db_error",
			expectError:    true,
			expectedCode:   "DATABASE_ERROR",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.GetUser(tt.userID)

			if tt.expectError {
				assert.Error(t, err)

				var webhookErr *WebhookError
				if stderrors.As(err, &webhookErr) {
					assert.Equal(t, tt.expectedCode, webhookErr.Code)
					assert.Equal(t, tt.expectedStatus, webhookErr.GetHTTPStatus())
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.userID, user.ID)
			}
		})
	}
}

func TestErrorHandlingDeliveryEngineIntegration(t *testing.T) {
	tests := []struct {
		name         string
		endpointURL  string
		expectError  bool
		expectedCode string
	}{
		{
			name:        "successful delivery",
			endpointURL: "https://example.com/webhook",
			expectError: false,
		},
		{
			name:         "delivery failure",
			endpointURL:  "https://example.com/webhook",
			expectError:  true,
			expectedCode: "WEBHOOK_SERVER_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExampleErrorHandlingInDeliveryEngine(tt.endpointURL, []byte("test payload"))

			if tt.expectError {
				assert.Error(t, err)

				var webhookErr *WebhookError
				if stderrors.As(err, &webhookErr) {
					assert.Equal(t, tt.expectedCode, webhookErr.Code)
					assert.Contains(t, webhookErr.Details, "endpoint_url")
					assert.Contains(t, webhookErr.Details, "http_status")
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
