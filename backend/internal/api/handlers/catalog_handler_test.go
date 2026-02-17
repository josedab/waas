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

func setupCatalogTest() (*CatalogHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	handler := NewCatalogHandler(nil, logger)
	router := gin.New()
	return handler, router
}

func TestCatalogHandler_CreateEventType_Unauthorized(t *testing.T) {
	handler, router := setupCatalogTest()
	router.POST("/event-types", handler.CreateEventType)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{"name": "test"})
	req, _ := http.NewRequest("POST", "/event-types", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCatalogHandler_GetEventType_InvalidID(t *testing.T) {
	handler, router := setupCatalogTest()
	router.GET("/event-types/:id", handler.GetEventType)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/event-types/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCatalogHandler_SearchEventTypes_Unauthorized(t *testing.T) {
	handler, router := setupCatalogTest()
	router.GET("/event-types", handler.SearchEventTypes)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/event-types?q=test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCatalogHandler_DeleteEventType_InvalidID(t *testing.T) {
	handler, router := setupCatalogTest()
	router.DELETE("/event-types/:id", handler.DeleteEventType)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/event-types/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCatalogHandler_ListCategories_Unauthorized(t *testing.T) {
	handler, router := setupCatalogTest()
	router.GET("/categories", handler.ListCategories)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/categories", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCatalogHandler_PublishVersion_InvalidID(t *testing.T) {
	handler, router := setupCatalogTest()
	router.POST("/event-types/:id/versions", handler.PublishVersion)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{})
	req, _ := http.NewRequest("POST", "/event-types/not-a-uuid/versions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
