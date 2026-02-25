package services

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/models"
	"github.com/josedab/waas/pkg/repository"
	"github.com/josedab/waas/pkg/utils"
)

// SDKGeneratorService handles SDK generation
type SDKGeneratorService struct {
	repo      repository.SDKGeneratorRepository
	logger    *utils.Logger
	storageURL string
}

// NewSDKGeneratorService creates a new SDK generator service
func NewSDKGeneratorService(repo repository.SDKGeneratorRepository, logger *utils.Logger) *SDKGeneratorService {
	return &SDKGeneratorService{
		repo:      repo,
		logger:    logger,
		storageURL: "/sdk-artifacts", // Would be S3 or similar in production
	}
}

// CreateConfig creates a new SDK configuration
func (s *SDKGeneratorService) CreateConfig(ctx context.Context, tenantID uuid.UUID, req *models.CreateSDKConfigRequest) (*models.SDKConfiguration, error) {
	// Validate languages
	validLanguages := map[string]bool{
		models.SDKLanguageGo:         true,
		models.SDKLanguageTypeScript: true,
		models.SDKLanguagePython:     true,
		models.SDKLanguageJava:       true,
		models.SDKLanguageRuby:       true,
		models.SDKLanguagePHP:        true,
	}

	for _, lang := range req.Languages {
		if !validLanguages[lang] {
			return nil, fmt.Errorf("unsupported language: %s", lang)
		}
	}

	config := &models.SDKConfiguration{
		TenantID:         tenantID,
		Name:             req.Name,
		Description:      req.Description,
		PackagePrefix:    req.PackagePrefix,
		OrganizationName: req.OrganizationName,
		Branding:         req.Branding,
		Languages:        req.Languages,
		APIBaseURL:       req.APIBaseURL,
		Features:         req.Features,
	}

	if err := s.repo.CreateConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	return config, nil
}

// GetConfig retrieves an SDK configuration
func (s *SDKGeneratorService) GetConfig(ctx context.Context, tenantID, configID uuid.UUID) (*models.SDKConfiguration, error) {
	config, err := s.repo.GetConfig(ctx, configID)
	if err != nil {
		return nil, err
	}

	if config.TenantID != tenantID {
		return nil, fmt.Errorf("config not found")
	}

	return config, nil
}

// GetConfigs retrieves all SDK configurations for a tenant
func (s *SDKGeneratorService) GetConfigs(ctx context.Context, tenantID uuid.UUID) ([]*models.SDKConfiguration, error) {
	return s.repo.GetConfigsByTenant(ctx, tenantID)
}

// GenerateSDK generates SDK artifacts for the specified languages
func (s *SDKGeneratorService) GenerateSDK(ctx context.Context, tenantID uuid.UUID, req *models.GenerateSDKRequest) ([]*models.SDKGenerationResult, error) {
	configID, err := uuid.Parse(req.ConfigID)
	if err != nil {
		return nil, fmt.Errorf("invalid config_id")
	}

	config, err := s.repo.GetConfig(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("config not found")
	}

	if config.TenantID != tenantID {
		return nil, fmt.Errorf("config not found")
	}

	languages := req.Languages
	if len(languages) == 0 {
		languages = config.Languages
	}

	var results []*models.SDKGenerationResult

	for _, lang := range languages {
		gen := &models.SDKGeneration{
			ConfigID: configID,
			TenantID: tenantID,
			Version:  req.Version,
			Language: lang,
		}

		if err := s.repo.CreateGeneration(ctx, gen); err != nil {
			return nil, fmt.Errorf("failed to create generation record: %w", err)
		}

		// Generate SDK in background
		go func(config *models.SDKConfiguration, gen *models.SDKGeneration) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			s.generateSDKForLanguage(ctx, config, gen)
		}(config, gen)

		results = append(results, &models.SDKGenerationResult{
			Generation:   gen,
			Instructions: fmt.Sprintf("SDK generation for %s v%s started. Check status with GET /sdk/generations/%s", lang, req.Version, gen.ID),
		})
	}

	return results, nil
}

