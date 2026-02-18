package versioning

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service provides versioning operations
type Service struct {
	repo Repository
	wg   sync.WaitGroup
}

// NewService creates a new versioning service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Close waits for all in-flight deprecation/sunset notice goroutines to complete.
func (s *Service) Close() {
	s.wg.Wait()
}

// CreateVersion creates a new version
func (s *Service) CreateVersion(ctx context.Context, tenantID string, req *CreateVersionRequest) (*Version, error) {
	// Parse version string
	sv, err := ParseSemanticVersion(req.Label)
	if err != nil {
		return nil, fmt.Errorf("invalid version label: %w", err)
	}

	version := &Version{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		WebhookID:  req.WebhookID,
		Major:      sv.Major,
		Minor:      sv.Minor,
		Patch:      sv.Patch,
		Label:      req.Label,
		SchemaID:   req.SchemaID,
		Status:     StatusDraft,
		Changelog:  req.Changelog,
		Breaking:   req.Breaking,
		Transforms: req.Transforms,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.SaveVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("save version: %w", err)
	}

	return version, nil
}

// GetVersion retrieves a version
func (s *Service) GetVersion(ctx context.Context, tenantID, versionID string) (*Version, error) {
	return s.repo.GetVersion(ctx, tenantID, versionID)
}

// ListVersions lists versions for a webhook
func (s *Service) ListVersions(ctx context.Context, tenantID, webhookID string) ([]Version, error) {
	return s.repo.ListVersions(ctx, tenantID, webhookID)
}

// PublishVersion publishes a draft version
func (s *Service) PublishVersion(ctx context.Context, tenantID, versionID string) (*Version, error) {
	version, err := s.repo.GetVersion(ctx, tenantID, versionID)
	if err != nil {
		return nil, err
	}

	if version.Status != StatusDraft {
		return nil, fmt.Errorf("version is not in draft status")
	}

	now := time.Now()
	version.Status = StatusPublished
	version.PublishedAt = &now
	version.UpdatedAt = now

	if err := s.repo.SaveVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("publish version: %w", err)
	}

	return version, nil
}

// DeprecateVersion marks a version as deprecated
func (s *Service) DeprecateVersion(ctx context.Context, tenantID, versionID string, req *DeprecateRequest) (*Version, error) {
	version, err := s.repo.GetVersion(ctx, tenantID, versionID)
	if err != nil {
		return nil, err
	}

	if version.Status != StatusPublished {
		return nil, fmt.Errorf("can only deprecate published versions")
	}

	now := time.Now()
	version.Status = StatusDeprecated
	version.DeprecatedAt = &now
	version.UpdatedAt = now

	// Set sunset date
	if req.SunsetDays > 0 {
		sunsetAt := now.AddDate(0, 0, req.SunsetDays)
		version.SunsetAt = &sunsetAt
	}

	// Set replacement version
	if req.ReplacementID != "" {
		version.Replacement = req.ReplacementID
	}

	// Set sunset policy
	if req.Policy != nil {
		version.SunsetPolicy = req.Policy
	}

	if err := s.repo.SaveVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("deprecate version: %w", err)
	}

	// Send deprecation notices to subscribers
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.sendDeprecationNotices(ctx, version)
	}()

	return version, nil
}

// SunsetVersion marks a version as sunset (no longer available)
func (s *Service) SunsetVersion(ctx context.Context, tenantID, versionID string) (*Version, error) {
	version, err := s.repo.GetVersion(ctx, tenantID, versionID)
	if err != nil {
		return nil, err
	}

	version.Status = StatusSunset
	version.UpdatedAt = time.Now()

	if err := s.repo.SaveVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("sunset version: %w", err)
	}

	// Send sunset notices
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.sendSunsetNotices(ctx, version)
	}()

	return version, nil
}

