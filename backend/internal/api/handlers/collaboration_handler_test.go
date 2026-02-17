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

func setupCollaborationTest() (*CollaborationHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	handler := NewCollaborationHandler(nil, logger)
	router := gin.New()
	return handler, router
}

func TestCollaborationHandler_CreateTeam_Unauthorized(t *testing.T) {
	handler, router := setupCollaborationTest()
	router.POST("/teams", handler.CreateTeam)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{"name": "test-team"})
	req, _ := http.NewRequest("POST", "/teams", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCollaborationHandler_GetTeams_Unauthorized(t *testing.T) {
	handler, router := setupCollaborationTest()
	router.GET("/teams", handler.GetTeams)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/teams", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCollaborationHandler_GetTeam_InvalidTeamID(t *testing.T) {
	handler, router := setupCollaborationTest()
	router.GET("/teams/:team_id", handler.GetTeam)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/teams/not-a-uuid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCollaborationHandler_InviteMember_InvalidTeamID(t *testing.T) {
	handler, router := setupCollaborationTest()
	router.POST("/teams/:team_id/members", handler.InviteMember)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{})
	req, _ := http.NewRequest("POST", "/teams/not-a-uuid/members", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCollaborationHandler_CreateChangeRequest_InvalidTeamID(t *testing.T) {
	handler, router := setupCollaborationTest()
	router.POST("/teams/:team_id/changes", handler.CreateChangeRequest)

	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]interface{}{})
	req, _ := http.NewRequest("POST", "/teams/not-a-uuid/changes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCollaborationHandler_GetActivityFeed_InvalidTeamID(t *testing.T) {
	handler, router := setupCollaborationTest()
	router.GET("/teams/:team_id/activity", handler.GetActivityFeed)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/teams/not-a-uuid/activity", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
