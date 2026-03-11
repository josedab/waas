package services

import (
	"archive/zip"
	"bytes"
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

// --- GetConfig Tests ---

func TestSDKGeneratorService_GetConfig_Valid(t *testing.T) {
	t.Parallel()
	repo := &MockSDKGeneratorRepo{}
	logger := utils.NewLogger("test")
	svc := NewSDKGeneratorService(repo, logger)

	tenantID := uuid.New()
	configID := uuid.New()
	config := &models.SDKConfiguration{
		ID:       configID,
		TenantID: tenantID,
		Name:     "Test SDK",
	}
	repo.On("GetConfig", mock.Anything, configID).Return(config, nil)

	result, err := svc.GetConfig(context.Background(), tenantID, configID)
	require.NoError(t, err)
	assert.Equal(t, configID, result.ID)
	assert.Equal(t, tenantID, result.TenantID)
	repo.AssertExpectations(t)
}

func TestSDKGeneratorService_GetConfig_TenantMismatch(t *testing.T) {
	t.Parallel()
	repo := &MockSDKGeneratorRepo{}
	logger := utils.NewLogger("test")
	svc := NewSDKGeneratorService(repo, logger)

	configID := uuid.New()
	config := &models.SDKConfiguration{
		ID:       configID,
		TenantID: uuid.New(),
		Name:     "Test SDK",
	}
	repo.On("GetConfig", mock.Anything, configID).Return(config, nil)

	_, err := svc.GetConfig(context.Background(), uuid.New(), configID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not found")
}

// --- GenerateSDK Tests ---

func TestSDKGeneratorService_GenerateSDK_Valid(t *testing.T) {
	t.Parallel()
	repo := &MockSDKGeneratorRepo{}
	logger := utils.NewLogger("test")
	svc := NewSDKGeneratorService(repo, logger)

	tenantID := uuid.New()
	configID := uuid.New()
	config := &models.SDKConfiguration{
		ID:        configID,
		TenantID:  tenantID,
		Name:      "Test SDK",
		Languages: []string{models.SDKLanguageGo},
	}
	repo.On("GetConfig", mock.Anything, configID).Return(config, nil)
	repo.On("CreateGeneration", mock.Anything, mock.AnythingOfType("*models.SDKGeneration")).Return(nil)
	// Background goroutine calls these; set up expectations loosely
	repo.On("UpdateGenerationStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("GetTemplate", mock.Anything, mock.Anything, mock.Anything).Return(&models.SDKTemplate{}, nil).Maybe()
	repo.On("UpdateGeneration", mock.Anything, mock.Anything).Return(nil).Maybe()

	results, err := svc.GenerateSDK(context.Background(), tenantID, &models.GenerateSDKRequest{
		ConfigID:  configID.String(),
		Version:   "1.0.0",
		Languages: []string{models.SDKLanguageGo},
	})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results[0].Instructions, "go")
	assert.Contains(t, results[0].Instructions, "1.0.0")
}

// --- GetGeneration Tests ---

func TestSDKGeneratorService_GetGeneration_Valid(t *testing.T) {
	t.Parallel()
	repo := &MockSDKGeneratorRepo{}
	logger := utils.NewLogger("test")
	svc := NewSDKGeneratorService(repo, logger)

	tenantID := uuid.New()
	genID := uuid.New()
	gen := &models.SDKGeneration{
		ID:       genID,
		TenantID: tenantID,
		Language: models.SDKLanguageGo,
		Version:  "1.0.0",
	}
	repo.On("GetGeneration", mock.Anything, genID).Return(gen, nil)

	result, err := svc.GetGeneration(context.Background(), tenantID, genID)
	require.NoError(t, err)
	assert.Equal(t, genID, result.ID)
	repo.AssertExpectations(t)
}

func TestSDKGeneratorService_GetGeneration_TenantMismatch(t *testing.T) {
	t.Parallel()
	repo := &MockSDKGeneratorRepo{}
	logger := utils.NewLogger("test")
	svc := NewSDKGeneratorService(repo, logger)

	genID := uuid.New()
	gen := &models.SDKGeneration{
		ID:       genID,
		TenantID: uuid.New(),
	}
	repo.On("GetGeneration", mock.Anything, genID).Return(gen, nil)

	_, err := svc.GetGeneration(context.Background(), uuid.New(), genID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "generation not found")
}

// --- GetGenerations Tests ---

func TestSDKGeneratorService_GetGenerations_Valid(t *testing.T) {
	t.Parallel()
	repo := &MockSDKGeneratorRepo{}
	logger := utils.NewLogger("test")
	svc := NewSDKGeneratorService(repo, logger)

	tenantID := uuid.New()
	configID := uuid.New()
	config := &models.SDKConfiguration{
		ID:       configID,
		TenantID: tenantID,
	}
	gens := []*models.SDKGeneration{
		{ID: uuid.New(), ConfigID: configID, TenantID: tenantID, Language: models.SDKLanguageGo},
	}
	repo.On("GetConfig", mock.Anything, configID).Return(config, nil)
	repo.On("GetGenerationsByConfig", mock.Anything, configID).Return(gens, nil)

	result, err := svc.GetGenerations(context.Background(), tenantID, configID)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	repo.AssertExpectations(t)
}

// --- RecordDownload Tests ---

func TestSDKGeneratorService_RecordDownload_Valid(t *testing.T) {
	t.Parallel()
	repo := &MockSDKGeneratorRepo{}
	logger := utils.NewLogger("test")
	svc := NewSDKGeneratorService(repo, logger)

	tenantID := uuid.New()
	genID := uuid.New()
	gen := &models.SDKGeneration{
		ID:       genID,
		TenantID: tenantID,
	}
	repo.On("GetGeneration", mock.Anything, genID).Return(gen, nil)
	repo.On("RecordDownload", mock.Anything, mock.AnythingOfType("*models.SDKDownload")).Return(nil)

	err := svc.RecordDownload(context.Background(), tenantID, genID, "zip", "127.0.0.1", "test-agent")
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestSDKGeneratorService_RecordDownload_TenantMismatch(t *testing.T) {
	t.Parallel()
	repo := &MockSDKGeneratorRepo{}
	logger := utils.NewLogger("test")
	svc := NewSDKGeneratorService(repo, logger)

	genID := uuid.New()
	gen := &models.SDKGeneration{
		ID:       genID,
		TenantID: uuid.New(),
	}
	repo.On("GetGeneration", mock.Anything, genID).Return(gen, nil)

	err := svc.RecordDownload(context.Background(), uuid.New(), genID, "zip", "127.0.0.1", "test-agent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "generation not found")
}

// --- HashOpenAPISpec Tests ---

func TestSDKGeneratorService_HashOpenAPISpec_Deterministic(t *testing.T) {
	t.Parallel()
	spec := []byte(`{"openapi":"3.0.0","info":{"title":"Test","version":"1.0"}}`)
	hash1 := HashOpenAPISpec(spec)
	hash2 := HashOpenAPISpec(spec)
	assert.Equal(t, hash1, hash2)
	assert.Len(t, hash1, 64) // SHA-256 hex string
}

func TestSDKGeneratorService_HashOpenAPISpec_DifferentSpecs(t *testing.T) {
	t.Parallel()
	spec1 := []byte(`{"openapi":"3.0.0","info":{"title":"A","version":"1.0"}}`)
	spec2 := []byte(`{"openapi":"3.0.0","info":{"title":"B","version":"2.0"}}`)
	assert.NotEqual(t, HashOpenAPISpec(spec1), HashOpenAPISpec(spec2))
}

// --- getMainFilename Tests ---

func TestSDKGeneratorService_GetMainFilename(t *testing.T) {
	t.Parallel()
	svc := NewSDKGeneratorService(nil, utils.NewLogger("test"))

	tests := []struct {
		language string
		expected string
	}{
		{models.SDKLanguageGo, "client.go"},
		{models.SDKLanguageTypeScript, "src/client.ts"},
		{models.SDKLanguagePython, "client.py"},
		{models.SDKLanguageJava, "src/main/java/Client.java"},
		{models.SDKLanguageRuby, "lib/client.rb"},
		{models.SDKLanguagePHP, "src/Client.php"},
		{"unknown", "client"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, svc.getMainFilename(tc.language), "language: %s", tc.language)
	}
}

// --- getManifestFilename Tests ---

func TestSDKGeneratorService_GetManifestFilename(t *testing.T) {
	t.Parallel()
	svc := NewSDKGeneratorService(nil, utils.NewLogger("test"))

	tests := []struct {
		language string
		expected string
	}{
		{models.SDKLanguageGo, "go.mod"},
		{models.SDKLanguageTypeScript, "package.json"},
		{models.SDKLanguagePython, "pyproject.toml"},
		{models.SDKLanguageJava, "pom.xml"},
		{models.SDKLanguageRuby, "Gemfile"},
		{models.SDKLanguagePHP, "composer.json"},
		{"unknown", ""},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, svc.getManifestFilename(tc.language), "language: %s", tc.language)
	}
}

// --- getPackageName Tests ---

func TestSDKGeneratorService_GetPackageName(t *testing.T) {
	t.Parallel()
	svc := NewSDKGeneratorService(nil, utils.NewLogger("test"))

	config := &models.SDKConfiguration{
		PackagePrefix:    "acme",
		OrganizationName: "AcmeCorp",
	}

	assert.Equal(t, "github.com/acme/acmecorp-go", svc.getPackageName(config, models.SDKLanguageGo))
	assert.Equal(t, "@acme/acmecorp", svc.getPackageName(config, models.SDKLanguageTypeScript))
	assert.Equal(t, "acme-acmecorp", svc.getPackageName(config, models.SDKLanguagePython))
	assert.Equal(t, "acme", svc.getPackageName(config, models.SDKLanguageJava))
}

// --- renderTemplate Tests ---

func TestSDKGeneratorService_RenderTemplate_Valid(t *testing.T) {
	t.Parallel()
	svc := NewSDKGeneratorService(nil, utils.NewLogger("test"))

	tmpl := &models.SDKTemplate{
		Content: "package {{.PackageName}}\n// Version: {{.Version}}",
	}
	config := &models.SDKConfiguration{
		PackagePrefix:    "acme",
		OrganizationName: "AcmeCorp",
	}
	gen := &models.SDKGeneration{
		Version: "1.0.0",
	}

	result, err := svc.renderTemplate(tmpl, config, gen)
	require.NoError(t, err)
	assert.Contains(t, result, "package acme")
	assert.Contains(t, result, "Version: 1.0.0")
}

func TestSDKGeneratorService_RenderTemplate_InvalidTemplate(t *testing.T) {
	t.Parallel()
	svc := NewSDKGeneratorService(nil, utils.NewLogger("test"))

	tmpl := &models.SDKTemplate{
		Content: "{{.Invalid",
	}
	config := &models.SDKConfiguration{}
	gen := &models.SDKGeneration{}

	_, err := svc.renderTemplate(tmpl, config, gen)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse template")
}

// --- createSDKArchive Tests ---

func TestSDKGeneratorService_CreateSDKArchive_Structure(t *testing.T) {
	t.Parallel()
	svc := NewSDKGeneratorService(nil, utils.NewLogger("test"))

	config := &models.SDKConfiguration{
		PackagePrefix:    "acme",
		OrganizationName: "AcmeCorp",
		Branding: models.SDKBranding{
			LicenseType: "MIT",
		},
	}
	gen := &models.SDKGeneration{
		Language: models.SDKLanguageGo,
		Version:  "1.0.0",
	}

	archive, err := svc.createSDKArchive(config, gen, "package acme")
	require.NoError(t, err)
	assert.NotEmpty(t, archive)

	// Verify ZIP contents
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	require.NoError(t, err)

	fileNames := make(map[string]bool)
	for _, f := range reader.File {
		fileNames[f.Name] = true
	}
	assert.True(t, fileNames["client.go"], "should contain client.go")
	assert.True(t, fileNames["README.md"], "should contain README.md")
	assert.True(t, fileNames["go.mod"], "should contain go.mod")
	assert.True(t, fileNames["LICENSE"], "should contain LICENSE")
}

// --- generateReadme Tests ---

func TestSDKGeneratorService_GenerateReadme(t *testing.T) {
	t.Parallel()
	svc := NewSDKGeneratorService(nil, utils.NewLogger("test"))

	config := &models.SDKConfiguration{
		PackagePrefix:    "acme",
		OrganizationName: "AcmeCorp",
		Branding: models.SDKBranding{
			DocumentationURL: "https://docs.acme.com",
			LicenseType:      "MIT",
		},
	}
	gen := &models.SDKGeneration{
		Language: models.SDKLanguageGo,
		Version:  "2.0.0",
	}

	readme := svc.generateReadme(config, gen)
	assert.Contains(t, readme, "AcmeCorp")
	assert.Contains(t, readme, "2.0.0")
	assert.Contains(t, readme, "https://docs.acme.com")
	assert.Contains(t, readme, "MIT")
}

// --- generateManifest Tests ---

func TestSDKGeneratorService_GenerateManifest_Go(t *testing.T) {
	t.Parallel()
	svc := NewSDKGeneratorService(nil, utils.NewLogger("test"))

	config := &models.SDKConfiguration{
		PackagePrefix:    "acme",
		OrganizationName: "AcmeCorp",
	}
	gen := &models.SDKGeneration{Language: models.SDKLanguageGo, Version: "1.0.0"}

	manifest := svc.generateManifest(config, gen)
	assert.Contains(t, manifest, "module github.com/acme/acmecorp-go")
	assert.Contains(t, manifest, "go 1.21")
}

func TestSDKGeneratorService_GenerateManifest_TypeScript(t *testing.T) {
	t.Parallel()
	svc := NewSDKGeneratorService(nil, utils.NewLogger("test"))

	config := &models.SDKConfiguration{
		PackagePrefix:    "acme",
		OrganizationName: "AcmeCorp",
	}
	gen := &models.SDKGeneration{Language: models.SDKLanguageTypeScript, Version: "1.0.0"}

	manifest := svc.generateManifest(config, gen)
	assert.Contains(t, manifest, "@acme/acmecorp")
	assert.Contains(t, manifest, "1.0.0")
}

func TestSDKGeneratorService_GenerateManifest_Python(t *testing.T) {
	t.Parallel()
	svc := NewSDKGeneratorService(nil, utils.NewLogger("test"))

	config := &models.SDKConfiguration{
		PackagePrefix:    "acme",
		OrganizationName: "AcmeCorp",
	}
	gen := &models.SDKGeneration{Language: models.SDKLanguagePython, Version: "1.0.0"}

	manifest := svc.generateManifest(config, gen)
	assert.Contains(t, manifest, "acme-acmecorp")
	assert.Contains(t, manifest, "1.0.0")
	assert.Contains(t, manifest, "AcmeCorp SDK")
}

// --- generateLicense Tests ---

func TestSDKGeneratorService_GenerateLicense_MIT(t *testing.T) {
	t.Parallel()
	svc := NewSDKGeneratorService(nil, utils.NewLogger("test"))

	config := &models.SDKConfiguration{
		OrganizationName: "AcmeCorp",
		Branding: models.SDKBranding{
			LicenseType:     "MIT",
			CopyrightHolder: "Acme Inc.",
		},
	}

	license := svc.generateLicense(config)
	assert.Contains(t, license, "MIT License")
	assert.Contains(t, license, "Acme Inc.")
	assert.Contains(t, license, "Permission is hereby granted")
}
