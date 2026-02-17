package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func setupMultiRegionTest() (*MultiRegionHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	handler := &MultiRegionHandler{logger: logger}
	router := gin.New()
	return handler, router
}

func TestMultiRegionHandler_CreateRegion_InvalidBody(t *testing.T) {
	handler, router := setupMultiRegionTest()
	router.POST("/regions", handler.CreateRegion)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/regions", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMultiRegionHandler_TriggerFailover_InvalidBody(t *testing.T) {
	handler, router := setupMultiRegionTest()
	router.POST("/failover", handler.TriggerFailover)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/failover", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMultiRegionHandler_GetRoutingPolicy_Unauthorized(t *testing.T) {
	handler, router := setupMultiRegionTest()
	router.GET("/routing-policies/:id", handler.GetRoutingPolicy)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/routing-policies/test-id", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}