// generateSDKForLanguage generates SDK for a specific language
func (s *SDKGeneratorService) generateSDKForLanguage(ctx context.Context, config *models.SDKConfiguration, gen *models.SDKGeneration) {
	// Update status to generating
	s.repo.UpdateGenerationStatus(ctx, gen.ID, models.SDKStatusGenerating, "")

	var logBuilder strings.Builder
	logBuilder.WriteString(fmt.Sprintf("Starting SDK generation for %s v%s\n", gen.Language, gen.Version))

	// Get template
	tmpl, err := s.repo.GetTemplate(ctx, gen.Language, "client")
	if err != nil {
		s.logger.Error("Failed to get template", map[string]interface{}{"error": err, "language": gen.Language})
		s.repo.UpdateGenerationStatus(ctx, gen.ID, models.SDKStatusFailed, "Template not found for language: "+gen.Language)
		return
	}

	logBuilder.WriteString(fmt.Sprintf("Using template: %s\n", tmpl.Name))

	// Generate the SDK code
	code, err := s.renderTemplate(tmpl, config, gen)
	if err != nil {
		s.logger.Error("Failed to render template", map[string]interface{}{"error": err})
		s.repo.UpdateGenerationStatus(ctx, gen.ID, models.SDKStatusFailed, err.Error())
		return
	}

	logBuilder.WriteString("Template rendered successfully\n")

	// Create ZIP archive
	archive, err := s.createSDKArchive(config, gen, code)
	if err != nil {
		s.logger.Error("Failed to create archive", map[string]interface{}{"error": err})
		s.repo.UpdateGenerationStatus(ctx, gen.ID, models.SDKStatusFailed, err.Error())
		return
	}

	logBuilder.WriteString(fmt.Sprintf("Archive created: %d bytes\n", len(archive)))

	// In production, upload to S3/GCS
	artifactURL := fmt.Sprintf("%s/%s/%s-%s-%s.zip", s.storageURL, gen.TenantID, config.PackagePrefix, gen.Language, gen.Version)

	// Update generation record
	now := time.Now()
	gen.Status = models.SDKStatusCompleted
	gen.ArtifactURL = artifactURL
	gen.ArtifactSizeBytes = int64(len(archive))
	gen.PackageName = s.getPackageName(config, gen.Language)
	gen.PackageRegistry = s.getPackageRegistry(gen.Language)
	gen.GenerationLog = logBuilder.String()
	gen.CompletedAt = &now

	if err := s.repo.UpdateGeneration(ctx, gen); err != nil {
		s.logger.Error("Failed to update generation", map[string]interface{}{"error": err})
	}

	s.logger.Info("SDK generation completed", map[string]interface{}{"generation_id": gen.ID, "language": gen.Language})
}

