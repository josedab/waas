package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
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

// mockCollaborationRepo is an in-memory mock of repository.CollaborationRepository.
type mockCollaborationRepo struct {
	mu              sync.RWMutex
	teams           map[uuid.UUID]*models.Team
	members         map[uuid.UUID]*models.CollabTeamMember
	configs         map[uuid.UUID]*models.SharedConfiguration
	configVersions  map[uuid.UUID][]*models.ConfigVersion
	changeRequests  map[uuid.UUID]*models.ChangeRequest
	reviews         map[uuid.UUID][]*models.ChangeRequestReview
	comments        map[uuid.UUID][]*models.ChangeRequestComment
	activities      map[uuid.UUID][]*models.ActivityFeedItem
	notifPrefs      map[uuid.UUID][]*models.NotificationPreference
	notifInteg      map[uuid.UUID][]*models.NotificationIntegration
}

func newMockCollaborationRepo() *mockCollaborationRepo {
	return &mockCollaborationRepo{
		teams:          make(map[uuid.UUID]*models.Team),
		members:        make(map[uuid.UUID]*models.CollabTeamMember),
		configs:        make(map[uuid.UUID]*models.SharedConfiguration),
		configVersions: make(map[uuid.UUID][]*models.ConfigVersion),
		changeRequests: make(map[uuid.UUID]*models.ChangeRequest),
		reviews:        make(map[uuid.UUID][]*models.ChangeRequestReview),
		comments:       make(map[uuid.UUID][]*models.ChangeRequestComment),
		activities:     make(map[uuid.UUID][]*models.ActivityFeedItem),
		notifPrefs:     make(map[uuid.UUID][]*models.NotificationPreference),
		notifInteg:     make(map[uuid.UUID][]*models.NotificationIntegration),
	}
}

func (m *mockCollaborationRepo) CreateTeam(_ context.Context, team *models.Team) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if team.ID == uuid.Nil {
		team.ID = uuid.New()
	}
	m.teams[team.ID] = team
	return nil
}

func (m *mockCollaborationRepo) GetTeam(_ context.Context, id uuid.UUID) (*models.Team, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.teams[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return t, nil
}

func (m *mockCollaborationRepo) GetTeamsByTenant(_ context.Context, tenantID uuid.UUID) ([]*models.Team, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.Team
	for _, t := range m.teams {
		if t.TenantID == tenantID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockCollaborationRepo) UpdateTeam(_ context.Context, team *models.Team) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.teams[team.ID] = team
	return nil
}

func (m *mockCollaborationRepo) DeleteTeam(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.teams, id)
	return nil
}

func (m *mockCollaborationRepo) AddTeamMember(_ context.Context, member *models.CollabTeamMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if member.ID == uuid.Nil {
		member.ID = uuid.New()
	}
	m.members[member.ID] = member
	return nil
}

