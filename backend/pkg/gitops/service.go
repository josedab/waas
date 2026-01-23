package gitops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Service provides GitOps configuration management functionality
type Service struct {
	repo Repository
}

// NewService creates a new GitOps service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ValidateManifest parses YAML content and validates its structure
func (s *Service) ValidateManifest(ctx context.Context, content string) ([]string, error) {
	_, errors := s.parseYAMLManifest(content)
	if len(errors) > 0 {
		return errors, fmt.Errorf("manifest validation failed with %d error(s)", len(errors))
	}
	return nil, nil
}

// UploadManifest validates, computes a checksum, and stores a new manifest
func (s *Service) UploadManifest(ctx context.Context, tenantID, name, content string) (*ConfigManifest, error) {
	_, errors := s.parseYAMLManifest(content)
	if len(errors) > 0 {
		return nil, fmt.Errorf("manifest validation failed: %s", strings.Join(errors, "; "))
	}

	checksum := s.computeChecksum(content)

	manifest := &ConfigManifest{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Name:      name,
		Version:   1,
		Content:   content,
		Checksum:  checksum,
		Status:    ManifestStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Check for existing manifests with the same name to increment version
	existing, err := s.repo.ListManifests(ctx, tenantID)
	if err == nil {
		for _, m := range existing {
			if m.Name == name && m.Version >= manifest.Version {
				manifest.Version = m.Version + 1
			}
		}
	}

	if err := s.repo.CreateManifest(ctx, manifest); err != nil {
		return nil, fmt.Errorf("failed to store manifest: %w", err)
	}

	return manifest, nil
}

// GetManifest retrieves a manifest by ID
func (s *Service) GetManifest(ctx context.Context, tenantID, manifestID string) (*ConfigManifest, error) {
	return s.repo.GetManifest(ctx, tenantID, manifestID)
}

// ListManifests retrieves all manifests for a tenant
func (s *Service) ListManifests(ctx context.Context, tenantID string) ([]ConfigManifest, error) {
	return s.repo.ListManifests(ctx, tenantID)
}

// PlanApply compares desired state against current state and generates an ApplyPlan
func (s *Service) PlanApply(ctx context.Context, tenantID, manifestID string) (*ApplyPlan, error) {
	manifest, err := s.repo.GetManifest(ctx, tenantID, manifestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	parsed, _ := s.parseYAMLManifest(manifest.Content)

	plan := &ApplyPlan{
		ManifestID: manifestID,
		Resources:  make([]PlannedAction, 0),
	}

	for resourceType, resources := range parsed {
		currentState, err := s.repo.GetCurrentState(ctx, tenantID, resourceType)
		if err != nil {
			return nil, fmt.Errorf("failed to get current state for %s: %w", resourceType, err)
		}

		currentMap := make(map[string]ConfigResource)
		for _, r := range currentState {
			currentMap[r.ResourceID] = r
		}

		desiredMap := make(map[string]bool)
		for _, res := range resources {
			resID, _ := res["id"].(string)
			if resID == "" {
				continue
			}
			desiredMap[resID] = true

			action := PlannedAction{
				ResourceType: resourceType,
				ResourceID:   resID,
			}

			if existing, ok := currentMap[resID]; ok {
				desiredYAML, _ := yaml.Marshal(res)
				if s.diffStates(existing.DesiredState, string(desiredYAML)) {
					action.Action = ActionUpdate
					action.Description = fmt.Sprintf("Update %s %s", resourceType, resID)
					plan.TotalUpdates++
				} else {
					action.Action = ActionNoChange
					action.Description = fmt.Sprintf("No change for %s %s", resourceType, resID)
					plan.TotalNoChange++
				}
			} else {
				action.Action = ActionCreate
				action.Description = fmt.Sprintf("Create %s %s", resourceType, resID)
				plan.TotalCreates++
			}

			plan.Resources = append(plan.Resources, action)
		}

		// Detect deletions: resources in current state but not in desired state
		for _, r := range currentState {
			if !desiredMap[r.ResourceID] {
				action := PlannedAction{
					ResourceType:  resourceType,
					ResourceID:    r.ResourceID,
					Action:        ActionDelete,
					Description:   fmt.Sprintf("Delete %s %s", resourceType, r.ResourceID),
					IsDestructive: true,
				}
				plan.Resources = append(plan.Resources, action)
				plan.TotalDeletes++
				plan.HasDestructive = true
			}
		}
	}

	return plan, nil
}

// ApplyManifest executes the plan and records results
func (s *Service) ApplyManifest(ctx context.Context, tenantID, manifestID string, force bool) (*ApplyResult, error) {
	start := time.Now()

	plan, err := s.PlanApply(ctx, tenantID, manifestID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate apply plan: %w", err)
	}

	if plan.HasDestructive && !force {
		return nil, fmt.Errorf("plan contains destructive actions; use force=true to proceed")
	}

	result := &ApplyResult{
		ManifestID: manifestID,
		Status:     ManifestStatusApplied,
		Errors:     make([]string, 0),
	}

	for _, action := range plan.Resources {
		if action.Action == ActionNoChange {
			continue
		}

		resource := &ConfigResource{
			ID:           uuid.New().String(),
			ManifestID:   manifestID,
			ResourceType: action.ResourceType,
			ResourceID:   action.ResourceID,
			Action:       action.Action,
			DesiredState: action.Description,
			Status:       ResourceStatusApplied,
		}

		if err := s.repo.CreateResource(ctx, resource); err != nil {
			resource.Status = ResourceStatusFailed
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("failed to apply %s %s: %v", action.ResourceType, action.ResourceID, err))
			continue
		}

		result.AppliedCount++
	}

	if result.FailedCount > 0 {
		result.Status = ManifestStatusFailed
	}

	result.Duration = time.Since(start)

	// Update manifest status
	manifest, err := s.repo.GetManifest(ctx, tenantID, manifestID)
	if err == nil {
		now := time.Now()
		manifest.Status = result.Status
		manifest.AppliedAt = &now
		manifest.UpdatedAt = now
		_ = s.repo.UpdateManifest(ctx, manifest)
	}

	return result, nil
}

// RollbackManifest reverts a manifest to its previous state
func (s *Service) RollbackManifest(ctx context.Context, tenantID, manifestID string) (*ApplyResult, error) {
	start := time.Now()

	manifest, err := s.repo.GetManifest(ctx, tenantID, manifestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	if manifest.Status != ManifestStatusApplied && manifest.Status != ManifestStatusFailed {
		return nil, fmt.Errorf("manifest is in %s state and cannot be rolled back", manifest.Status)
	}

	resources, err := s.repo.ListResourcesByManifest(ctx, manifestID)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	result := &ApplyResult{
		ManifestID: manifestID,
		Status:     ManifestStatusRolledBack,
		Errors:     make([]string, 0),
	}

	for _, resource := range resources {
		if resource.PreviousState != "" {
			resource.DesiredState = resource.PreviousState
			resource.Status = ResourceStatusApplied
			if err := s.repo.UpdateResource(ctx, &resource); err != nil {
				result.FailedCount++
				result.Errors = append(result.Errors, fmt.Sprintf("failed to rollback %s %s: %v", resource.ResourceType, resource.ResourceID, err))
				continue
			}
		}
		result.AppliedCount++
	}

	if result.FailedCount > 0 {
		result.Status = ManifestStatusFailed
	}

	result.Duration = time.Since(start)

	manifest.Status = result.Status
	manifest.UpdatedAt = time.Now()
	_ = s.repo.UpdateManifest(ctx, manifest)

	return result, nil
}

// DetectDrift compares applied config against actual state and generates a DriftReport
func (s *Service) DetectDrift(ctx context.Context, tenantID string) (*DriftReport, error) {
	manifests, err := s.repo.ListManifests(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list manifests: %w", err)
	}

	report := &DriftReport{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		DetectedAt: time.Now(),
		Details:    make([]DriftDetail, 0),
		Status:     DriftStatusDetected,
	}

	for _, manifest := range manifests {
		if manifest.Status != ManifestStatusApplied {
			continue
		}

		resources, err := s.repo.ListResourcesByManifest(ctx, manifest.ID)
		if err != nil {
			continue
		}

		for _, resource := range resources {
			report.ResourceCount++

			currentState, err := s.repo.GetCurrentState(ctx, tenantID, resource.ResourceType)
			if err != nil {
				continue
			}

			var actualState string
			for _, current := range currentState {
				if current.ResourceID == resource.ResourceID {
					actualState = current.DesiredState
					break
				}
			}

			if s.diffStates(resource.DesiredState, actualState) {
				report.DriftedCount++
				detail := DriftDetail{
					ResourceType:  resource.ResourceType,
					ResourceID:    resource.ResourceID,
					Field:         "state",
					ExpectedValue: resource.DesiredState,
					ActualValue:   actualState,
					Severity:      s.calculateDriftSeverity(resource.ResourceType),
				}
				report.Details = append(report.Details, detail)
			}
		}
	}

	if err := s.repo.CreateDriftReport(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to store drift report: %w", err)
	}

	return report, nil
}

// GetDriftReport retrieves a drift report by ID
func (s *Service) GetDriftReport(ctx context.Context, tenantID, reportID string) (*DriftReport, error) {
	return s.repo.GetDriftReport(ctx, tenantID, reportID)
}

// ListDriftReports retrieves all drift reports for a tenant
func (s *Service) ListDriftReports(ctx context.Context, tenantID string) ([]DriftReport, error) {
	return s.repo.ListDriftReports(ctx, tenantID)
}

// parseYAMLManifest parses YAML content and returns structured resources grouped by type
func (s *Service) parseYAMLManifest(content string) (map[string][]map[string]interface{}, []string) {
	var errors []string

	if strings.TrimSpace(content) == "" {
		errors = append(errors, "manifest content is empty")
		return nil, errors
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &parsed); err != nil {
		errors = append(errors, fmt.Sprintf("invalid YAML: %v", err))
		return nil, errors
	}

	result := make(map[string][]map[string]interface{})
	validTypes := map[string]bool{
		ResourceTypeEndpoint:  true,
		ResourceTypeTransform: true,
		ResourceTypeContract:  true,
		ResourceTypeRoute:     true,
		ResourceTypeAlert:     true,
	}

	resourcesRaw, ok := parsed["resources"]
	if !ok {
		errors = append(errors, "manifest must contain a 'resources' key")
		return nil, errors
	}

	resourcesList, ok := resourcesRaw.([]interface{})
	if !ok {
		errors = append(errors, "'resources' must be a list")
		return nil, errors
	}

	for i, item := range resourcesList {
		resMap, ok := item.(map[string]interface{})
		if !ok {
			errors = append(errors, fmt.Sprintf("resource at index %d is not a valid object", i))
			continue
		}

		resType, _ := resMap["type"].(string)
		if resType == "" {
			errors = append(errors, fmt.Sprintf("resource at index %d is missing 'type' field", i))
			continue
		}

		if !validTypes[resType] {
			errors = append(errors, fmt.Sprintf("resource at index %d has invalid type '%s'", i, resType))
			continue
		}

		resID, _ := resMap["id"].(string)
		if resID == "" {
			errors = append(errors, fmt.Sprintf("resource at index %d is missing 'id' field", i))
			continue
		}

		result[resType] = append(result[resType], resMap)
	}

	return result, errors
}

// computeChecksum computes a SHA256 checksum of the given content
func (s *Service) computeChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// diffStates compares two state strings and returns true if they differ
func (s *Service) diffStates(current, desired string) bool {
	return strings.TrimSpace(current) != strings.TrimSpace(desired)
}

func (s *Service) calculateDriftSeverity(resourceType string) string {
	switch resourceType {
	case ResourceTypeEndpoint, ResourceTypeRoute:
		return DriftSeverityCritical
	case ResourceTypeContract, ResourceTypeTransform:
		return DriftSeverityWarning
	default:
		return DriftSeverityInfo
	}
}
