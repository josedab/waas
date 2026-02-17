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

func setupBidirectionalSyncTest() (*BidirectionalSyncHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	handler := NewBidirectionalSyncHandler(nil, logger)
	router := gin.New()
	return handler, router
}

func TestBidirectionalSyncHandler_CreateConfig_Unauthorized(t *testing.T) {
	handler, router := setupBidirectionalSyncTest()
	router.POST("/configs", handler.CreateConfig)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{})
	req, _ := http.NewRequest("POST", "/configs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBidirectionalSyncHandler_GetConfigs_Unauthorized(t *testing.T) {
	handler, router := setupBidirectionalSyncTest()
	router.GET("/configs", handler.GetConfigs)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/configs", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBidirectionalSyncHandler_GetConfig_Unauthorized(t *testing.T) {
	handler, router := setupBidirectionalSyncTest()
	router.GET("/configs/:id", handler.GetConfig)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/configs/test-id", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBidirectionalSyncHandler_SendSyncRequest_Unauthorized(t *testing.T) {
	handler, router := setupBidirectionalSyncTest()
	router.POST("/sync/request", handler.SendSyncRequest)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{})
	req, _ := http.NewRequest("POST", "/sync/request", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBidirectionalSyncHandler_GetTransaction_Unauthorized(t *testing.T) {
	handler, router := setupBidirectionalSyncTest()
	router.GET("/transactions/:id", handler.GetTransaction)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/transactions/test-id", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBidirectionalSyncHandler_GetConflicts_Unauthorized(t *testing.T) {
	handler, router := setupBidirectionalSyncTest()
	router.GET("/conflicts", handler.GetConflicts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/conflicts", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBidirectionalSyncHandler_GetDashboard_Unauthorized(t *testing.T) {
	handler, router := setupBidirectionalSyncTest()
	router.GET("/dashboard", handler.GetDashboard)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dashboard", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