// CheckCompatibility checks compatibility between two versions
func (s *Service) CheckCompatibility(ctx context.Context, tenantID, sourceID, targetID string) (*CompatibilityResult, error) {
	source, err := s.repo.GetVersion(ctx, tenantID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("get source version: %w", err)
	}

	target, err := s.repo.GetVersion(ctx, tenantID, targetID)
	if err != nil {
		return nil, fmt.Errorf("get target version: %w", err)
	}

	// Get schemas
	sourceSchema, err := s.repo.GetSchema(ctx, tenantID, source.SchemaID)
	if err != nil {
		return nil, fmt.Errorf("get source schema: %w", err)
	}

	targetSchema, err := s.repo.GetSchema(ctx, tenantID, target.SchemaID)
	if err != nil {
		return nil, fmt.Errorf("get target schema: %w", err)
	}

	return s.compareSchemas(sourceSchema, targetSchema)
}

// compareSchemas compares two schemas for compatibility
func (s *Service) compareSchemas(source, target *VersionSchema) (*CompatibilityResult, error) {
	result := &CompatibilityResult{
		Compatible: true,
		Direction:  CompatFull,
	}

	// Extract required fields from both schemas
	sourceRequired := extractRequiredFields(source.Definition)
	targetRequired := extractRequiredFields(target.Definition)

	sourceProps := extractProperties(source.Definition)
	targetProps := extractProperties(target.Definition)

	// Check for breaking changes
	for field := range sourceProps {
		if _, exists := targetProps[field]; !exists {
			// Field removed
			result.BreakingChanges = append(result.BreakingChanges, BreakingChange{
				Type:        BreakingFieldRemoved,
				Path:        field,
				Description: fmt.Sprintf("Field '%s' was removed", field),
				Severity:    "high",
			})
			result.Compatible = false
			result.Direction = CompatNone
		}
	}

	// Check for new required fields
	for field := range targetRequired {
		if _, wasRequired := sourceRequired[field]; !wasRequired {
			if _, existed := sourceProps[field]; !existed {
				// New required field
				result.BreakingChanges = append(result.BreakingChanges, BreakingChange{
					Type:        BreakingFieldRequired,
					Path:        field,
					Description: fmt.Sprintf("Field '%s' is now required", field),
					Severity:    "high",
				})
				result.Compatible = false
			}
		}
	}

	// Check for type changes
	for field, sourceType := range sourceProps {
		if targetType, exists := targetProps[field]; exists {
			if sourceType != targetType {
				result.BreakingChanges = append(result.BreakingChanges, BreakingChange{
					Type:        BreakingTypeChanged,
					Path:        field,
					Description: fmt.Sprintf("Field '%s' type changed from '%s' to '%s'", field, sourceType, targetType),
					Severity:    "high",
				})
				result.Compatible = false
			}
		}
	}

	// Generate suggested transforms
	if !result.Compatible {
		result.Transforms = s.suggestTransforms(result.BreakingChanges)
	}

	return result, nil
}

// extractRequiredFields extracts required fields from schema
func extractRequiredFields(schema map[string]any) map[string]bool {
	required := make(map[string]bool)
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			if field, ok := r.(string); ok {
				required[field] = true
			}
		}
	}
	return required
}

// extractProperties extracts property types from schema
func extractProperties(schema map[string]any) map[string]string {
	props := make(map[string]string)
	if properties, ok := schema["properties"].(map[string]any); ok {
		for name, prop := range properties {
			if propMap, ok := prop.(map[string]any); ok {
				if t, ok := propMap["type"].(string); ok {
					props[name] = t
				}
			}
		}
	}
	return props
}

// suggestTransforms suggests transforms for breaking changes
func (s *Service) suggestTransforms(changes []BreakingChange) []Transform {
	var transforms []Transform
	for _, change := range changes {
		switch change.Type {
		case BreakingFieldRemoved:
			transforms = append(transforms, Transform{
				Type:       TransformRemove,
				SourcePath: change.Path,
			})
		case BreakingFieldRequired:
			transforms = append(transforms, Transform{
				Type:       TransformAdd,
				TargetPath: change.Path,
				Value:      nil, // Default value needed
			})
		}
	}
	return transforms
}

