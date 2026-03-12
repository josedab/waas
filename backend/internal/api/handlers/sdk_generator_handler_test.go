package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func setupSDKGeneratorTest() (*SDKGeneratorHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	handler := NewSDKGeneratorHandler(nil, logger)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", uuid.New())
		c.Next()
	})
	return handler, router
}

func TestSDKGeneratorHandler_CreateConfig_InvalidBody(t *testing.T) {
	handler, router := setupSDKGeneratorTest()
	router.POST("/configs", handler.CreateConfig)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/configs", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSDKGeneratorHandler_GenerateSDK_InvalidBody(t *testing.T) {
	handler, router := setupSDKGeneratorTest()
	router.POST("/generate/:id", handler.GenerateSDK)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/generate/test-id", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSDKGeneratorHandler_GetSupportedLanguages_Success(t *testing.T) {
	handler, router := setupSDKGeneratorTest()
	router.GET("/languages", handler.GetSupportedLanguages)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/languages", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSDKGeneratorHandler_DownloadSDK_InvalidGenerationID(t *testing.T) {
	handler, router := setupSDKGeneratorTest()
	router.GET("/download/:generation_id", handler.DownloadSDK)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/download/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