// renderTemplate renders an SDK template with configuration values
func (s *SDKGeneratorService) renderTemplate(tmpl *models.SDKTemplate, config *models.SDKConfiguration, gen *models.SDKGeneration) (string, error) {
	t, err := template.New("sdk").Parse(tmpl.Content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := map[string]interface{}{
		"PackageName":      config.PackagePrefix,
		"OrganizationName": config.OrganizationName,
		"BaseURL":          config.APIBaseURL,
		"Version":          gen.Version,
		"Branding":         config.Branding,
		"Features":         config.Features,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// createSDKArchive creates a ZIP archive with SDK files
func (s *SDKGeneratorService) createSDKArchive(config *models.SDKConfiguration, gen *models.SDKGeneration, mainCode string) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Add main client file
	filename := s.getMainFilename(gen.Language)
	f, err := w.Create(filename)
	if err != nil {
		return nil, err
	}
	if _, err := f.Write([]byte(mainCode)); err != nil {
		return nil, err
	}

	// Add README
	readme := s.generateReadme(config, gen)
	f, err = w.Create("README.md")
	if err != nil {
		return nil, err
	}
	if _, err := f.Write([]byte(readme)); err != nil {
		return nil, err
	}

	// Add package manifest
	manifest := s.generateManifest(config, gen)
	manifestFilename := s.getManifestFilename(gen.Language)
	if manifestFilename != "" {
		f, err = w.Create(manifestFilename)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write([]byte(manifest)); err != nil {
			return nil, err
		}
	}

	// Add LICENSE if specified
	if config.Branding.LicenseType != "" {
		license := s.generateLicense(config)
		f, err = w.Create("LICENSE")
		if err != nil {
			return nil, err
		}
		if _, err := f.Write([]byte(license)); err != nil {
			return nil, err
		}
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Helper functions

func (s *SDKGeneratorService) getMainFilename(language string) string {
	switch language {
	case models.SDKLanguageGo:
		return "client.go"
	case models.SDKLanguageTypeScript:
		return "src/client.ts"
	case models.SDKLanguagePython:
		return "client.py"
	case models.SDKLanguageJava:
		return "src/main/java/Client.java"
	case models.SDKLanguageRuby:
		return "lib/client.rb"
	case models.SDKLanguagePHP:
		return "src/Client.php"
	default:
		return "client"
	}
}

func (s *SDKGeneratorService) getManifestFilename(language string) string {
	switch language {
	case models.SDKLanguageGo:
		return "go.mod"
	case models.SDKLanguageTypeScript:
		return "package.json"
	case models.SDKLanguagePython:
		return "pyproject.toml"
	case models.SDKLanguageJava:
		return "pom.xml"
	case models.SDKLanguageRuby:
		return "Gemfile"
	case models.SDKLanguagePHP:
		return "composer.json"
	default:
		return ""
	}
}

func (s *SDKGeneratorService) getPackageName(config *models.SDKConfiguration, language string) string {
	switch language {
	case models.SDKLanguageGo:
		return fmt.Sprintf("github.com/%s/%s-go", config.PackagePrefix, strings.ToLower(config.OrganizationName))
	case models.SDKLanguageTypeScript:
		return fmt.Sprintf("@%s/%s", config.PackagePrefix, strings.ToLower(config.OrganizationName))
	case models.SDKLanguagePython:
		return fmt.Sprintf("%s-%s", config.PackagePrefix, strings.ToLower(config.OrganizationName))
	default:
		return config.PackagePrefix
	}
}

func (s *SDKGeneratorService) getPackageRegistry(language string) string {
	switch language {
	case models.SDKLanguageGo:
		return "pkg.go.dev"
	case models.SDKLanguageTypeScript:
		return "npmjs.com"
	case models.SDKLanguagePython:
		return "pypi.org"
	case models.SDKLanguageJava:
		return "maven.org"
	case models.SDKLanguageRuby:
		return "rubygems.org"
	case models.SDKLanguagePHP:
		return "packagist.org"
	default:
		return ""
	}
}

func (s *SDKGeneratorService) generateReadme(config *models.SDKConfiguration, gen *models.SDKGeneration) string {
	return fmt.Sprintf(`# %s SDK for %s

Official %s SDK for %s.

## Installation

%s

## Quick Start

See the [documentation](%s) for more information.

## Version

%s

## License

%s
`, config.OrganizationName, strings.Title(gen.Language),
		gen.Language, config.OrganizationName,
		s.getInstallInstructions(config, gen.Language),
		config.Branding.DocumentationURL,
		gen.Version,
		config.Branding.LicenseType)
}

func (s *SDKGeneratorService) getInstallInstructions(config *models.SDKConfiguration, language string) string {
	packageName := s.getPackageName(config, language)
	switch language {
	case models.SDKLanguageGo:
		return fmt.Sprintf("```bash\ngo get %s\n```", packageName)
	case models.SDKLanguageTypeScript:
		return fmt.Sprintf("```bash\nnpm install %s\n```", packageName)
	case models.SDKLanguagePython:
		return fmt.Sprintf("```bash\npip install %s\n```", packageName)
	default:
		return "See documentation for installation instructions."
	}
}

func (s *SDKGeneratorService) generateManifest(config *models.SDKConfiguration, gen *models.SDKGeneration) string {
	switch gen.Language {
	case models.SDKLanguageGo:
		return fmt.Sprintf("module %s\n\ngo 1.21\n", s.getPackageName(config, gen.Language))
	case models.SDKLanguageTypeScript:
		manifest := map[string]interface{}{
			"name":    s.getPackageName(config, gen.Language),
			"version": gen.Version,
			"main":    "dist/client.js",
			"types":   "dist/client.d.ts",
		}
		data, _ := json.MarshalIndent(manifest, "", "  ")
		return string(data)
	case models.SDKLanguagePython:
		return fmt.Sprintf(`[project]
name = "%s"
version = "%s"
description = "%s SDK"
`, s.getPackageName(config, gen.Language), gen.Version, config.OrganizationName)
	default:
		return ""
	}
}

func (s *SDKGeneratorService) generateLicense(config *models.SDKConfiguration) string {
	year := time.Now().Year()
	holder := config.Branding.CopyrightHolder
	if holder == "" {
		holder = config.OrganizationName
	}

	switch config.Branding.LicenseType {
	case "MIT":
		return fmt.Sprintf(`MIT License

Copyright (c) %d %s

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
`, year, holder)
	default:
		return fmt.Sprintf("Copyright (c) %d %s. All rights reserved.\n", year, holder)
	}
}

// GetGeneration retrieves an SDK generation
func (s *SDKGeneratorService) GetGeneration(ctx context.Context, tenantID, generationID uuid.UUID) (*models.SDKGeneration, error) {
	gen, err := s.repo.GetGeneration(ctx, generationID)
	if err != nil {
		return nil, err
	}

	if gen.TenantID != tenantID {
		return nil, fmt.Errorf("generation not found")
	}

	return gen, nil
}

// GetGenerations retrieves all generations for a config
func (s *SDKGeneratorService) GetGenerations(ctx context.Context, tenantID, configID uuid.UUID) ([]*models.SDKGeneration, error) {
	config, err := s.repo.GetConfig(ctx, configID)
	if err != nil {
		return nil, err
	}

	if config.TenantID != tenantID {
		return nil, fmt.Errorf("config not found")
	}

	return s.repo.GetGenerationsByConfig(ctx, configID)
}

// RecordDownload records an SDK download
func (s *SDKGeneratorService) RecordDownload(ctx context.Context, tenantID, generationID uuid.UUID, downloadType, ipAddress, userAgent string) error {
	gen, err := s.repo.GetGeneration(ctx, generationID)
	if err != nil {
		return err
	}

	if gen.TenantID != tenantID {
		return fmt.Errorf("generation not found")
	}

	download := &models.SDKDownload{
		GenerationID: generationID,
		TenantID:     tenantID,
		DownloadType: downloadType,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}

	return s.repo.RecordDownload(ctx, download)
}

// HashOpenAPISpec generates a hash of the OpenAPI spec for change detection
func HashOpenAPISpec(spec []byte) string {
	hash := sha256.Sum256(spec)
	return hex.EncodeToString(hash[:])
}
