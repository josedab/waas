package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"webhook-platform/pkg/database"
	"webhook-platform/pkg/models"
)

type SDKGeneratorRepository interface {
	// Configuration operations
	CreateConfig(ctx context.Context, config *models.SDKConfiguration) error
	GetConfig(ctx context.Context, id uuid.UUID) (*models.SDKConfiguration, error)
	GetConfigsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.SDKConfiguration, error)
	UpdateConfig(ctx context.Context, config *models.SDKConfiguration) error
	DeleteConfig(ctx context.Context, id uuid.UUID) error

	// Generation operations
	CreateGeneration(ctx context.Context, gen *models.SDKGeneration) error
	GetGeneration(ctx context.Context, id uuid.UUID) (*models.SDKGeneration, error)
	GetGenerationsByConfig(ctx context.Context, configID uuid.UUID) ([]*models.SDKGeneration, error)
	GetLatestGeneration(ctx context.Context, configID uuid.UUID, language string) (*models.SDKGeneration, error)
	UpdateGeneration(ctx context.Context, gen *models.SDKGeneration) error
	UpdateGenerationStatus(ctx context.Context, id uuid.UUID, status, errorMsg string) error

	// Template operations
	GetTemplates(ctx context.Context, language string) ([]*models.SDKTemplate, error)
	GetTemplate(ctx context.Context, language, templateType string) (*models.SDKTemplate, error)
	SaveTemplate(ctx context.Context, template *models.SDKTemplate) error

	// Download tracking
	RecordDownload(ctx context.Context, download *models.SDKDownload) error
	GetDownloadCount(ctx context.Context, generationID uuid.UUID) (int, error)
}

type sdkGeneratorRepository struct {
	db *database.DB
}

func NewSDKGeneratorRepository(db *database.DB) SDKGeneratorRepository {
	return &sdkGeneratorRepository{db: db}
}

func (r *sdkGeneratorRepository) CreateConfig(ctx context.Context, config *models.SDKConfiguration) error {
	query := `
		INSERT INTO sdk_configurations (id, tenant_id, name, description, package_prefix, organization_name, 
		                                branding, languages, api_base_url, features, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now
	config.IsActive = true

	brandingJSON, _ := json.Marshal(config.Branding)
	languagesJSON, _ := json.Marshal(config.Languages)
	featuresJSON, _ := json.Marshal(config.Features)

	_, err := r.db.Pool.Exec(ctx, query,
		config.ID, config.TenantID, config.Name, config.Description,
		config.PackagePrefix, config.OrganizationName, brandingJSON, languagesJSON,
		config.APIBaseURL, featuresJSON, config.IsActive, config.CreatedAt, config.UpdatedAt)
	return err
}

func (r *sdkGeneratorRepository) GetConfig(ctx context.Context, id uuid.UUID) (*models.SDKConfiguration, error) {
	query := `
		SELECT id, tenant_id, name, description, package_prefix, organization_name, branding, 
		       languages, api_base_url, features, is_active, created_at, updated_at
		FROM sdk_configurations WHERE id = $1
	`

	var config models.SDKConfiguration
	var brandingJSON, languagesJSON, featuresJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&config.ID, &config.TenantID, &config.Name, &config.Description,
		&config.PackagePrefix, &config.OrganizationName, &brandingJSON, &languagesJSON,
		&config.APIBaseURL, &featuresJSON, &config.IsActive, &config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(brandingJSON, &config.Branding)
	json.Unmarshal(languagesJSON, &config.Languages)
	json.Unmarshal(featuresJSON, &config.Features)

	return &config, nil
}

func (r *sdkGeneratorRepository) GetConfigsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.SDKConfiguration, error) {
	query := `
		SELECT id, tenant_id, name, description, package_prefix, organization_name, branding,
		       languages, api_base_url, features, is_active, created_at, updated_at
		FROM sdk_configurations WHERE tenant_id = $1 ORDER BY name
	`

	rows, err := r.db.Pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.SDKConfiguration
	for rows.Next() {
		var config models.SDKConfiguration
		var brandingJSON, languagesJSON, featuresJSON []byte

		if err := rows.Scan(&config.ID, &config.TenantID, &config.Name, &config.Description,
			&config.PackagePrefix, &config.OrganizationName, &brandingJSON, &languagesJSON,
			&config.APIBaseURL, &featuresJSON, &config.IsActive, &config.CreatedAt, &config.UpdatedAt); err != nil {
			return nil, err
		}

		json.Unmarshal(brandingJSON, &config.Branding)
		json.Unmarshal(languagesJSON, &config.Languages)
		json.Unmarshal(featuresJSON, &config.Features)
		configs = append(configs, &config)
	}

	return configs, nil
}

func (r *sdkGeneratorRepository) UpdateConfig(ctx context.Context, config *models.SDKConfiguration) error {
	query := `
		UPDATE sdk_configurations 
		SET name = $2, description = $3, package_prefix = $4, organization_name = $5,
		    branding = $6, languages = $7, api_base_url = $8, features = $9, is_active = $10, updated_at = $11
		WHERE id = $1
	`

	config.UpdatedAt = time.Now()
	brandingJSON, _ := json.Marshal(config.Branding)
	languagesJSON, _ := json.Marshal(config.Languages)
	featuresJSON, _ := json.Marshal(config.Features)

	_, err := r.db.Pool.Exec(ctx, query,
		config.ID, config.Name, config.Description, config.PackagePrefix, config.OrganizationName,
		brandingJSON, languagesJSON, config.APIBaseURL, featuresJSON, config.IsActive, config.UpdatedAt)
	return err
}

func (r *sdkGeneratorRepository) DeleteConfig(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM sdk_configurations WHERE id = $1`, id)
	return err
}

