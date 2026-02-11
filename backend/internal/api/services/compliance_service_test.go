package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock ComplianceRepository ---
type MockComplianceRepo struct {
	mock.Mock
}

func (m *MockComplianceRepo) CreateProfile(ctx context.Context, profile *models.ComplianceProfile) error {
	return m.Called(ctx, profile).Error(0)
}
func (m *MockComplianceRepo) GetProfile(ctx context.Context, id uuid.UUID) (*models.ComplianceProfile, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.ComplianceProfile), args.Error(1)
}
func (m *MockComplianceRepo) GetProfilesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.ComplianceProfile, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.ComplianceProfile), args.Error(1)
}
func (m *MockComplianceRepo) GetProfileByFramework(ctx context.Context, tenantID uuid.UUID, framework string) (*models.ComplianceProfile, error) {
	args := m.Called(ctx, tenantID, framework)
	return args.Get(0).(*models.ComplianceProfile), args.Error(1)
}
func (m *MockComplianceRepo) UpdateProfile(ctx context.Context, profile *models.ComplianceProfile) error {
	return m.Called(ctx, profile).Error(0)
}
func (m *MockComplianceRepo) DeleteProfile(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockComplianceRepo) CreateRetentionPolicy(ctx context.Context, policy *models.DataRetentionPolicy) error {
	return m.Called(ctx, policy).Error(0)
}
func (m *MockComplianceRepo) GetRetentionPolicy(ctx context.Context, id uuid.UUID) (*models.DataRetentionPolicy, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.DataRetentionPolicy), args.Error(1)
}
func (m *MockComplianceRepo) GetRetentionPoliciesByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.DataRetentionPolicy, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.DataRetentionPolicy), args.Error(1)
}
func (m *MockComplianceRepo) GetDueRetentionPolicies(ctx context.Context) ([]*models.DataRetentionPolicy, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.DataRetentionPolicy), args.Error(1)
}
func (m *MockComplianceRepo) UpdateRetentionPolicyExecution(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockComplianceRepo) CreatePIIPattern(ctx context.Context, pattern *models.PIIDetectionPattern) error {
	return m.Called(ctx, pattern).Error(0)
}
func (m *MockComplianceRepo) GetPIIPatterns(ctx context.Context, tenantID *uuid.UUID) ([]*models.PIIDetectionPattern, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.PIIDetectionPattern), args.Error(1)
}
func (m *MockComplianceRepo) DeletePIIPattern(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockComplianceRepo) CreatePIIDetection(ctx context.Context, detection *models.PIIDetection) error {
	return m.Called(ctx, detection).Error(0)
}
func (m *MockComplianceRepo) GetPIIDetectionsBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) ([]*models.PIIDetection, error) {
	args := m.Called(ctx, sourceType, sourceID)
	return args.Get(0).([]*models.PIIDetection), args.Error(1)
}
func (m *MockComplianceRepo) GetPIIDetectionsByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.PIIDetection, error) {
	args := m.Called(ctx, tenantID, limit)
	return args.Get(0).([]*models.PIIDetection), args.Error(1)
}
func (m *MockComplianceRepo) CountPIIDetectionsToday(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}
func (m *MockComplianceRepo) CreateAuditLog(ctx context.Context, log *models.ComplianceAuditLog) error {
	return m.Called(ctx, log).Error(0)
}
func (m *MockComplianceRepo) QueryAuditLogs(ctx context.Context, query *models.AuditLogQuery) ([]*models.ComplianceAuditLog, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]*models.ComplianceAuditLog), args.Error(1)
}
func (m *MockComplianceRepo) GetAuditLogsByResource(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*models.ComplianceAuditLog, error) {
	args := m.Called(ctx, resourceType, resourceID)
	return args.Get(0).([]*models.ComplianceAuditLog), args.Error(1)
}
func (m *MockComplianceRepo) CreateReport(ctx context.Context, report *models.ComplianceReport) error {
	return m.Called(ctx, report).Error(0)
}
func (m *MockComplianceRepo) GetReport(ctx context.Context, id uuid.UUID) (*models.ComplianceReport, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.ComplianceReport), args.Error(1)
}
func (m *MockComplianceRepo) GetReportsByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.ComplianceReport, error) {
	args := m.Called(ctx, tenantID, limit)
	return args.Get(0).([]*models.ComplianceReport), args.Error(1)
}
func (m *MockComplianceRepo) UpdateReportStatus(ctx context.Context, id uuid.UUID, status string, reportData map[string]interface{}, artifactURL string) error {
	return m.Called(ctx, id, status, reportData, artifactURL).Error(0)
}
func (m *MockComplianceRepo) CreateFinding(ctx context.Context, finding *models.ComplianceFinding) error {
	return m.Called(ctx, finding).Error(0)
}
func (m *MockComplianceRepo) GetFindingsByReport(ctx context.Context, reportID uuid.UUID) ([]*models.ComplianceFinding, error) {
	args := m.Called(ctx, reportID)
	return args.Get(0).([]*models.ComplianceFinding), args.Error(1)
}
func (m *MockComplianceRepo) GetOpenFindingsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.ComplianceFinding, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.ComplianceFinding), args.Error(1)
}
func (m *MockComplianceRepo) UpdateFindingStatus(ctx context.Context, id uuid.UUID, status string) error {
	return m.Called(ctx, id, status).Error(0)
}
func (m *MockComplianceRepo) CountFindingsBySeverity(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).(map[string]int), args.Error(1)
}
func (m *MockComplianceRepo) CreateDSR(ctx context.Context, dsr *models.DataSubjectRequest) error {
	return m.Called(ctx, dsr).Error(0)
}
func (m *MockComplianceRepo) GetDSR(ctx context.Context, id uuid.UUID) (*models.DataSubjectRequest, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.DataSubjectRequest), args.Error(1)
}
func (m *MockComplianceRepo) GetDSRsByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]*models.DataSubjectRequest, error) {
	args := m.Called(ctx, tenantID, status)
	return args.Get(0).([]*models.DataSubjectRequest), args.Error(1)
}
func (m *MockComplianceRepo) UpdateDSRStatus(ctx context.Context, id uuid.UUID, status string, responseData map[string]interface{}) error {
	return m.Called(ctx, id, status, responseData).Error(0)
}
func (m *MockComplianceRepo) CountPendingDSRs(ctx context.Context, tenantID uuid.UUID) (int, error) {
	args := m.Called(ctx, tenantID)
	return args.Int(0), args.Error(1)
}