// TransformPayload transforms a payload between versions
func (s *Service) TransformPayload(ctx context.Context, payload map[string]any, transforms []Transform) (map[string]any, error) {
	result := make(map[string]any)
	// Deep copy payload
	payloadJSON, _ := json.Marshal(payload)
	json.Unmarshal(payloadJSON, &result)

	for _, t := range transforms {
		switch t.Type {
		case TransformRename:
			if val, exists := result[t.SourcePath]; exists {
				delete(result, t.SourcePath)
				result[t.TargetPath] = val
			}
		case TransformRemove:
			delete(result, t.SourcePath)
		case TransformAdd:
			if _, exists := result[t.TargetPath]; !exists {
				result[t.TargetPath] = t.Value
			}
		case TransformRemap:
			if val, exists := result[t.SourcePath]; exists {
				if strVal, ok := val.(string); ok {
					if mapped, exists := t.Mapping[strVal]; exists {
						result[t.SourcePath] = mapped
					}
				}
			}
		}
	}

	return result, nil
}

// NegotiateVersion negotiates the version to use
func (s *Service) NegotiateVersion(ctx context.Context, tenantID, webhookID, acceptHeader string) (*Version, error) {
	versions, err := s.repo.ListVersions(ctx, tenantID, webhookID)
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions available")
	}

	// Parse accept header
	requestedVersion := parseVersionHeader(acceptHeader)

	// Find matching version
	for _, v := range versions {
		if v.Status == StatusPublished || v.Status == StatusDeprecated {
			if requestedVersion == "" || v.Label == requestedVersion {
				return &v, nil
			}
		}
	}

	// Return latest if no match
	return s.repo.GetLatestVersion(ctx, tenantID, webhookID)
}

// parseVersionHeader parses version from header
func parseVersionHeader(header string) string {
	if header == "" {
		return ""
	}
	// Support formats: "v1.0.0", "1.0.0", "application/vnd.waas.v1+json"
	if strings.Contains(header, "vnd.waas.") {
		re := regexp.MustCompile(`v(\d+)`)
		matches := re.FindStringSubmatch(header)
		if len(matches) > 1 {
			return "v" + matches[1] + ".0.0"
		}
	}
	return strings.TrimPrefix(header, "v")
}

// StartMigration starts a version migration
func (s *Service) StartMigration(ctx context.Context, tenantID string, req *StartMigrationRequest) (*Migration, error) {
	// Get subscriptions to migrate
	subs, err := s.repo.ListSubscriptions(ctx, tenantID, req.FromVersionID)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}

	migration := &Migration{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		WebhookID:   req.WebhookID,
		FromVersion: req.FromVersionID,
		ToVersion:   req.ToVersionID,
		Status:      MigStatusPending,
		Strategy:    req.Strategy,
		Progress: MigrationProgress{
			TotalEndpoints: len(subs),
			CurrentPhase:   "initializing",
		},
		StartedAt: time.Now(),
	}

	if err := s.repo.SaveMigration(ctx, migration); err != nil {
		return nil, fmt.Errorf("save migration: %w", err)
	}

	// Start migration in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.executeMigration(ctx, migration, subs)
	}()

	return migration, nil
}

// executeMigration executes the migration
func (s *Service) executeMigration(ctx context.Context, migration *Migration, subs []VersionSubscription) {
	migration.Status = MigStatusRunning
	migration.Progress.CurrentPhase = "migrating"
	s.repo.SaveMigration(ctx, migration)

	migrated := 0
	failed := 0

	for _, sub := range subs {
		if sub.Pinned {
			continue // Skip pinned subscriptions
		}

		// Update subscription to new version
		sub.VersionID = migration.ToVersion
		sub.Status = SubMigrating
		sub.UpdatedAt = time.Now()

		if err := s.repo.SaveSubscription(ctx, &sub); err != nil {
			failed++
		} else {
			migrated++
		}

		// Update progress
		migration.Progress.MigratedEndpoints = migrated
		migration.Progress.FailedEndpoints = failed
		migration.Progress.Percentage = float64(migrated+failed) / float64(migration.Progress.TotalEndpoints) * 100
		s.repo.SaveMigration(ctx, migration)
	}

	// Complete migration
	now := time.Now()
	migration.CompletedAt = &now
	if failed > 0 {
		migration.Status = MigStatusFailed
		migration.Error = fmt.Sprintf("%d endpoints failed to migrate", failed)
	} else {
		migration.Status = MigStatusCompleted
	}
	migration.Progress.CurrentPhase = "completed"
	s.repo.SaveMigration(ctx, migration)
}

// GetMigration retrieves a migration
func (s *Service) GetMigration(ctx context.Context, migID string) (*Migration, error) {
	return s.repo.GetMigration(ctx, migID)
}

