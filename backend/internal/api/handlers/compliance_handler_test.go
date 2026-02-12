package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/josedab/waas/internal/api/services"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockComplianceRepo implements repository.ComplianceRepository using testify/mock
type mockComplianceRepo struct {
	mock.Mock
}

var _ repository.ComplianceRepository = (*mockComplianceRepo)(nil)

func (m *mockComplianceRepo) CreateProfile(ctx context.Context, profile *models.ComplianceProfile) error {
	args := m.Called(mock.Anything, profile)
	return args.Error(0)
}

func (m *mockComplianceRepo) GetProfile(ctx context.Context, id uuid.UUID) (*models.ComplianceProfile, error) {
	args := m.Called(mock.Anything, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ComplianceProfile), args.Error(1)
}

func (m *mockComplianceRepo) GetProfilesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.ComplianceProfile, error) {
	args := m.Called(mock.Anything, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ComplianceProfile), args.Error(1)
}

func (m *mockComplianceRepo) GetProfileByFramework(ctx context.Context, tenantID uuid.UUID, framework string) (*models.ComplianceProfile, error) {
	args := m.Called(mock.Anything, tenantID, framework)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ComplianceProfile), args.Error(1)
}

func (m *mockComplianceRepo) UpdateProfile(ctx context.Context, profile *models.ComplianceProfile) error {
	args := m.Called(mock.Anything, profile)
	return args.Error(0)
}

func (m *mockComplianceRepo) DeleteProfile(ctx context.Context, id uuid.UUID) error {
	args := m.Called(mock.Anything, id)
	return args.Error(0)
}

func (m *mockComplianceRepo) CreateRetentionPolicy(ctx context.Context, policy *models.DataRetentionPolicy) error {
	args := m.Called(mock.Anything, policy)
	return args.Error(0)
}

func (m *mockComplianceRepo) GetRetentionPolicy(ctx context.Context, id uuid.UUID) (*models.DataRetentionPolicy, error) {
	args := m.Called(mock.Anything, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DataRetentionPolicy), args.Error(1)
}

func (m *mockComplianceRepo) GetRetentionPoliciesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.DataRetentionPolicy, error) {
	args := m.Called(mock.Anything, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DataRetentionPolicy), args.Error(1)
}

func (m *mockComplianceRepo) GetDueRetentionPolicies(ctx context.Context) ([]*models.DataRetentionPolicy, error) {
	args := m.Called(mock.Anything)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DataRetentionPolicy), args.Error(1)
}

func (m *mockComplianceRepo) UpdateRetentionPolicyExecution(ctx context.Context, id uuid.UUID) error {
	args := m.Called(mock.Anything, id)
	return args.Error(0)
}

func (m *mockComplianceRepo) CreatePIIPattern(ctx context.Context, pattern *models.PIIDetectionPattern) error {
	args := m.Called(mock.Anything, pattern)
	return args.Error(0)
}

func (m *mockComplianceRepo) GetPIIPatterns(ctx context.Context, tenantID *uuid.UUID) ([]*models.PIIDetectionPattern, error) {
	args := m.Called(mock.Anything, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PIIDetectionPattern), args.Error(1)
}

func (m *mockComplianceRepo) DeletePIIPattern(ctx context.Context, id uuid.UUID) error {
	args := m.Called(mock.Anything, id)
	return args.Error(0)
}

func (m *mockComplianceRepo) CreatePIIDetection(ctx context.Context, detection *models.PIIDetection) error {
	args := m.Called(mock.Anything, detection)
	return args.Error(0)
}

func (m *mockComplianceRepo) GetPIIDetectionsBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) ([]*models.PIIDetection, error) {
	args := m.Called(mock.Anything, sourceType, sourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PIIDetection), args.Error(1)
}

func (m *mockComplianceRepo) GetPIIDetectionsByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.PIIDetection, error) {
	args := m.Called(mock.Anything, tenantID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PIIDetection), args.Error(1)
}

func (m *mockComplianceRepo) CountPIIDetectionsToday(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(mock.Anything, tenantID)
	return args.Int(0), args.Error(1)
}

