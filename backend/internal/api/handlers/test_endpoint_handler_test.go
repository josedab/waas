package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func setupTestEndpointTest() (*TestEndpointHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	handler := NewTestEndpointHandler(logger)
	router := gin.New()
	return handler, router
}

func TestTestEndpointHandler_ReceiveTestWebhook_InvalidEndpointID(t *testing.T) {
	handler, router := setupTestEndpointTest()
	router.POST("/test-endpoints/:endpoint_id", handler.ReceiveTestWebhook)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{"event": "test.event"})
	req, _ := http.NewRequest("POST", "/test-endpoints/not-a-uuid", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTestEndpointHandler_GetTestEndpointReceives_Success(t *testing.T) {
	handler, router := setupTestEndpointTest()
	router.GET("/test-endpoints/:endpoint_id/receives", handler.GetTestEndpointReceives)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test-endpoints/00000000-0000-0000-0000-000000000001/receives", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTestEndpointHandler_GetTestEndpointReceive_InvalidReceiveID(t *testing.T) {
	handler, router := setupTestEndpointTest()
	router.GET("/test-endpoints/:endpoint_id/receives/:receive_id", handler.GetTestEndpointReceive)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test-endpoints/00000000-0000-0000-0000-000000000001/receives/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTestEndpointHandler_ClearTestEndpointReceives_Success(t *testing.T) {
	handler, router := setupTestEndpointTest()
	router.DELETE("/test-endpoints/:endpoint_id/receives", handler.ClearTestEndpointReceives)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/test-endpoints/00000000-0000-0000-0000-000000000001/receives", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