// Generation operations

func (r *sdkGeneratorRepository) CreateGeneration(ctx context.Context, gen *models.SDKGeneration) error {
	query := `
		INSERT INTO sdk_generations (id, config_id, tenant_id, version, language, status, openapi_spec_hash, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	if gen.ID == uuid.Nil {
		gen.ID = uuid.New()
	}
	gen.CreatedAt = time.Now()
	gen.Status = models.SDKStatusPending

	_, err := r.db.Pool.Exec(ctx, query,
		gen.ID, gen.ConfigID, gen.TenantID, gen.Version, gen.Language, gen.Status, gen.OpenAPISpecHash, gen.CreatedAt)
	return err
}

func (r *sdkGeneratorRepository) GetGeneration(ctx context.Context, id uuid.UUID) (*models.SDKGeneration, error) {
	query := `
		SELECT id, config_id, tenant_id, version, language, status, openapi_spec_hash,
		       artifact_url, artifact_size_bytes, package_registry, package_name,
		       generation_log, error_message, started_at, completed_at, created_at
		FROM sdk_generations WHERE id = $1
	`

	var gen models.SDKGeneration
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&gen.ID, &gen.ConfigID, &gen.TenantID, &gen.Version, &gen.Language, &gen.Status, &gen.OpenAPISpecHash,
		&gen.ArtifactURL, &gen.ArtifactSizeBytes, &gen.PackageRegistry, &gen.PackageName,
		&gen.GenerationLog, &gen.ErrorMessage, &gen.StartedAt, &gen.CompletedAt, &gen.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &gen, nil
}

func (r *sdkGeneratorRepository) GetGenerationsByConfig(ctx context.Context, configID uuid.UUID) ([]*models.SDKGeneration, error) {
	query := `
		SELECT id, config_id, tenant_id, version, language, status, openapi_spec_hash,
		       artifact_url, artifact_size_bytes, package_registry, package_name,
		       generation_log, error_message, started_at, completed_at, created_at
		FROM sdk_generations WHERE config_id = $1 ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, configID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var generations []*models.SDKGeneration
	for rows.Next() {
		var gen models.SDKGeneration
		if err := rows.Scan(&gen.ID, &gen.ConfigID, &gen.TenantID, &gen.Version, &gen.Language, &gen.Status, &gen.OpenAPISpecHash,
			&gen.ArtifactURL, &gen.ArtifactSizeBytes, &gen.PackageRegistry, &gen.PackageName,
			&gen.GenerationLog, &gen.ErrorMessage, &gen.StartedAt, &gen.CompletedAt, &gen.CreatedAt); err != nil {
			return nil, err
		}
		generations = append(generations, &gen)
	}

	return generations, nil
}