func (m *mockCollaborationRepo) GetTeamMember(_ context.Context, id uuid.UUID) (*models.CollabTeamMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mem, ok := m.members[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return mem, nil
}

func (m *mockCollaborationRepo) GetTeamMembers(_ context.Context, teamID uuid.UUID) ([]*models.CollabTeamMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.CollabTeamMember
	for _, mem := range m.members {
		if mem.TeamID == teamID {
			result = append(result, mem)
		}
	}
	return result, nil
}

func (m *mockCollaborationRepo) GetTeamMemberByEmail(_ context.Context, teamID uuid.UUID, email string) (*models.CollabTeamMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, mem := range m.members {
		if mem.TeamID == teamID && mem.Email == email {
			return mem, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockCollaborationRepo) UpdateTeamMember(_ context.Context, member *models.CollabTeamMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.members[member.ID] = member
	return nil
}

func (m *mockCollaborationRepo) RemoveTeamMember(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.members, id)
	return nil
}

func (m *mockCollaborationRepo) CreateSharedConfig(_ context.Context, config *models.SharedConfiguration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}
	m.configs[config.ID] = config
	return nil
}

func (m *mockCollaborationRepo) GetSharedConfig(_ context.Context, id uuid.UUID) (*models.SharedConfiguration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.configs[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return c, nil
}

func (m *mockCollaborationRepo) GetTeamConfigs(_ context.Context, teamID uuid.UUID) ([]*models.SharedConfiguration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.SharedConfiguration
	for _, c := range m.configs {
		if c.TeamID == teamID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockCollaborationRepo) UpdateSharedConfig(_ context.Context, config *models.SharedConfiguration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[config.ID] = config
	return nil
}

func (m *mockCollaborationRepo) DeleteSharedConfig(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.configs, id)
	return nil
}

func (m *mockCollaborationRepo) LockConfig(_ context.Context, id, _ uuid.UUID) error {
	return nil
}

func (m *mockCollaborationRepo) UnlockConfig(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockCollaborationRepo) SaveConfigVersion(_ context.Context, version *models.ConfigVersion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if version.ID == uuid.Nil {
		version.ID = uuid.New()
	}
	m.configVersions[version.ConfigID] = append(m.configVersions[version.ConfigID], version)
	return nil
}

func (m *mockCollaborationRepo) GetConfigVersions(_ context.Context, configID uuid.UUID) ([]*models.ConfigVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.configVersions[configID], nil
}

func (m *mockCollaborationRepo) GetConfigVersion(_ context.Context, configID uuid.UUID, version int) (*models.ConfigVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.configVersions[configID] {
		if v.Version == version {
			return v, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockCollaborationRepo) CreateChangeRequest(_ context.Context, cr *models.ChangeRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cr.ID == uuid.Nil {
		cr.ID = uuid.New()
	}
	cr.Status = models.ChangeRequestPending
	m.changeRequests[cr.ID] = cr
	return nil
}

func (m *mockCollaborationRepo) GetChangeRequest(_ context.Context, id uuid.UUID) (*models.ChangeRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cr, ok := m.changeRequests[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return cr, nil
}

func (m *mockCollaborationRepo) GetTeamChangeRequests(_ context.Context, teamID uuid.UUID, status string) ([]*models.ChangeRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*models.ChangeRequest
	for _, cr := range m.changeRequests {
		if cr.TeamID == teamID && (status == "" || cr.Status == status) {
			result = append(result, cr)
		}
	}
	return result, nil
}

func (m *mockCollaborationRepo) UpdateChangeRequest(_ context.Context, cr *models.ChangeRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.changeRequests[cr.ID] = cr
	return nil
}

func (m *mockCollaborationRepo) AddReview(_ context.Context, review *models.ChangeRequestReview) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if review.ID == uuid.Nil {
		review.ID = uuid.New()
	}
	m.reviews[review.ChangeRequestID] = append(m.reviews[review.ChangeRequestID], review)
	return nil
}

func (m *mockCollaborationRepo) GetReviews(_ context.Context, crID uuid.UUID) ([]*models.ChangeRequestReview, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.reviews[crID], nil
}

func (m *mockCollaborationRepo) AddComment(_ context.Context, comment *models.ChangeRequestComment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if comment.ID == uuid.Nil {
		comment.ID = uuid.New()
	}
	m.comments[comment.ChangeRequestID] = append(m.comments[comment.ChangeRequestID], comment)
	return nil
}

func (m *mockCollaborationRepo) GetComments(_ context.Context, crID uuid.UUID) ([]*models.ChangeRequestComment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.comments[crID], nil
}

func (m *mockCollaborationRepo) AddActivity(_ context.Context, activity *models.ActivityFeedItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if activity.ID == uuid.Nil {
		activity.ID = uuid.New()
	}
	m.activities[activity.TeamID] = append(m.activities[activity.TeamID], activity)
	return nil
}

func (m *mockCollaborationRepo) GetTeamActivity(_ context.Context, teamID uuid.UUID, limit, offset int) ([]*models.ActivityFeedItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	all := m.activities[teamID]
	if offset >= len(all) {
		return nil, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

func (m *mockCollaborationRepo) SaveNotificationPreference(_ context.Context, pref *models.NotificationPreference) error {
	return nil
}

func (m *mockCollaborationRepo) GetNotificationPreferences(_ context.Context, _ uuid.UUID) ([]*models.NotificationPreference, error) {
	return nil, nil
}

func (m *mockCollaborationRepo) CreateNotificationIntegration(_ context.Context, _ *models.NotificationIntegration) error {
	return nil
}

func (m *mockCollaborationRepo) GetNotificationIntegrations(_ context.Context, _ uuid.UUID) ([]*models.NotificationIntegration, error) {
	return nil, nil
}

func (m *mockCollaborationRepo) RecordSentNotification(_ context.Context, _ *models.SentNotification) error {
	return nil
}

func setupCollaborationTestWithMock() (*CollaborationHandler, *gin.Engine, *mockCollaborationRepo) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	mock := newMockCollaborationRepo()
	handler := NewCollaborationHandler(mock, logger)
	router := gin.New()
	return handler, router, mock
}

func setTenantID(tenantID uuid.UUID) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Next()
	}
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

// --- Happy-path tests ---

func TestCollaborationHandler_CreateTeam_Success(t *testing.T) {
	handler, router, _ := setupCollaborationTestWithMock()
	tenantID := uuid.New()
	router.POST("/teams", setTenantID(tenantID), handler.CreateTeam)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "alpha-team",
		"description": "Alpha squad",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/teams", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp models.Team
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "alpha-team", resp.Name)
	assert.Equal(t, tenantID, resp.TenantID)
}

func TestCollaborationHandler_GetTeams_Success(t *testing.T) {
	handler, router, mock := setupCollaborationTestWithMock()
	tenantID := uuid.New()
	router.GET("/teams", setTenantID(tenantID), handler.GetTeams)

	mock.CreateTeam(context.Background(), &models.Team{TenantID: tenantID, Name: "team-a"})
	mock.CreateTeam(context.Background(), &models.Team{TenantID: tenantID, Name: "team-b"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/teams", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []models.Team
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
}

func TestCollaborationHandler_GetTeam_Success(t *testing.T) {
	handler, router, mock := setupCollaborationTestWithMock()
	teamID := uuid.New()
	mock.teams[teamID] = &models.Team{ID: teamID, Name: "my-team"}
	router.GET("/teams/:team_id", handler.GetTeam)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/teams/"+teamID.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.Team
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "my-team", resp.Name)
}

func TestCollaborationHandler_GetTeamMembers_Success(t *testing.T) {
	handler, router, mock := setupCollaborationTestWithMock()
	teamID := uuid.New()
	mock.AddTeamMember(context.Background(), &models.CollabTeamMember{TeamID: teamID, Email: "a@test.com", Role: "editor"})
	mock.AddTeamMember(context.Background(), &models.CollabTeamMember{TeamID: teamID, Email: "b@test.com", Role: "viewer"})
	router.GET("/teams/:team_id/members", handler.GetTeamMembers)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/teams/"+teamID.String()+"/members", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []models.CollabTeamMember
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
}

func TestCollaborationHandler_InviteMember_Success(t *testing.T) {
	handler, router, _ := setupCollaborationTestWithMock()
	teamID := uuid.New()
	router.POST("/teams/:team_id/members", handler.InviteMember)

	body, _ := json.Marshal(map[string]interface{}{
		"email": "newuser@test.com",
		"role":  "editor",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/teams/"+teamID.String()+"/members", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp models.CollabTeamMember
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "newuser@test.com", resp.Email)
	assert.Equal(t, "editor", resp.Role)
}

func TestCollaborationHandler_InviteMember_Duplicate(t *testing.T) {
	handler, router, mock := setupCollaborationTestWithMock()
	teamID := uuid.New()
	mock.AddTeamMember(context.Background(), &models.CollabTeamMember{TeamID: teamID, Email: "dup@test.com", Role: "editor"})
	router.POST("/teams/:team_id/members", handler.InviteMember)

	body, _ := json.Marshal(map[string]interface{}{
		"email": "dup@test.com",
		"role":  "viewer",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/teams/"+teamID.String()+"/members", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCollaborationHandler_CreateChangeRequest_Success(t *testing.T) {
	handler, router, mock := setupCollaborationTestWithMock()
	teamID := uuid.New()
	configID := uuid.New()
	tenantID := uuid.New()

	mock.configs[configID] = &models.SharedConfiguration{
		ID:         configID,
		TeamID:     teamID,
		Name:       "prod-config",
		ConfigData: map[string]interface{}{"key": "old"},
		Version:    3,
	}
	router.POST("/teams/:team_id/changes", setTenantID(tenantID), handler.CreateChangeRequest)

	body, _ := json.Marshal(map[string]interface{}{
		"config_id":        configID.String(),
		"title":            "Update key",
		"description":      "Change key value",
		"proposed_changes": map[string]interface{}{"key": "new"},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/teams/"+teamID.String()+"/changes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp models.ChangeRequest
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Update key", resp.Title)
	assert.Equal(t, 3, resp.BaseVersion)
	assert.Equal(t, models.ChangeRequestPending, resp.Status)
}

func TestCollaborationHandler_GetChangeRequests_Success(t *testing.T) {
	handler, router, mock := setupCollaborationTestWithMock()
	teamID := uuid.New()
	mock.changeRequests[uuid.New()] = &models.ChangeRequest{ID: uuid.New(), TeamID: teamID, Title: "CR1", Status: models.ChangeRequestPending}
	mock.changeRequests[uuid.New()] = &models.ChangeRequest{ID: uuid.New(), TeamID: teamID, Title: "CR2", Status: models.ChangeRequestPending}
	router.GET("/teams/:team_id/changes", handler.GetChangeRequests)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/teams/"+teamID.String()+"/changes", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []models.ChangeRequest
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
}

func TestCollaborationHandler_ReviewChangeRequest_Approve(t *testing.T) {
	handler, router, mock := setupCollaborationTestWithMock()
	teamID := uuid.New()
	crID := uuid.New()
	mock.changeRequests[crID] = &models.ChangeRequest{ID: crID, TeamID: teamID, Title: "CR", Status: models.ChangeRequestPending}
	router.POST("/teams/:team_id/changes/:cr_id/reviews", handler.ReviewChangeRequest)

	body, _ := json.Marshal(map[string]interface{}{
		"status":   models.ReviewApproved,
		"comments": "Looks good",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/teams/"+teamID.String()+"/changes/"+crID.String()+"/reviews", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.ChangeRequestReview
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, models.ReviewApproved, resp.Status)

	// Verify CR status was updated to approved
	assert.Equal(t, models.ChangeRequestApproved, mock.changeRequests[crID].Status)
}

func TestCollaborationHandler_MergeChangeRequest_Success(t *testing.T) {
	handler, router, mock := setupCollaborationTestWithMock()
	teamID := uuid.New()
	crID := uuid.New()
	configID := uuid.New()
	createdBy := uuid.New()

	mock.configs[configID] = &models.SharedConfiguration{
		ID:         configID,
		TeamID:     teamID,
		Name:       "prod",
		ConfigData: map[string]interface{}{"a": "1"},
		Version:    2,
	}
	mock.changeRequests[crID] = &models.ChangeRequest{
		ID:              crID,
		TeamID:          teamID,
		ConfigID:        configID,
		Title:           "Update a",
		Status:          models.ChangeRequestApproved,
		ProposedChanges: map[string]interface{}{"a": "2"},
		CreatedBy:       createdBy,
	}
	router.POST("/teams/:team_id/changes/:cr_id/merge", handler.MergeChangeRequest)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/teams/"+teamID.String()+"/changes/"+crID.String()+"/merge", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, models.ChangeRequestMerged, mock.changeRequests[crID].Status)
	assert.Equal(t, "2", mock.configs[configID].ConfigData["a"])
}

func TestCollaborationHandler_MergeChangeRequest_NotApproved(t *testing.T) {
	handler, router, mock := setupCollaborationTestWithMock()
	teamID := uuid.New()
	crID := uuid.New()
	mock.changeRequests[crID] = &models.ChangeRequest{
		ID:     crID,
		TeamID: teamID,
		Status: models.ChangeRequestPending,
	}
	router.POST("/teams/:team_id/changes/:cr_id/merge", handler.MergeChangeRequest)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/teams/"+teamID.String()+"/changes/"+crID.String()+"/merge", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCollaborationHandler_GetActivityFeed_Success(t *testing.T) {
	handler, router, mock := setupCollaborationTestWithMock()
	teamID := uuid.New()
	mock.activities[teamID] = []*models.ActivityFeedItem{
		{ID: uuid.New(), TeamID: teamID, ActionType: "team_created", ResourceType: "team"},
		{ID: uuid.New(), TeamID: teamID, ActionType: "member_invited", ResourceType: "member"},
	}
	router.GET("/teams/:team_id/activity", handler.GetActivityFeed)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/teams/"+teamID.String()+"/activity", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []models.ActivityFeedItem
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
}