// ListMigrations lists migrations
func (s *Service) ListMigrations(ctx context.Context, tenantID, webhookID string) ([]Migration, error) {
	return s.repo.ListMigrations(ctx, tenantID, webhookID)
}

// sendDeprecationNotices sends deprecation notices
func (s *Service) sendDeprecationNotices(ctx context.Context, version *Version) {
	subs, err := s.repo.ListSubscriptions(ctx, version.TenantID, version.ID)
	if err != nil {
		return
	}

	for _, sub := range subs {
		notice := &DeprecationNotice{
			ID:         uuid.New().String(),
			TenantID:   version.TenantID,
			VersionID:  version.ID,
			EndpointID: sub.EndpointID,
			Type:       NoticeDeprecation,
			Message:    fmt.Sprintf("Version %s has been deprecated", version.Label),
			SentAt:     time.Now(),
		}

		if version.SunsetAt != nil {
			notice.Message += fmt.Sprintf(" and will sunset on %s", version.SunsetAt.Format("2006-01-02"))
		}
		if version.Replacement != "" {
			notice.Message += fmt.Sprintf(". Please migrate to version %s", version.Replacement)
		}

		s.repo.SaveNotice(ctx, notice)
	}
}

// sendSunsetNotices sends sunset notices
func (s *Service) sendSunsetNotices(ctx context.Context, version *Version) {
	subs, err := s.repo.ListSubscriptions(ctx, version.TenantID, version.ID)
	if err != nil {
		return
	}

	for _, sub := range subs {
		notice := &DeprecationNotice{
			ID:         uuid.New().String(),
			TenantID:   version.TenantID,
			VersionID:  version.ID,
			EndpointID: sub.EndpointID,
			Type:       NoticeSunset,
			Message:    fmt.Sprintf("Version %s has been sunset and is no longer available", version.Label),
			SentAt:     time.Now(),
		}

		s.repo.SaveNotice(ctx, notice)
	}
}

// CreateSchema creates a new schema
func (s *Service) CreateSchema(ctx context.Context, tenantID string, req *CreateSchemaRequest) (*VersionSchema, error) {
	schema := &VersionSchema{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Format:      req.Format,
		Definition:  req.Definition,
		Examples:    req.Examples,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.SaveSchema(ctx, schema); err != nil {
		return nil, fmt.Errorf("save schema: %w", err)
	}

	return schema, nil
}

// GetSchema retrieves a schema
func (s *Service) GetSchema(ctx context.Context, tenantID, schemaID string) (*VersionSchema, error) {
	return s.repo.GetSchema(ctx, tenantID, schemaID)
}

// ListSchemas lists schemas
func (s *Service) ListSchemas(ctx context.Context, tenantID string) ([]VersionSchema, error) {
	return s.repo.ListSchemas(ctx, tenantID)
}

// GetPolicy retrieves versioning policy
func (s *Service) GetPolicy(ctx context.Context, tenantID string) (*VersionPolicy, error) {
	return s.repo.GetPolicy(ctx, tenantID)
}

// UpdatePolicy updates versioning policy
func (s *Service) UpdatePolicy(ctx context.Context, tenantID string, req *UpdatePolicyRequest) (*VersionPolicy, error) {
	policy, err := s.repo.GetPolicy(ctx, tenantID)
	if err != nil {
		// Create new policy
		policy = &VersionPolicy{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			Enabled:         true,
			DefaultVersion:  "latest",
			DeprecationDays: 90,
			SunsetDays:      180,
			CreatedAt:       time.Now(),
		}
	}

	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}
	if req.DefaultVersion != "" {
		policy.DefaultVersion = req.DefaultVersion
	}
	if req.RequireVersionHeader != nil {
		policy.RequireVersionHeader = *req.RequireVersionHeader
	}
	if req.AllowDeprecated != nil {
		policy.AllowDeprecated = *req.AllowDeprecated
	}
	if req.AutoUpgrade != nil {
		policy.AutoUpgrade = *req.AutoUpgrade
	}
	if req.DeprecationDays > 0 {
		policy.DeprecationDays = req.DeprecationDays
	}
	if req.SunsetDays > 0 {
		policy.SunsetDays = req.SunsetDays
	}
	if req.NotificationChannels != nil {
		policy.NotificationChannels = req.NotificationChannels
	}
	policy.UpdatedAt = time.Now()

	if err := s.repo.SavePolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("save policy: %w", err)
	}

	return policy, nil
}