func (m *mockComplianceRepo) CreateAuditLog(ctx context.Context, log *models.ComplianceAuditLog) error {
	args := m.Called(mock.Anything, log)
	return args.Error(0)
}

func (m *mockComplianceRepo) QueryAuditLogs(ctx context.Context, query *models.AuditLogQuery) ([]*models.ComplianceAuditLog, error) {
	args := m.Called(mock.Anything, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ComplianceAuditLog), args.Error(1)
}

func (m *mockComplianceRepo) GetAuditLogsByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*models.ComplianceAuditLog, error) {
	args := m.Called(mock.Anything, resourceType, resourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ComplianceAuditLog), args.Error(1)
}

func (m *mockComplianceRepo) CreateReport(ctx context.Context, report *models.ComplianceReport) error {
	args := m.Called(mock.Anything, report)
	return args.Error(0)
}

func (m *mockComplianceRepo) GetReport(ctx context.Context, id uuid.UUID) (*models.ComplianceReport, error) {
	args := m.Called(mock.Anything, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ComplianceReport), args.Error(1)
}

func (m *mockComplianceRepo) GetReportsByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.ComplianceReport, error) {
	args := m.Called(mock.Anything, tenantID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ComplianceReport), args.Error(1)
}

func (m *mockComplianceRepo) UpdateReportStatus(ctx context.Context, id uuid.UUID, status string, reportData map[string]interface{}, artifactURL string) error {
	args := m.Called(mock.Anything, id, status, reportData, artifactURL)
	return args.Error(0)
}

func (m *mockComplianceRepo) CreateFinding(ctx context.Context, finding *models.ComplianceFinding) error {
	args := m.Called(mock.Anything, finding)
	return args.Error(0)
}

func (m *mockComplianceRepo) GetFindingsByReport(ctx context.Context, reportID uuid.UUID) ([]*models.ComplianceFinding, error) {
	args := m.Called(mock.Anything, reportID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ComplianceFinding), args.Error(1)
}

func (m *mockComplianceRepo) GetOpenFindingsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.ComplianceFinding, error) {
	args := m.Called(mock.Anything, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ComplianceFinding), args.Error(1)
}

func (m *mockComplianceRepo) UpdateFindingStatus(ctx context.Context, id uuid.UUID, status string) error {
	args := m.Called(mock.Anything, id, status)
	return args.Error(0)
}

func (m *mockComplianceRepo) CountFindingsBySeverity(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	args := m.Called(mock.Anything, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *mockComplianceRepo) CreateDSR(ctx context.Context, dsr *models.DataSubjectRequest) error {
	args := m.Called(mock.Anything, dsr)
	return args.Error(0)
}

func (m *mockComplianceRepo) GetDSR(ctx context.Context, id uuid.UUID) (*models.DataSubjectRequest, error) {
	args := m.Called(mock.Anything, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DataSubjectRequest), args.Error(1)
}

func (m *mockComplianceRepo) GetDSRsByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]*models.DataSubjectRequest, error) {
	args := m.Called(mock.Anything, tenantID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DataSubjectRequest), args.Error(1)
}

func (m *mockComplianceRepo) UpdateDSRStatus(ctx context.Context, id uuid.UUID, status string, responseData map[string]interface{}) error {
	args := m.Called(mock.Anything, id, status, responseData)
	return args.Error(0)
}

func (m *mockComplianceRepo) CountPendingDSRs(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(mock.Anything, tenantID)
	return args.Int(0), args.Error(1)
}

func setupComplianceTestRouter(repo *mockComplianceRepo) (*gin.Engine, *ComplianceHandler) {
	gin.SetMode(gin.TestMode)
	logger := utils.NewLogger("test")
	service := services.NewComplianceService(repo, logger)
	handler := NewComplianceHandler(service, logger)

	router := gin.New()
	router.Use(gin.Recovery())

	api := router.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		if tid := c.GetHeader("X-Tenant-ID"); tid != "" {
			parsed, err := uuid.Parse(tid)
			if err == nil {
				c.Set("tenant_id", parsed)
			}
		}
		c.Next()
	})

	compliance := api.Group("/compliance")
	compliance.GET("/frameworks", handler.GetSupportedFrameworks)
	compliance.GET("/dashboard", handler.GetDashboard)
	compliance.GET("/profiles", handler.GetProfiles)
	compliance.POST("/profiles", handler.CreateProfile)

	return router, handler
}

