package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/playground"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestPlaygroundHandler creates a PlaygroundHandler backed by an in-memory
// playground.Service (nil repo/engine). Handlers that only use the in-memory
// service features (test suites, shared scenarios) work end-to-end; handlers
// that hit the database (CreateSession, GetSession) are tested for their
// error-path response patterns only.
func newTestPlaygroundHandler() *PlaygroundHandler {
	svc := playground.NewService(nil, nil)
	logger := utils.NewLogger("test")
	return NewPlaygroundHandler(svc, logger)
}

func setupPlaygroundRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// ---------------------------------------------------------------------------
// CreateSession – missing tenant_id → 401
// ---------------------------------------------------------------------------

func TestPlaygroundHandler_CreateSession_MissingTenantID(t *testing.T) {
	handler := newTestPlaygroundHandler()
	router := setupPlaygroundRouter()
	router.POST("/playground/sessions", handler.CreateSession)

	body := `{"name":"my session"}`
	req := httptest.NewRequest(http.MethodPost, "/playground/sessions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "UNAUTHORIZED", resp.Code)
}

// ---------------------------------------------------------------------------
// GetSession – invalid UUID → 400
// ---------------------------------------------------------------------------

func TestPlaygroundHandler_GetSession_InvalidID(t *testing.T) {
	handler := newTestPlaygroundHandler()
	router := setupPlaygroundRouter()
	router.GET("/playground/sessions/:id", handler.GetSession)

	req := httptest.NewRequest(http.MethodGet, "/playground/sessions/not-a-uuid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_ID", resp.Code)
}

// ---------------------------------------------------------------------------
// SaveSession – invalid JSON (missing required "name") → 400
// ---------------------------------------------------------------------------

func TestPlaygroundHandler_SaveSession_InvalidBody(t *testing.T) {
	handler := newTestPlaygroundHandler()
	router := setupPlaygroundRouter()
	router.POST("/playground/sessions/:id/save", handler.SaveSession)

	sessionID := uuid.New().String()
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/playground/sessions/"+sessionID+"/save", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", resp.Code)
}

// ---------------------------------------------------------------------------
// CreateTestSuite – success (201)
// Uses the in-memory service path so no database is needed.
// ---------------------------------------------------------------------------

func TestPlaygroundHandler_CreateTestSuite_Success(t *testing.T) {
	handler := newTestPlaygroundHandler()
	router := setupPlaygroundRouter()

	// Inject tenant_id via middleware (same pattern the real auth middleware uses)
	router.POST("/playground/suites", func(c *gin.Context) {
		c.Set("tenant_id", uuid.New().String())
		c.Next()
	}, handler.CreateTestSuite)

	body := `{"name":"smoke tests","description":"basic suite","is_public":false}`
	req := httptest.NewRequest(http.MethodPost, "/playground/suites", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var suite playground.TestSuite
	err := json.Unmarshal(w.Body.Bytes(), &suite)
	require.NoError(t, err)
	assert.Equal(t, "smoke tests", suite.Name)
	assert.NotEqual(t, uuid.Nil, suite.ID)
}

// ---------------------------------------------------------------------------
// CreateTestSuite – missing tenant_id → 401
// ---------------------------------------------------------------------------

func TestPlaygroundHandler_CreateTestSuite_MissingTenantID(t *testing.T) {
	handler := newTestPlaygroundHandler()
	router := setupPlaygroundRouter()
	router.POST("/playground/suites", handler.CreateTestSuite)

	body := `{"name":"suite"}`
	req := httptest.NewRequest(http.MethodPost, "/playground/suites", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "UNAUTHORIZED", resp.Code)
}

// ---------------------------------------------------------------------------
// CreateTestSuite – invalid JSON (missing required "name") → 400
// ---------------------------------------------------------------------------

func TestPlaygroundHandler_CreateTestSuite_InvalidJSON(t *testing.T) {
	handler := newTestPlaygroundHandler()
	router := setupPlaygroundRouter()
	router.POST("/playground/suites", func(c *gin.Context) {
		c.Set("tenant_id", uuid.New().String())
		c.Next()
	}, handler.CreateTestSuite)

	body := `{"description":"no name field"}`
	req := httptest.NewRequest(http.MethodPost, "/playground/suites", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", resp.Code)
}

// ---------------------------------------------------------------------------
// ListTestSuites – success (200)
// ---------------------------------------------------------------------------

func TestPlaygroundHandler_ListTestSuites_Success(t *testing.T) {
	handler := newTestPlaygroundHandler()
	router := setupPlaygroundRouter()

	tenantID := uuid.New().String()

	// Seed a suite via the create endpoint first.
	router.POST("/playground/suites", func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Next()
	}, handler.CreateTestSuite)

	router.GET("/playground/suites", func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Next()
	}, handler.ListTestSuites)

	// Create a suite
	createBody := `{"name":"suite-a"}`
	createReq := httptest.NewRequest(http.MethodPost, "/playground/suites", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	cw := httptest.NewRecorder()
	router.ServeHTTP(cw, createReq)
	require.Equal(t, http.StatusCreated, cw.Code)

	// List suites
	listReq := httptest.NewRequest(http.MethodGet, "/playground/suites", nil)
	lw := httptest.NewRecorder()
	router.ServeHTTP(lw, listReq)

	assert.Equal(t, http.StatusOK, lw.Code)

	var suites []playground.TestSuite
	err := json.Unmarshal(lw.Body.Bytes(), &suites)
	require.NoError(t, err)
	assert.Len(t, suites, 1)
	assert.Equal(t, "suite-a", suites[0].Name)
}
