package httputil

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestInternalErrorGeneric_Returns500(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)

	InternalErrorGeneric(c, errors.New("something went wrong"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestInternalErrorGeneric_JSONContentType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)

	InternalErrorGeneric(c, errors.New("db connection lost"))

	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestInternalErrorGeneric_ContainsCorrelationID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)

	InternalErrorGeneric(c, errors.New("internal failure"))

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	errMsg, ok := resp["error"].(string)
	require.True(t, ok)
	assert.Contains(t, errMsg, "Correlation ID:")
}

func TestInternalErrorGeneric_DoesNotLeakStackTrace(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)

	InternalErrorGeneric(c, errors.New("pq: relation \"users\" does not exist"))

	body := w.Body.String()
	assert.NotContains(t, body, "pq:")
	assert.NotContains(t, body, "does not exist")
	assert.NotContains(t, body, "users")
}

func TestInternalError_Returns500WithCode(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/webhooks", nil)

	InternalError(c, "WEBHOOK_CREATE_FAILED", errors.New("db error"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestInternalError_ResponseStructure(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/webhooks", nil)

	InternalError(c, "WEBHOOK_CREATE_FAILED", errors.New("connection timeout"))

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	errObj, ok := resp["error"].(map[string]interface{})
	require.True(t, ok, "error should be an object")
	assert.Equal(t, "WEBHOOK_CREATE_FAILED", errObj["code"])

	msg, ok := errObj["message"].(string)
	require.True(t, ok)
	assert.Contains(t, msg, "Correlation ID:")
}

func TestInternalError_DoesNotLeakInternalDetails(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/webhooks", nil)

	InternalError(c, "DB_ERROR", errors.New("connection to 10.0.0.5:5432 refused"))

	body := w.Body.String()
	assert.NotContains(t, body, "10.0.0.5")
	assert.NotContains(t, body, "5432")
	assert.NotContains(t, body, "refused")
}