// --- GetSupportedFrameworks tests ---

func TestGetSupportedFrameworks_Success(t *testing.T) {
	repo := &mockComplianceRepo{}
	router, _ := setupComplianceTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/frameworks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	frameworks, ok := resp["frameworks"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 5, len(frameworks))
}

// --- GetDashboard tests ---

func TestGetDashboard_Success(t *testing.T) {
	repo := &mockComplianceRepo{}
	tenantID := uuid.New()
	router, _ := setupComplianceTestRouter(repo)

	repo.On("GetProfilesByTenant", mock.Anything, tenantID).Return([]*models.ComplianceProfile{}, nil)
	repo.On("GetOpenFindingsByTenant", mock.Anything, tenantID).Return([]*models.ComplianceFinding{}, nil)
	repo.On("CountPendingDSRs", mock.Anything, tenantID).Return(0, nil)
	repo.On("CountPIIDetectionsToday", mock.Anything, tenantID).Return(0, nil)
	repo.On("GetReportsByTenant", mock.Anything, tenantID, 5).Return([]*models.ComplianceReport{}, nil)
	repo.On("CountFindingsBySeverity", mock.Anything, tenantID).Return(map[string]int{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/dashboard", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.ComplianceDashboard
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 100.0, resp.ComplianceScore)
}

func TestGetDashboard_MissingTenantID(t *testing.T) {
	repo := &mockComplianceRepo{}
	router, _ := setupComplianceTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/dashboard", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Handler panics on missing tenant_id; gin.Recovery returns 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- GetProfiles tests ---

func TestGetProfiles_Success(t *testing.T) {
	repo := &mockComplianceRepo{}
	tenantID := uuid.New()
	router, _ := setupComplianceTestRouter(repo)

	profiles := []*models.ComplianceProfile{
		{ID: uuid.New(), TenantID: tenantID, Name: "SOC2", Framework: "soc2", Enabled: true},
	}
	repo.On("GetProfilesByTenant", mock.Anything, tenantID).Return(profiles, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/profiles", nil)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]json.RawMessage
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp, "profiles")
}

func TestGetProfiles_MissingTenantID(t *testing.T) {
	repo := &mockComplianceRepo{}
	router, _ := setupComplianceTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/compliance/profiles", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- CreateProfile tests ---

func TestCreateProfile_Success(t *testing.T) {
	repo := &mockComplianceRepo{}
	tenantID := uuid.New()
	router, _ := setupComplianceTestRouter(repo)

	repo.On("CreateProfile", mock.Anything, mock.AnythingOfType("*models.ComplianceProfile")).Return(nil)
	// CreateProfile spawns a goroutine that creates default retention policies
	repo.On("CreateRetentionPolicy", mock.Anything, mock.Anything).Return(nil).Maybe()

	body := `{"name":"SOC2 Profile","framework":"soc2","description":"SOC2 compliance"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/profiles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp models.ComplianceProfile
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "SOC2 Profile", resp.Name)
	assert.Equal(t, "soc2", resp.Framework)
	assert.True(t, resp.Enabled)
}

func TestCreateProfile_MissingTenantID(t *testing.T) {
	repo := &mockComplianceRepo{}
	router, _ := setupComplianceTestRouter(repo)

	body := `{"name":"SOC2 Profile","framework":"soc2"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/profiles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateProfile_InvalidJSON(t *testing.T) {
	repo := &mockComplianceRepo{}
	tenantID := uuid.New()
	router, _ := setupComplianceTestRouter(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/profiles", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateProfile_MissingRequiredFields(t *testing.T) {
	repo := &mockComplianceRepo{}
	tenantID := uuid.New()
	router, _ := setupComplianceTestRouter(repo)

	body := `{"name":"SOC2 Profile"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/profiles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateProfile_UnsupportedFramework(t *testing.T) {
	repo := &mockComplianceRepo{}
	tenantID := uuid.New()
	router, _ := setupComplianceTestRouter(repo)

	body := `{"name":"Invalid","framework":"unknown_framework"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/compliance/profiles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID.String())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
