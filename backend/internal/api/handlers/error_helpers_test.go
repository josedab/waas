package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	webhookerrors "github.com/josedab/waas/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRespondWithError(t *testing.T) {
	tests := []struct {
		name           string
		err            *webhookerrors.WebhookError
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "bad request error",
			err:            webhookerrors.NewWebhookError("INVALID_INPUT", "bad input", webhookerrors.CategoryValidation, http.StatusBadRequest, webhookerrors.SeverityLow),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_INPUT",
		},
		{
			name:           "not found error",
			err:            webhookerrors.NewWebhookError("NOT_FOUND", "resource not found", webhookerrors.CategoryNotFound, http.StatusNotFound, webhookerrors.SeverityLow),
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "unauthorized error",
			err:            webhookerrors.NewWebhookError("UNAUTHORIZED", "not authenticated", webhookerrors.CategoryAuthentication, http.StatusUnauthorized, webhookerrors.SeverityMedium),
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
		},
		{
			name:           "internal server error",
			err:            webhookerrors.NewWebhookError("INTERNAL", "something broke", webhookerrors.CategoryInternal, http.StatusInternalServerError, webhookerrors.SeverityHigh),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL",
		},
		{
			name:           "rate limit error",
			err:            webhookerrors.NewWebhookError("RATE_LIMITED", "too many requests", webhookerrors.CategoryRateLimit, http.StatusTooManyRequests, webhookerrors.SeverityMedium),
			expectedStatus: http.StatusTooManyRequests,
			expectedCode:   "RATE_LIMITED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			RespondWithError(c, tt.err)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp webhookerrors.WebhookError
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.Code)
			assert.NotEmpty(t, resp.Message)
		})
	}
}

func TestInternalError(t *testing.T) {
	t.Run("returns 500 with correlation ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		InternalError(c, "DB_FAILURE", errors.New("connection refused"))

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var resp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "DB_FAILURE", resp.Code)
		assert.Contains(t, resp.Message, "Correlation ID:")
	})

	t.Run("does not leak internal error details", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		InternalError(c, "SECRET_FAILURE", errors.New("password=hunter2 host=db.internal:5432"))

		body := w.Body.String()
		assert.NotContains(t, body, "hunter2")
		assert.NotContains(t, body, "db.internal")
		assert.Contains(t, body, "Correlation ID:")
	})

	t.Run("each call generates unique correlation ID", func(t *testing.T) {
		w1 := httptest.NewRecorder()
		c1, _ := gin.CreateTestContext(w1)
		InternalError(c1, "ERR", errors.New("err1"))

		w2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(w2)
		InternalError(c2, "ERR", errors.New("err2"))

		var r1, r2 ErrorResponse
		require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &r1))
		require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &r2))
		assert.NotEqual(t, r1.Message, r2.Message, "correlation IDs should differ")
	})
}

func TestInternalErrorGeneric(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	InternalErrorGeneric(c, errors.New("sensitive db error"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INTERNAL_ERROR", resp.Code)
	assert.Contains(t, resp.Message, "Correlation ID:")
	assert.NotContains(t, resp.Message, "sensitive db error")
}

func TestRequireTenantID(t *testing.T) {
	tests := []struct {
		name           string
		setupCtx       func(c *gin.Context)
		expectOK       bool
		expectedStatus int
	}{
		{
			name: "valid tenant UUID",
			setupCtx: func(c *gin.Context) {
				c.Set("tenant_id", uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))
			},
			expectOK: true,
		},
		{
			name:           "missing tenant_id in context",
			setupCtx:       func(c *gin.Context) {},
			expectOK:       false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "tenant_id is wrong type (string instead of UUID)",
			setupCtx: func(c *gin.Context) {
				c.Set("tenant_id", "not-a-uuid-object")
			},
			expectOK:       false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "tenant_id is nil",
			setupCtx: func(c *gin.Context) {
				c.Set("tenant_id", nil)
			},
			expectOK:       false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupCtx(c)

			tid, ok := RequireTenantID(c)

			assert.Equal(t, tt.expectOK, ok)
			if tt.expectOK {
				assert.NotEqual(t, uuid.Nil, tid)
			} else {
				assert.Equal(t, uuid.Nil, tid)
				assert.Equal(t, tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestParseQueryInt(t *testing.T) {
	tests := []struct {
		name       string
		queryKey   string
		queryValue string
		defaultVal int
		expected   int
	}{
		{
			name:       "valid integer",
			queryKey:   "limit",
			queryValue: "50",
			defaultVal: 10,
			expected:   50,
		},
		{
			name:       "empty string uses default",
			queryKey:   "limit",
			queryValue: "",
			defaultVal: 10,
			expected:   10,
		},
		{
			name:       "non-numeric input uses default",
			queryKey:   "limit",
			queryValue: "abc",
			defaultVal: 10,
			expected:   10,
		},
		{
			name:       "negative value is valid",
			queryKey:   "offset",
			queryValue: "-5",
			defaultVal: 0,
			expected:   -5,
		},
		{
			name:       "zero value",
			queryKey:   "page",
			queryValue: "0",
			defaultVal: 1,
			expected:   0,
		},
		{
			name:       "large number",
			queryKey:   "limit",
			queryValue: "999999",
			defaultVal: 10,
			expected:   999999,
		},
		{
			name:       "float string uses default",
			queryKey:   "limit",
			queryValue: "3.14",
			defaultVal: 10,
			expected:   10,
		},
		{
			name:       "overflow uses default",
			queryKey:   "limit",
			queryValue: "99999999999999999999",
			defaultVal: 10,
			expected:   10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			if tt.queryValue != "" {
				c.Request, _ = http.NewRequest("GET", "/?"+tt.queryKey+"="+tt.queryValue, nil)
			} else {
				c.Request, _ = http.NewRequest("GET", "/", nil)
			}

			result := ParseQueryInt(c, tt.queryKey, tt.defaultVal)
			assert.Equal(t, tt.expected, result)
		})
	}
}
