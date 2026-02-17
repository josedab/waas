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

func setupFederatedMeshTest() (*FederatedMeshHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	handler := NewFederatedMeshHandler(nil, logger)
	router := gin.New()
	return handler, router
}

func TestFederatedMeshHandler_SetupTenantRegion_Unauthorized(t *testing.T) {
	handler, router := setupFederatedMeshTest()
	router.POST("/tenant-regions", handler.SetupTenantRegion)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{})
	req, _ := http.NewRequest("POST", "/tenant-regions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestFederatedMeshHandler_CreateRoutingRule_Unauthorized(t *testing.T) {
	handler, router := setupFederatedMeshTest()
	router.POST("/routing-rules", handler.CreateRoutingRule)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{})
	req, _ := http.NewRequest("POST", "/routing-rules", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestFederatedMeshHandler_RouteEvent_Unauthorized(t *testing.T) {
	handler, router := setupFederatedMeshTest()
	router.POST("/route", handler.RouteEvent)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{})
	req, _ := http.NewRequest("POST", "/route", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestFederatedMeshHandler_InitiateFailover_InvalidBody(t *testing.T) {
	handler, router := setupFederatedMeshTest()
	router.POST("/failover", handler.InitiateFailover)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/failover", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFederatedMeshHandler_GetDashboard_Unauthorized(t *testing.T) {
	handler, router := setupFederatedMeshTest()
	router.GET("/dashboard", handler.GetDashboard)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dashboard", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
