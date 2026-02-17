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

func setupAIComposerTest() (*AIComposerHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	handler := NewAIComposerHandler(nil, logger)
	router := gin.New()
	return handler, router
}

func TestAIComposerHandler_Compose_Unauthorized(t *testing.T) {
	handler, router := setupAIComposerTest()
	router.POST("/compose", handler.Compose)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{"prompt": "test"})
	req, _ := http.NewRequest("POST", "/compose", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAIComposerHandler_ApplyConfig_Unauthorized(t *testing.T) {
	handler, router := setupAIComposerTest()
	router.POST("/apply", handler.ApplyConfig)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{})
	req, _ := http.NewRequest("POST", "/apply", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAIComposerHandler_SubmitFeedback_InvalidBody(t *testing.T) {
	handler, router := setupAIComposerTest()
	router.POST("/feedback", handler.SubmitFeedback)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/feedback", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAIComposerHandler_GetSessionHistory_Success(t *testing.T) {
	handler, router := setupAIComposerTest()
	router.GET("/history", handler.GetSessionHistory)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/history", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