// GetVersionMetrics retrieves version usage metrics
func (s *Service) GetVersionMetrics(ctx context.Context, tenantID, versionID string) (*VersionMetrics, error) {
	return s.repo.GetVersionMetrics(ctx, tenantID, versionID)
}

// CompareVersions compares two versions
func (s *Service) CompareVersions(ctx context.Context, tenantID, sourceID, targetID string) (*VersionComparison, error) {
	source, err := s.repo.GetVersion(ctx, tenantID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("get source: %w", err)
	}

	target, err := s.repo.GetVersion(ctx, tenantID, targetID)
	if err != nil {
		return nil, fmt.Errorf("get target: %w", err)
	}

	comparison := &VersionComparison{
		Source: source,
		Target: target,
	}

	// Compare semantic versions
	sv := SemanticVersion{Major: source.Major, Minor: source.Minor, Patch: source.Patch}
	tv := SemanticVersion{Major: target.Major, Minor: target.Minor, Patch: target.Patch}

	cmp := sv.Compare(tv)
	if cmp < 0 {
		comparison.Comparison = "older"
		comparison.CanUpgrade = true
	} else if cmp > 0 {
		comparison.Comparison = "newer"
		comparison.CanDowngrade = !target.Breaking
	} else {
		comparison.Comparison = "same"
	}

	return comparison, nil
}

// ParseSemanticVersion parses a semantic version string
func ParseSemanticVersion(version string) (*SemanticVersion, error) {
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid version format")
	}

	sv := &SemanticVersion{}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version")
	}
	sv.Major = major

	if len(parts) >= 2 {
		minor, err := strconv.Atoi(parts[1])
		if err == nil {
			sv.Minor = minor
		}
	}

	if len(parts) >= 3 {
		patchPart := parts[2]
		// Handle prerelease
		if idx := strings.Index(patchPart, "-"); idx != -1 {
			sv.Prerelease = patchPart[idx+1:]
			patchPart = patchPart[:idx]
		}
		patch, err := strconv.Atoi(patchPart)
		if err == nil {
			sv.Patch = patch
		}
	}

	return sv, nil
}

// Request types

// CreateVersionRequest for creating a version
type CreateVersionRequest struct {
	WebhookID  string      `json:"webhook_id" binding:"required"`
	Label      string      `json:"label" binding:"required"` // e.g., "v1.2.3"
	SchemaID   string      `json:"schema_id"`
	Changelog  string      `json:"changelog"`
	Breaking   bool        `json:"breaking"`
	Transforms []Transform `json:"transforms"`
}

// DeprecateRequest for deprecating a version
type DeprecateRequest struct {
	SunsetDays    int           `json:"sunset_days"`
	ReplacementID string        `json:"replacement_id"`
	Policy        *SunsetPolicy `json:"policy"`
}

// StartMigrationRequest for starting a migration
type StartMigrationRequest struct {
	WebhookID     string            `json:"webhook_id" binding:"required"`
	FromVersionID string            `json:"from_version_id" binding:"required"`
	ToVersionID   string            `json:"to_version_id" binding:"required"`
	Strategy      MigrationStrategy `json:"strategy"`
}

// CreateSchemaRequest for creating a schema
type CreateSchemaRequest struct {
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	Format      SchemaFormat     `json:"format"`
	Definition  map[string]any   `json:"definition" binding:"required"`
	Examples    []map[string]any `json:"examples"`
}

// UpdatePolicyRequest for updating policy
type UpdatePolicyRequest struct {
	Enabled              *bool    `json:"enabled"`
	DefaultVersion       string   `json:"default_version"`
	RequireVersionHeader *bool    `json:"require_version_header"`
	AllowDeprecated      *bool    `json:"allow_deprecated"`
	AutoUpgrade          *bool    `json:"auto_upgrade"`
	DeprecationDays      int      `json:"deprecation_days"`
	SunsetDays           int      `json:"sunset_days"`
	NotificationChannels []string `json:"notification_channels"`
}