// --- Compliance Service Tests ---

func TestComplianceService_CreateProfile_ValidFramework(t *testing.T) {
	t.Parallel()
	repo := &MockComplianceRepo{}
	logger := utils.NewLogger("test")
	svc := NewComplianceService(repo, logger)

	tenantID := uuid.New()
	req := &models.CreateComplianceProfileRequest{
		Name:      "SOC2 Profile",
		Framework: models.ComplianceFrameworkSOC2,
	}

	repo.On("CreateProfile", mock.Anything, mock.AnythingOfType("*models.ComplianceProfile")).Return(nil)
	repo.On("CreateRetentionPolicy", mock.Anything, mock.AnythingOfType("*models.DataRetentionPolicy")).Return(nil).Maybe()

	profile, err := svc.CreateProfile(context.Background(), tenantID, req)
	require.NoError(t, err)
	assert.Equal(t, tenantID, profile.TenantID)
	assert.Equal(t, models.ComplianceFrameworkSOC2, profile.Framework)
	assert.True(t, profile.Enabled)
	assert.NotNil(t, profile.Settings)
}

func TestComplianceService_CreateProfile_InvalidFramework(t *testing.T) {
	t.Parallel()
	repo := &MockComplianceRepo{}
	logger := utils.NewLogger("test")
	svc := NewComplianceService(repo, logger)

	_, err := svc.CreateProfile(context.Background(), uuid.New(), &models.CreateComplianceProfileRequest{
		Name:      "Invalid",
		Framework: "INVALID_FRAMEWORK",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported framework")
}

func TestComplianceService_CreateProfile_RepoError(t *testing.T) {
	t.Parallel()
	repo := &MockComplianceRepo{}
	logger := utils.NewLogger("test")
	svc := NewComplianceService(repo, logger)

	repo.On("CreateProfile", mock.Anything, mock.Anything).Return(fmt.Errorf("db error"))

	_, err := svc.CreateProfile(context.Background(), uuid.New(), &models.CreateComplianceProfileRequest{
		Name:      "GDPR",
		Framework: models.ComplianceFrameworkGDPR,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create profile")
}

func TestComplianceService_GetProfile_TenantIsolation(t *testing.T) {
	t.Parallel()
	repo := &MockComplianceRepo{}
	logger := utils.NewLogger("test")
	svc := NewComplianceService(repo, logger)

	profileID := uuid.New()
	otherTenantID := uuid.New()
	requestingTenantID := uuid.New()

	repo.On("GetProfile", mock.Anything, profileID).Return(&models.ComplianceProfile{
		TenantID: otherTenantID,
	}, nil)

	_, err := svc.GetProfile(context.Background(), requestingTenantID, profileID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "profile not found")
}

func TestComplianceService_GenerateReport(t *testing.T) {
	t.Parallel()
	repo := &MockComplianceRepo{}
	logger := utils.NewLogger("test")
	svc := NewComplianceService(repo, logger)

	tenantID := uuid.New()
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	req := &models.GenerateReportRequest{
		ReportType:  "soc2_audit",
		Title:       "Monthly SOC2 Audit",
		PeriodStart: &start,
		PeriodEnd:   &end,
	}

	repo.On("CreateReport", mock.Anything, mock.AnythingOfType("*models.ComplianceReport")).Return(nil)
	// Async report generation mocks
	repo.On("UpdateReportStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("QueryAuditLogs", mock.Anything, mock.Anything).Return([]*models.ComplianceAuditLog{}, nil).Maybe()
	repo.On("GetRetentionPoliciesByTenant", mock.Anything, mock.Anything).Return([]*models.DataRetentionPolicy{}, nil).Maybe()
	repo.On("CreateFinding", mock.Anything, mock.Anything).Return(nil).Maybe()

	report, err := svc.GenerateReport(context.Background(), tenantID, req)
	require.NoError(t, err)
	assert.Equal(t, tenantID, report.TenantID)
	assert.Equal(t, "soc2_audit", report.ReportType)
	assert.Equal(t, models.ReportStatusPending, report.Status)
}

func TestComplianceService_GenerateReport_RepoError(t *testing.T) {
	t.Parallel()
	repo := &MockComplianceRepo{}
	logger := utils.NewLogger("test")
	svc := NewComplianceService(repo, logger)

	repo.On("CreateReport", mock.Anything, mock.Anything).Return(fmt.Errorf("db error"))

	_, err := svc.GenerateReport(context.Background(), uuid.New(), &models.GenerateReportRequest{
		ReportType: "generic",
		Title:      "Test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create report")
}

func TestComplianceService_CreateDSR_ValidType(t *testing.T) {
	t.Parallel()
	repo := &MockComplianceRepo{}
	logger := utils.NewLogger("test")
	svc := NewComplianceService(repo, logger)

	tenantID := uuid.New()
	repo.On("CreateDSR", mock.Anything, mock.AnythingOfType("*models.DataSubjectRequest")).Return(nil)
	repo.On("CreateAuditLog", mock.Anything, mock.AnythingOfType("*models.ComplianceAuditLog")).Return(nil)

	dsr, err := svc.CreateDSR(context.Background(), tenantID, &models.CreateDSRRequest{
		RequestType:      models.DSRTypeAccess,
		DataSubjectEmail: "user@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, tenantID, dsr.TenantID)
	assert.Equal(t, models.DSRTypeAccess, dsr.RequestType)
}

func TestComplianceService_CreateDSR_InvalidType(t *testing.T) {
	t.Parallel()
	repo := &MockComplianceRepo{}
	logger := utils.NewLogger("test")
	svc := NewComplianceService(repo, logger)

	_, err := svc.CreateDSR(context.Background(), uuid.New(), &models.CreateDSRRequest{
		RequestType: "invalid_type",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid request type")
}

func TestComplianceService_ProcessDSR_NotPending(t *testing.T) {
	t.Parallel()
	repo := &MockComplianceRepo{}
	logger := utils.NewLogger("test")
	svc := NewComplianceService(repo, logger)

	tenantID := uuid.New()
	dsrID := uuid.New()
	repo.On("GetDSR", mock.Anything, dsrID).Return(&models.DataSubjectRequest{
		TenantID: tenantID,
		Status:   models.DSRStatusCompleted,
	}, nil)

	err := svc.ProcessDSR(context.Background(), tenantID, dsrID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in pending status")
}

func TestComplianceService_GetDashboard(t *testing.T) {
	t.Parallel()
	repo := &MockComplianceRepo{}
	logger := utils.NewLogger("test")
	svc := NewComplianceService(repo, logger)

	tenantID := uuid.New()
	repo.On("GetProfilesByTenant", mock.Anything, tenantID).Return([]*models.ComplianceProfile{
		{Enabled: true}, {Enabled: false},
	}, nil)
	repo.On("GetOpenFindingsByTenant", mock.Anything, tenantID).Return([]*models.ComplianceFinding{}, nil)
	repo.On("CountPendingDSRs", mock.Anything, tenantID).Return(3, nil)
	repo.On("CountPIIDetectionsToday", mock.Anything, tenantID).Return(5, nil)
	repo.On("GetReportsByTenant", mock.Anything, tenantID, 5).Return([]*models.ComplianceReport{}, nil)
	repo.On("CountFindingsBySeverity", mock.Anything, tenantID).Return(map[string]int{"critical": 0}, nil)

	dashboard, err := svc.GetDashboard(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, dashboard.ActiveProfiles)
	assert.Equal(t, 3, dashboard.PendingDSRs)
	assert.Equal(t, 5, dashboard.PIIDetectionsToday)
	assert.Equal(t, 100.0, dashboard.ComplianceScore)
}

func TestComplianceService_ScanForPII_NoPatternsFound(t *testing.T) {
	t.Parallel()
	repo := &MockComplianceRepo{}
	logger := utils.NewLogger("test")
	svc := NewComplianceService(repo, logger)

	tenantID := uuid.New()
	repo.On("GetPIIPatterns", mock.Anything, mock.Anything).Return([]*models.PIIDetectionPattern{}, nil)

	result, err := svc.ScanForPII(context.Background(), tenantID, &models.ScanForPIIRequest{
		Content: "no PII here",
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.TotalFound)
	assert.False(t, result.WasRedacted)
}

func TestComplianceService_GetReports_DefaultLimit(t *testing.T) {
	t.Parallel()
	repo := &MockComplianceRepo{}
	logger := utils.NewLogger("test")
	svc := NewComplianceService(repo, logger)

	tenantID := uuid.New()
	repo.On("GetReportsByTenant", mock.Anything, tenantID, 20).Return([]*models.ComplianceReport{}, nil)

	reports, err := svc.GetReports(context.Background(), tenantID, 0)
	require.NoError(t, err)
	assert.Empty(t, reports)
	repo.AssertExpectations(t)
}
