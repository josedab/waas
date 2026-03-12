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

func setupGraphQLTest() (*GraphQLHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	handler := NewGraphQLHandler(nil, logger)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", uuid.New())
		c.Next()
	})
	return handler, router
}

func TestGraphQLHandler_CreateSchema_InvalidBody(t *testing.T) {
	handler, router := setupGraphQLTest()
	router.POST("/schemas", handler.CreateSchema)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/schemas", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGraphQLHandler_ParseSchema_InvalidBody(t *testing.T) {
	handler, router := setupGraphQLTest()
	router.POST("/schemas/parse", handler.ParseSchema)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/schemas/parse", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGraphQLHandler_CreateSubscription_InvalidBody(t *testing.T) {
	handler, router := setupGraphQLTest()
	router.POST("/subscriptions", handler.CreateSubscription)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/subscriptions", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGraphQLHandler_IngestEvent_InvalidBody(t *testing.T) {
	handler, router := setupGraphQLTest()
	router.POST("/events", handler.IngestEvent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGraphQLHandler_AddFederationSource_InvalidBody(t *testing.T) {
	handler, router := setupGraphQLTest()
	router.POST("/federation/sources", handler.AddFederationSource)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/federation/sources", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGraphQLHandler_CreateTypeMapping_InvalidBody(t *testing.T) {
	handler, router := setupGraphQLTest()
	router.POST("/type-mappings", handler.CreateTypeMapping)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/type-mappings", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
