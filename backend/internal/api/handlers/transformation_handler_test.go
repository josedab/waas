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

func setupTransformationTest() (*TransformationHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	handler := &TransformationHandler{logger: logger}
	router := gin.New()
	return handler, router
}

func TestTransformationHandler_CreateTransformation_InvalidBody(t *testing.T) {
	handler, router := setupTransformationTest()
	router.POST("/transformations", handler.CreateTransformation)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/transformations", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransformationHandler_GetTransformation_InvalidID(t *testing.T) {
	handler, router := setupTransformationTest()
	router.GET("/transformations/:id", handler.GetTransformation)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/transformations/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransformationHandler_UpdateTransformation_InvalidID(t *testing.T) {
	handler, router := setupTransformationTest()
	router.PUT("/transformations/:id", handler.UpdateTransformation)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/transformations/not-a-uuid", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransformationHandler_UpdateTransformation_InvalidBody(t *testing.T) {
	handler, router := setupTransformationTest()
	router.PUT("/transformations/:id", handler.UpdateTransformation)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/transformations/00000000-0000-0000-0000-000000000001", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransformationHandler_DeleteTransformation_InvalidID(t *testing.T) {
	handler, router := setupTransformationTest()
	router.DELETE("/transformations/:id", handler.DeleteTransformation)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/transformations/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransformationHandler_TestTransformation_InvalidBody(t *testing.T) {
	handler, router := setupTransformationTest()
	router.POST("/transformations/:id/test", handler.TestTransformation)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/transformations/not-a-uuid/test", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransformationHandler_LinkTransformation_InvalidBody(t *testing.T) {
	handler, router := setupTransformationTest()
	router.POST("/transformations/link", handler.LinkTransformation)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/transformations/link", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
