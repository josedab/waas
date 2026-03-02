package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock SDKGeneratorRepository ---
type MockSDKGeneratorRepo struct {
	mock.Mock
}

func (m *MockSDKGeneratorRepo) CreateConfig(ctx context.Context, config *models.SDKConfiguration) error {
	return m.Called(ctx, config).Error(0)
}
func (m *MockSDKGeneratorRepo) GetConfig(ctx context.Context, id uuid.UUID) (*models.SDKConfiguration, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.SDKConfiguration), args.Error(1)
}
func (m *MockSDKGeneratorRepo) GetConfigsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.SDKConfiguration, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*models.SDKConfiguration), args.Error(1)
}
func (m *MockSDKGeneratorRepo) UpdateConfig(ctx context.Context, config *models.SDKConfiguration) error {
	return m.Called(ctx, config).Error(0)
}
func (m *MockSDKGeneratorRepo) DeleteConfig(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockSDKGeneratorRepo) CreateGeneration(ctx context.Context, gen *models.SDKGeneration) error {
	return m.Called(ctx, gen).Error(0)
}
func (m *MockSDKGeneratorRepo) GetGeneration(ctx context.Context, id uuid.UUID) (*models.SDKGeneration, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.SDKGeneration), args.Error(1)
}
func (m *MockSDKGeneratorRepo) GetGenerationsByConfig(ctx context.Context, configID uuid.UUID) ([]*models.SDKGeneration, error) {
	args := m.Called(ctx, configID)
	return args.Get(0).([]*models.SDKGeneration), args.Error(1)
}
func (m *MockSDKGeneratorRepo) GetLatestGeneration(ctx context.Context, configID uuid.UUID, language string) (*models.SDKGeneration, error) {
	args := m.Called(ctx, configID, language)
	return args.Get(0).(*models.SDKGeneration), args.Error(1)
}
func (m *MockSDKGeneratorRepo) UpdateGeneration(ctx context.Context, gen *models.SDKGeneration) error {
	return m.Called(ctx, gen).Error(0)
}
func (m *MockSDKGeneratorRepo) UpdateGenerationStatus(ctx context.Context, id uuid.UUID, status, errorMsg string) error {
	return m.Called(ctx, id, status, errorMsg).Error(0)
}
func (m *MockSDKGeneratorRepo) GetTemplates(ctx context.Context, language string) ([]*models.SDKTemplate, error) {
	args := m.Called(ctx, language)
	return args.Get(0).([]*models.SDKTemplate), args.Error(1)
}
func (m *MockSDKGeneratorRepo) GetTemplate(ctx context.Context, language, templateType string) (*models.SDKTemplate, error) {
	args := m.Called(ctx, language, templateType)
	return args.Get(0).(*models.SDKTemplate), args.Error(1)
}
func (m *MockSDKGeneratorRepo) SaveTemplate(ctx context.Context, template *models.SDKTemplate) error {
	return m.Called(ctx, template).Error(0)
}
func (m *MockSDKGeneratorRepo) RecordDownload(ctx context.Context, download *models.SDKDownload) error {
	return m.Called(ctx, download).Error(0)
}
func (m *MockSDKGeneratorRepo) GetDownloadCount(ctx context.Context, generationID uuid.UUID) (int, error) {
	args := m.Called(ctx, generationID)
	return args.Int(0), args.Error(1)
}

// --- SDK Generator Service Tests ---

func TestSDKGeneratorService_CreateConfig_ValidLanguages(t *testing.T) {
	t.Parallel()
	repo := &MockSDKGeneratorRepo{}
	logger := utils.NewLogger("test")
	svc := NewSDKGeneratorService(repo, logger)

	tenantID := uuid.New()
	repo.On("CreateConfig", mock.Anything, mock.AnythingOfType("*models.SDKConfiguration")).Return(nil)

	config, err := svc.CreateConfig(context.Background(), tenantID, &models.CreateSDKConfigRequest{
		Name:             "My SDK",
		PackagePrefix:    "acme",
		OrganizationName: "AcmeCorp",
		Languages:        []string{models.SDKLanguageGo, models.SDKLanguageTypeScript},
	})
	require.NoError(t, err)
	assert.Equal(t, tenantID, config.TenantID)
	assert.Equal(t, "My SDK", config.Name)
	assert.Equal(t, []string{models.SDKLanguageGo, models.SDKLanguageTypeScript}, config.Languages)
}

func TestSDKGeneratorService_CreateConfig_UnsupportedLanguage(t *testing.T) {
	t.Parallel()
	repo := &MockSDKGeneratorRepo{}
	logger := utils.NewLogger("test")
	svc := NewSDKGeneratorService(repo, logger)

	_, err := svc.CreateConfig(context.Background(), uuid.New(), &models.CreateSDKConfigRequest{
		Name:             "Bad SDK",
		PackagePrefix:    "bad",
		OrganizationName: "BadCorp",
		Languages:        []string{"cobol"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported language")
}

func TestSDKGeneratorService_CreateConfig_SetsDefaults(t *testing.T) {
	t.Parallel()
	repo := &MockSDKGeneratorRepo{}
	logger := utils.NewLogger("test")
	svc := NewSDKGeneratorService(repo, logger)

	tenantID := uuid.New()
	repo.On("CreateConfig", mock.Anything, mock.AnythingOfType("*models.SDKConfiguration")).Return(nil)

	config, err := svc.CreateConfig(context.Background(), tenantID, &models.CreateSDKConfigRequest{
		Name:             "Default SDK",
		PackagePrefix:    "myorg",
		OrganizationName: "MyOrg",
		Languages:        []string{models.SDKLanguagePython},
	})
	require.NoError(t, err)
	assert.Equal(t, "myorg", config.PackagePrefix)
	assert.Equal(t, "MyOrg", config.OrganizationName)
}

func TestSDKGeneratorService_CreateConfig_RepoError(t *testing.T) {
	t.Parallel()
	repo := &MockSDKGeneratorRepo{}
	logger := utils.NewLogger("test")
	svc := NewSDKGeneratorService(repo, logger)

	repo.On("CreateConfig", mock.Anything, mock.Anything).Return(fmt.Errorf("db error"))

	_, err := svc.CreateConfig(context.Background(), uuid.New(), &models.CreateSDKConfigRequest{
		Name:             "Fail SDK",
		PackagePrefix:    "fail",
		OrganizationName: "FailCorp",
		Languages:        []string{models.SDKLanguageGo},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create config")
}

func TestSDKGeneratorService_GetConfigs(t *testing.T) {
	t.Parallel()
	repo := &MockSDKGeneratorRepo{}
	logger := utils.NewLogger("test")
	svc := NewSDKGeneratorService(repo, logger)

	tenantID := uuid.New()
	configs := []*models.SDKConfiguration{
		{TenantID: tenantID, Name: "SDK 1", Languages: []string{models.SDKLanguageGo}},
	}
	repo.On("GetConfigsByTenant", mock.Anything, tenantID).Return(configs, nil)

	result, err := svc.GetConfigs(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	repo.AssertExpectations(t)
}