func (r *sdkGeneratorRepository) GetLatestGeneration(ctx context.Context, configID uuid.UUID, language string) (*models.SDKGeneration, error) {
	query := `
		SELECT id, config_id, tenant_id, version, language, status, openapi_spec_hash,
		       artifact_url, artifact_size_bytes, package_registry, package_name,
		       generation_log, error_message, started_at, completed_at, created_at
		FROM sdk_generations 
		WHERE config_id = $1 AND language = $2 AND status = $3
		ORDER BY created_at DESC LIMIT 1
	`

	var gen models.SDKGeneration
	err := r.db.Pool.QueryRow(ctx, query, configID, language, models.SDKStatusCompleted).Scan(
		&gen.ID, &gen.ConfigID, &gen.TenantID, &gen.Version, &gen.Language, &gen.Status, &gen.OpenAPISpecHash,
		&gen.ArtifactURL, &gen.ArtifactSizeBytes, &gen.PackageRegistry, &gen.PackageName,
		&gen.GenerationLog, &gen.ErrorMessage, &gen.StartedAt, &gen.CompletedAt, &gen.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &gen, nil
}

func (r *sdkGeneratorRepository) UpdateGeneration(ctx context.Context, gen *models.SDKGeneration) error {
	query := `
		UPDATE sdk_generations 
		SET status = $2, artifact_url = $3, artifact_size_bytes = $4, package_registry = $5,
		    package_name = $6, generation_log = $7, error_message = $8, started_at = $9, completed_at = $10
		WHERE id = $1
	`

	_, err := r.db.Pool.Exec(ctx, query,
		gen.ID, gen.Status, gen.ArtifactURL, gen.ArtifactSizeBytes, gen.PackageRegistry,
		gen.PackageName, gen.GenerationLog, gen.ErrorMessage, gen.StartedAt, gen.CompletedAt)
	return err
}

func (r *sdkGeneratorRepository) UpdateGenerationStatus(ctx context.Context, id uuid.UUID, status, errorMsg string) error {
	query := `UPDATE sdk_generations SET status = $2, error_message = $3`
	args := []interface{}{id, status, errorMsg}

	if status == models.SDKStatusGenerating {
		query += ", started_at = $4 WHERE id = $1"
		args = append(args, time.Now())
	} else if status == models.SDKStatusCompleted || status == models.SDKStatusFailed {
		query += ", completed_at = $4 WHERE id = $1"
		args = append(args, time.Now())
	} else {
		query += " WHERE id = $1"
	}

	_, err := r.db.Pool.Exec(ctx, query, args...)
	return err
}

// Template operations

func (r *sdkGeneratorRepository) GetTemplates(ctx context.Context, language string) ([]*models.SDKTemplate, error) {
	query := `SELECT id, language, template_type, name, content, variables, is_default, created_at, updated_at FROM sdk_templates WHERE language = $1`

	rows, err := r.db.Pool.Query(ctx, query, language)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*models.SDKTemplate
	for rows.Next() {
		var t models.SDKTemplate
		var variablesJSON []byte

		if err := rows.Scan(&t.ID, &t.Language, &t.TemplateType, &t.Name, &t.Content, &variablesJSON, &t.IsDefault, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}

		json.Unmarshal(variablesJSON, &t.Variables)
		templates = append(templates, &t)
	}

	return templates, nil
}

func (r *sdkGeneratorRepository) GetTemplate(ctx context.Context, language, templateType string) (*models.SDKTemplate, error) {
	query := `SELECT id, language, template_type, name, content, variables, is_default, created_at, updated_at FROM sdk_templates WHERE language = $1 AND template_type = $2 AND is_default = true`

	var t models.SDKTemplate
	var variablesJSON []byte

	err := r.db.Pool.QueryRow(ctx, query, language, templateType).Scan(
		&t.ID, &t.Language, &t.TemplateType, &t.Name, &t.Content, &variablesJSON, &t.IsDefault, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(variablesJSON, &t.Variables)
	return &t, nil
}

func (r *sdkGeneratorRepository) SaveTemplate(ctx context.Context, template *models.SDKTemplate) error {
	query := `
		INSERT INTO sdk_templates (id, language, template_type, name, content, variables, is_default, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (language, template_type, name) DO UPDATE SET content = $5, variables = $6, updated_at = $9
	`

	if template.ID == uuid.Nil {
		template.ID = uuid.New()
	}
	now := time.Now()
	template.CreatedAt = now
	template.UpdatedAt = now

	variablesJSON, _ := json.Marshal(template.Variables)

	_, err := r.db.Pool.Exec(ctx, query,
		template.ID, template.Language, template.TemplateType, template.Name,
		template.Content, variablesJSON, template.IsDefault, template.CreatedAt, template.UpdatedAt)
	return err
}

// Download tracking

func (r *sdkGeneratorRepository) RecordDownload(ctx context.Context, download *models.SDKDownload) error {
	query := `INSERT INTO sdk_downloads (id, generation_id, tenant_id, download_type, ip_address, user_agent, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	if download.ID == uuid.Nil {
		download.ID = uuid.New()
	}
	download.CreatedAt = time.Now()

	_, err := r.db.Pool.Exec(ctx, query,
		download.ID, download.GenerationID, download.TenantID, download.DownloadType,
		download.IPAddress, download.UserAgent, download.CreatedAt)
	return err
}

func (r *sdkGeneratorRepository) GetDownloadCount(ctx context.Context, generationID uuid.UUID) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM sdk_downloads WHERE generation_id = $1`, generationID).Scan(&count)
	return count, err
}
